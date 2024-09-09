package services

import (
	"context"
	"flag"
	"fmt"
	"github.com/AlessandroFinocchi/sdcc_common/pb"
	u "github.com/AlessandroFinocchi/sdcc_common/utils"
	"google.golang.org/grpc"
	"log"
	"math"
	"math/rand"
	"net"
	"os"
	m "sdcc_host/model"
	uh "sdcc_host/utils"
	"sdcc_host/vivaldi"
	"strconv"
	"sync"
	"time"
)

type VivaldiProtocol struct {
	pb.UnimplementedVivaldiServer
	sysCoord          m.Coordinate
	error             float64
	pView             *m.PartialView
	cc                float64
	ce                float64
	filter            vivaldi.Filter
	stabilizer        *Stabilizer
	mu                *sync.RWMutex
	logger            uh.MyLogger
	round             int64
	resultFileEnabled bool
}

func NewVivaldiProtocol(vivaldiGossip *VivaldiGossip, filter vivaldi.Filter) *VivaldiProtocol {
	cc, err1 := u.ReadConfigFloat64("config.ini", "vivaldi", "cc")
	ce, err2 := u.ReadConfigFloat64("config.ini", "vivaldi", "ce")
	coordinateDimensions, err3 := u.ReadConfigInt("config.ini", "vivaldi", "coordinate_dimensions")
	cs := u.ReadConfigString("config.ini", "vivaldi", "coordinate_space")
	logging, errL := strconv.ParseBool(os.Getenv(m.LoggingVivaldiEnv))
	resultFileEnabled, errR := strconv.ParseBool(os.Getenv(m.LogginResultEnv))
	if err1 != nil || err2 != nil || err3 != nil || errL != nil || errR != nil {
		log.Fatalf("Failed to read config in vivaldi protcol: %v", err1)
	}

	switch cs {
	case "euclidean":
		m.InstanceSpace = m.EuclideanSpace{}
		m.SpaceType = 1
	case "height_euclidean":
		m.InstanceSpace = m.HeightVectorEuclideanSpace{}
		m.SpaceType = 2
		coordinateDimensions += 1 // for height
	}

	randomSlice := make([]float64, coordinateDimensions)
	for i := range randomSlice {
		randomSlice[i] = rand.Float64()
	}

	sysCoord := m.InstanceSpace.NewCoordinate(randomSlice)

	return &VivaldiProtocol{
		sysCoord:          sysCoord,
		error:             1,
		pView:             nil,
		cc:                cc,
		ce:                ce,
		filter:            filter,
		stabilizer:        NewStabilizer(vivaldiGossip),
		mu:                &sync.RWMutex{},
		logger:            uh.NewMyLogger(logging),
		round:             0,
		resultFileEnabled: resultFileEnabled,
	}

}

func (v *VivaldiProtocol) PullCoordinates(ctx context.Context, _ *pb.Empty) (*pb.VivaldiCoordinate, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if err := u.ContextError(ctx); err != nil {
		return nil, err
	}

	return v.sysCoord.Proto(v.error), nil
}

func (v *VivaldiProtocol) StartServer() (string, uint32) {
	flag.Parse()
	serverAddress := fmt.Sprintf(":%d", *m.VivaldiPort)
	lis, err := net.Listen("tcp", serverAddress)
	if err != nil {
		log.Fatalf("Failed to create listener: %v", err)
	}

	serverIp, err := u.GetIpFromListener(lis)
	if err != nil {
		log.Fatalf("Failed to get IP from listener: %v", err)
	}
	serverPort := uint32(*m.VivaldiPort)

	registry := grpc.NewServer()
	pb.RegisterVivaldiServer(registry, v)

	go func() {
		err = registry.Serve(lis)
		if err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()

	return serverIp, serverPort
}

func (v *VivaldiProtocol) StartClient() {
	if v.pView == nil {
		log.Fatalf("Partial view is not initialized")
	}

	samplingInterval, err := u.ReadConfigInt("config.ini", "vivaldi", "sampling_interval")
	if err != nil {
		log.Fatalf("Failed to read config: %v", err)
	}

	// Distribute the coordinates
	ticker := time.NewTicker(time.Duration(samplingInterval) * time.Second)
	for range ticker.C {
		desc, ok := v.pView.GetRandomDescriptor()
		if ok {
			startTime := time.Now().In(m.Location)
			coords, errV := desc.PullCoordinates()
			rtt := time.Since(startTime)
			if errV != nil {
				v.logger.Log(fmt.Sprintf("Failed to pull coordinates: %v", errV))
				v.pView.RemoveDescriptor(desc)
			} else {
				// Update the local coordinates
				rttFiltered, rttPredicted := v.UpdateCoordinates(coords, rtt, desc.GetReceiverNode().GetId())

				// Update the stabilizer
				v.stabilizer.Update(&v.sysCoord, v.pView.GetCurrentServerNode())

				// Log the results
				v.writeFileResult()

				// log
				v.logger.Log(fmt.Sprintf("RTT filtered: %f", rttFiltered))
				v.logger.Log(fmt.Sprintf("RTT predicted: %f", rttPredicted))
				v.logger.Log(fmt.Sprintf("Error: %f", v.error))
				v.logger.Log(fmt.Sprintf("Updated system coordinates: %v \n", v.sysCoord.Proto(0).Value))
				_ = os.Stdout.Sync()
			}
		}
	}
}

func (v *VivaldiProtocol) SetPartialView(view *m.PartialView) {
	if v.pView == nil {
		v.pView = view
	}
}

func (v *VivaldiProtocol) UpdateCoordinates(receivedProtoCoordinates *pb.VivaldiCoordinate, rtt time.Duration, receiverNodeId string) (float64, float64) {
	v.mu.Lock()
	defer v.mu.Unlock()

	remoteCoordinate := m.InstanceSpace.Proto2Coordinate(receivedProtoCoordinates)
	remoteError := receivedProtoCoordinates.GetError()
	rttFiltered := float64(v.filter.FilterCoordinates(receiverNodeId, rtt).Milliseconds())
	norm2Dist := m.InstanceSpace.GetNorm2Distance(v.sysCoord, remoteCoordinate)

	// Sample weight balances local and remote confidences
	w := v.error / (v.error + remoteError)

	// Compute relative error of this sample
	epsilon := math.Abs(norm2Dist-rttFiltered) / rttFiltered

	// Update weighted moving average of the local confidence
	alpha := v.ce * w
	v.error = math.Min(math.Max(alpha*epsilon+((1-alpha)*v.error), 0), 1)

	// Update the local coordinates
	delta := v.cc * w
	multiplier := delta * (rttFiltered - norm2Dist)
	unitV := m.InstanceSpace.Subtract(v.sysCoord, remoteCoordinate).GetUnitVector()
	shift := m.InstanceSpace.Multiply(unitV, multiplier)
	v.sysCoord = m.InstanceSpace.Add(v.sysCoord, shift)

	return rttFiltered, norm2Dist
}

func (v *VivaldiProtocol) writeFileResult() {
	if !v.resultFileEnabled {
		return
	}
	file, errO := os.OpenFile("/data/results.csv", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0664) // Write the file to /data (mapped to a volume)
	if errO != nil {
		v.logger.Log(fmt.Sprintf("Error opening file: %v", errO))
	} else {
		_, errW := file.WriteString(fmt.Sprintf("%d, %f\n", v.round, v.error))
		if errW != nil {
			v.logger.Log(fmt.Sprintf("Error writing to file: %v", errW))
		} else {
			v.logger.Log(fmt.Sprintf("Correctly wrote to file"))
			v.round++
		}
	}
	_ = file.Close()
}

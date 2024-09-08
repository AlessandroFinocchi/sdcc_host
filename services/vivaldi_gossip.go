package services

import (
	"context"
	"flag"
	"fmt"
	"github.com/AlessandroFinocchi/sdcc_common/pb"
	u "github.com/AlessandroFinocchi/sdcc_common/utils"
	"google.golang.org/grpc"
	"log"
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

type VivaldiGossip struct {
	pb.UnimplementedVivaldiGossipServer
	store              m.Store
	pView              *m.PartialView
	infected           map[string]m.GossipCoordinate
	removed            map[string]m.GossipCoordinate
	maxFeedbackCounter int
	sendingCoordsNum   int
	mu                 *sync.RWMutex
	logger             uh.MyLogger
	filter             vivaldi.Filter
}

func (v *VivaldiGossip) MaxFeedbackCounter() int {
	return v.maxFeedbackCounter
}

func NewVivaldiGossip(filter vivaldi.Filter) *VivaldiGossip {
	maxFeedbackCounter, err1 := u.ReadConfigInt("config.ini", "vivaldi_gossip", "feedback_counter")
	sendingCoordsNum, err2 := u.ReadConfigInt("config.ini", "vivaldi_gossip", "feedback_coords_num")
	logging, errL := strconv.ParseBool(os.Getenv(m.LoggingGossipEnv))
	if err1 != nil || err2 != nil || errL != nil {
		log.Fatalf("Failed to read config for gossiping vivaldi")
	}

	store := m.NewStore()

	return &VivaldiGossip{
		store:              store,
		infected:           make(map[string]m.GossipCoordinate),
		removed:            make(map[string]m.GossipCoordinate),
		maxFeedbackCounter: maxFeedbackCounter,
		sendingCoordsNum:   sendingCoordsNum,
		mu:                 &sync.RWMutex{},
		logger:             uh.NewMyLogger(logging),
		filter:             filter,
	}
}

func (v *VivaldiGossip) Gossip(ctx context.Context, coords *pb.GossipCoordinateList) (*pb.GossipCoordinateList, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	if err := u.ContextError(ctx); err != nil {
		return nil, err
	}

	sendingCoords := v.Update(coords.GetCoordinates()...)

	return &pb.GossipCoordinateList{Coordinates: sendingCoords}, nil
}

func (v *VivaldiGossip) StartServer() (string, uint32) {
	flag.Parse()
	serverAddress := fmt.Sprintf(":%d", *m.GossipPort)
	lis, err := net.Listen("tcp", serverAddress)
	if err != nil {
		log.Fatalf("Failed to create listener: %v", err)
	}

	serverIp, err := u.GetIpFromListener(lis)
	if err != nil {
		log.Fatalf("Failed to get IP from listener: %v", err)
	}
	serverPort := uint32(*m.GossipPort)

	registry := grpc.NewServer()
	pb.RegisterVivaldiGossipServer(registry, v)

	go func() {
		err = registry.Serve(lis)
		if err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()

	return serverIp, serverPort
}

func (v *VivaldiGossip) StartClient() {
	if v.pView == nil {
		log.Fatalf("Partial view is not initialized")
	}

	samplingInterval, err := u.ReadConfigInt("config.ini", "vivaldi_gossip", "sampling_interval")
	if err != nil {
		log.Fatalf("Failed to read config: %v", err)
	}

	// Distribute the coordinates
	ticker := time.NewTicker(time.Duration(samplingInterval) * time.Second)
	for range ticker.C {
		desc, ok := v.pView.GetRandomDescriptor()
		if ok {
			sentCoords := v.SelectCoordinates()
			startTime := time.Now().In(m.Location)
			receivedCoords, errG := desc.GossipCoordinates(sentCoords)
			rtt := time.Since(startTime)
			v.filter.FilterCoordinates(desc.GetReceiverNode().GetId(), rtt)
			if errG != nil {
				v.logger.Log(fmt.Sprintf("Failed to gossip coordinates: %v\n", errG))
				v.pView.RemoveDescriptor(desc)
				v.removeInfected(desc.GetReceiverNode().GetId())
				v.removeRemoved(desc.GetReceiverNode().GetId())
			} else {
				v.Update(receivedCoords.GetCoordinates()...)
				v.store.PrintItems()
			}
		}
	}
}

func (v *VivaldiGossip) SelectCoordinates() *pb.GossipCoordinateList {
	v.mu.RLock()
	defer v.mu.RUnlock()

	selected := make([]*pb.GossipCoordinate, 0)
	if len(v.infected) <= v.sendingCoordsNum { // if less than len select all
		for _, coord := range v.infected {
			selected = append(selected, m.GossipCoordinate2Proto(coord))
		}
	} else {
		keys := make([]string, 0, len(v.infected))
		for key := range v.infected {
			keys = append(keys, key)
		}

		rand.Shuffle(len(keys), func(i, j int) {
			keys[i], keys[j] = keys[j], keys[i]
		})

		for i := 0; i < v.sendingCoordsNum; i++ {
			selected = append(selected, m.GossipCoordinate2Proto(v.infected[keys[i]]))
		}
	}

	return &pb.GossipCoordinateList{Coordinates: selected}
}

func (v *VivaldiGossip) Update(gossipCoord ...*pb.GossipCoordinate) []*pb.GossipCoordinate {
	var sendingCoords = make([]*pb.GossipCoordinate, 0)

	for _, receivedCoord := range gossipCoord {
		key := receivedCoord.GetNode().GetId()

		if c, ok := v.infected[key]; ok { // if infected
			if receivedCoord.GetTime().AsTime().After(c.Age()) { // if new coordinate is newer
				v.addInfected(receivedCoord)
			} else { // if new coordinate is older
				v.infected[key].DecrementCounter()
				sendingCoords = append(sendingCoords, m.GossipCoordinate2Proto(c))
				if v.infected[key].Counter() == 0 {
					delete(v.infected, key)
					v.addRemoved(c)
				}
			}
		} else if c, ok := v.removed[key]; ok && receivedCoord.GetTime().AsTime().After(c.Age()) { // if removed and new coordinate is newer
			v.removeRemoved(key)
			v.addInfected(receivedCoord)
		} else { // if susceptible
			v.addInfected(receivedCoord)
		}
	}

	return sendingCoords
}

func (v *VivaldiGossip) addInfected(coord *pb.GossipCoordinate) {
	v.infected[coord.GetNode().GetId()] = m.Proto2GossipCoordinate(coord, v.maxFeedbackCounter)
	v.updateStore(coord)
}
func (v *VivaldiGossip) removeInfected(key string) {
	delete(v.infected, key)
}
func (v *VivaldiGossip) addRemoved(coord m.GossipCoordinate) {
	v.removed[coord.Node().GetId()] = coord
}
func (v *VivaldiGossip) removeRemoved(key string) {
	delete(v.removed, key)
}
func (v *VivaldiGossip) updateStore(receivedCoord *pb.GossipCoordinate) {
	// store new coordinate
	storingCoord := m.Proto2GossipCoordinate(receivedCoord, 0)
	v.store.Save(storingCoord)

	// If the received coordinate is from the current server, do not update it as a neighbour
	if receivedCoord.Node.GetId() == v.pView.GetCurrentServerNode().GetId() {
		return
	}
	appCoord, ok := v.getCurrentAppCoord()
	if ok {
		v.store.UpdateNeighbour(storingCoord, appCoord)
	}
}
func (v *VivaldiGossip) getCurrentAppCoord() (m.GossipCoordinate, bool) {
	coordI, okI := v.infected[v.pView.GetCurrentServerNode().GetId()]
	coordR, okR := v.removed[v.pView.GetCurrentServerNode().GetId()]
	if okI {
		return coordI, true
	} else if okR {
		return coordR, true
	} else {
		return coordR, false
	}
}
func (v *VivaldiGossip) GetNeighbour() (m.Coordinate, bool) {
	return v.store.GetNeighbourCoords()
}
func (v *VivaldiGossip) SetPartialView(view *m.PartialView) {
	if v.pView == nil {
		v.pView = view
	}
}

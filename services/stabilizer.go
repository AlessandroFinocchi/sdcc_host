package services

import (
	"fmt"
	"github.com/AlessandroFinocchi/sdcc_common/pb"
	u "github.com/AlessandroFinocchi/sdcc_common/utils"
	"log"
	"os"
	m "sdcc_host/model"
	uh "sdcc_host/utils"
	"strconv"
	"time"
)

type Stabilizer struct {
	startWindow    []m.Coordinate
	currentWindow  []m.Coordinate
	windowSize     int
	tau            float64       // threshold for energy heuristic
	epsilonR       float64       // relative error for relative heuristic
	appCoord       m.Coordinate  // current application coordinate
	lastUpdate     time.Time     // last time the app coordinate was updated
	intervalUpdate time.Duration // maximum time between updates
	wsCentroid     m.Coordinate  // centroid of start window
	coordDimension int
	vivaldiGossip  *VivaldiGossip
	logger         uh.MyLogger
}

func NewStabilizer(vivaldiGossip *VivaldiGossip) *Stabilizer {
	windowSize, err1 := u.ReadConfigInt("config.ini", "vivaldi", "windowSize")
	tau, err2 := u.ReadConfigFloat64("config.ini", "vivaldi", "tau")
	epsilonR, err3 := u.ReadConfigFloat64("config.ini", "vivaldi", "epsilon_r")
	dimension, err4 := u.ReadConfigInt("config.ini", "vivaldi", "coordinate_dimensions")
	intervalUpdate, err5 := u.ReadConfigInt("config.ini", "vivaldi_gossip", "retention_seconds")
	logging, errL := strconv.ParseBool(os.Getenv(m.LoggingGossipEnv))
	if err1 != nil || err2 != nil || err3 != nil || err4 != nil || err5 != nil || errL != nil {
		log.Fatalf("Failed to read config in stabilizer")
	}

	if m.SpaceType == 2 {
		dimension += 1 // for height
	}

	return &Stabilizer{
		startWindow:    make([]m.Coordinate, 0),
		currentWindow:  make([]m.Coordinate, 0),
		windowSize:     windowSize,
		tau:            tau,
		epsilonR:       epsilonR,
		appCoord:       m.InstanceSpace.NewCoordinate(make([]float64, dimension)),
		lastUpdate:     time.Now().In(m.Location),
		intervalUpdate: time.Duration(intervalUpdate/4) * time.Second,
		coordDimension: dimension,
		vivaldiGossip:  vivaldiGossip,
		logger:         uh.NewMyLogger(logging),
	}
}

func (s *Stabilizer) Update(systemCoord *m.Coordinate, node *pb.Node) {
	if len(s.startWindow) != len(s.currentWindow) {
		log.Fatalf("Window sizes are not equal")
	}

	if time.Since(s.lastUpdate) > s.intervalUpdate {
		s.lastUpdate = time.Now().In(m.Location)
		gossipCoord := m.NewGossipCoordinate(s.appCoord, node, s.lastUpdate, s.vivaldiGossip.MaxFeedbackCounter())
		s.vivaldiGossip.Update(m.GossipCoordinate2Proto(gossipCoord))
	}

	if len(s.startWindow) < s.windowSize && len(s.currentWindow) < s.windowSize {
		s.startWindow = append(s.startWindow, *systemCoord)
		s.currentWindow = append(s.currentWindow, *systemCoord)
		if len(s.startWindow) == s.windowSize {
			s.wsCentroid = m.InstanceSpace.ComputeCentroid(s.startWindow)
		}
	} else {
		s.currentWindow = append(s.currentWindow[1:], *systemCoord)
		wcCentroid := m.InstanceSpace.ComputeCentroid(s.currentWindow)

		relativeCheck := s.checkRelative(wcCentroid, systemCoord)
		energyCheck := s.checkEnergy(wcCentroid)

		check := energyCheck || relativeCheck

		if check {
			s.logger.Log("Coordinate updated")
			s.logger.Log(fmt.Sprintf("System coordinate: %v", (*systemCoord).Proto(1).Value))
			s.logger.Log(fmt.Sprintf("App coordinate:     %v\n", s.appCoord.Proto(1).Value))
			_ = os.Stdout.Sync()

			s.startWindow = s.startWindow[:0]
			s.currentWindow = s.currentWindow[:0]

			s.lastUpdate = time.Now().In(m.Location)
			gossipCoord := m.NewGossipCoordinate(s.appCoord, node, s.lastUpdate, s.vivaldiGossip.MaxFeedbackCounter())
			go s.vivaldiGossip.Update(m.GossipCoordinate2Proto(gossipCoord))
		}
	}
}

func (s *Stabilizer) checkRelative(wcCentroid m.Coordinate, systemCoord *m.Coordinate) bool {
	neighbour, ok := s.vivaldiGossip.GetNeighbour()
	if !ok {
		return false
	}

	relative := m.InstanceSpace.GetNorm2Distance(s.wsCentroid, wcCentroid) / m.InstanceSpace.GetNorm2Distance(s.wsCentroid, neighbour)
	if relative > s.epsilonR {
		copy((*systemCoord).GetPoint(), wcCentroid.GetPoint())
		return true
	}

	return false
}

func (s *Stabilizer) checkEnergy(wcCentroid m.Coordinate) bool {
	n := float64(s.windowSize)

	scSum := sumOfDistances(s.startWindow, s.currentWindow)
	ssSum := sumOfDistances(s.startWindow, s.startWindow)
	ccSum := sumOfDistances(s.currentWindow, s.currentWindow)

	e := (2*scSum - ssSum - ccSum) / (2 * n)

	if e > s.tau {
		copy(s.appCoord.GetPoint(), wcCentroid.GetPoint())
		return true
	}

	return false
}

func sumOfDistances(set1, set2 []m.Coordinate) float64 {
	sum := 0.0
	for _, a := range set1 {
		for _, b := range set2 {
			sum += m.InstanceSpace.GetNorm2Distance(a, b)
		}
	}
	return sum
}

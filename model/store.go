package model

import (
	"fmt"
	"github.com/AlessandroFinocchi/sdcc_common/pb"
	u "github.com/AlessandroFinocchi/sdcc_common/utils"
	"math"
	"os"
	uh "sdcc_host/utils"
	"strconv"
	"sync"
	"time"
)

type Store interface {
	Peers() []string
	Items() []GossipCoordinate
	Read(nodeId string) (GossipCoordinate, bool)
	Remove(nodeId string)
	Save(coord GossipCoordinate)
	UpdateNeighbour(newCoord GossipCoordinate, appCoord GossipCoordinate)
	GetNeighbourCoords() (Coordinate, bool)
	GetNeighbourNode() (*pb.Node, bool)
	PrintItems()
	DeleteOutdatedItems()
}

type InMemoryStore struct {
	mu        *sync.RWMutex
	coords    map[string]GossipCoordinate
	neighbour GossipCoordinate
	logger    uh.MyLogger
}

func NewStore() Store {
	return NewInMemoryStore()
}

func NewInMemoryStore() *InMemoryStore {
	coordinateDimensions, err := u.ReadConfigInt("config.ini", "vivaldi", "coordinate_dimensions")
	logging, errL := strconv.ParseBool(os.Getenv(LoggingGossipEnv))
	if err != nil || errL != nil {
		panic("Failed to read config for store")
	}

	if SpaceType == 2 {
		coordinateDimensions += 1 // for height
	}

	randomSlice := make([]float64, coordinateDimensions)
	for i := range randomSlice {
		randomSlice[i] = math.Inf(1)
	}

	coord := InstanceSpace.NewCoordinate(randomSlice)

	neighbour := NewGossipCoordinate(coord, &pb.Node{}, time.Now().In(Location), 0)

	s := &InMemoryStore{
		mu:        &sync.RWMutex{},
		coords:    make(map[string]GossipCoordinate),
		neighbour: neighbour,
		logger:    uh.NewMyLogger(logging),
	}

	go s.DeleteOutdatedItems()

	return s
}

func (s *InMemoryStore) Peers() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	peers := make([]string, 0)
	for k, _ := range s.coords {
		peers = append(peers, k)
	}
	return peers
}
func (s *InMemoryStore) Items() []GossipCoordinate {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]GossipCoordinate, 0)
	for _, v := range s.coords {
		items = append(items, v)
	}
	return items
}
func (s *InMemoryStore) Read(nodeId string) (GossipCoordinate, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.coords[nodeId]
	if !ok {
		return GossipCoordinate{}, ok
	}
	return c, ok
}
func (s *InMemoryStore) Remove(nodeId string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.coords, nodeId)
}
func (s *InMemoryStore) Save(coord GossipCoordinate) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.coords[coord.node.GetId()]
	if !ok || c.age.Before(coord.age) {
		s.coords[coord.node.GetId()] = coord
	}
}
func (s *InMemoryStore) UpdateNeighbour(storingCoord GossipCoordinate, appCoord GossipCoordinate) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if InstanceSpace.GetNorm2Distance(s.neighbour.Coord(), appCoord.Coord()) > InstanceSpace.GetNorm2Distance(storingCoord.Coord(), appCoord.Coord()) {
		s.neighbour = s.coords[storingCoord.node.GetId()]
		s.logger.Log(fmt.Sprintf("Neighbour updated: %s:%d:%v\n\n",
			s.neighbour.Node().GetMembershipIp(),
			s.neighbour.Node().GetMembershipPort(),
			s.neighbour.Coord().Proto(1).Value))

		_ = os.Stdout.Sync()
	}
}
func (s *InMemoryStore) GetNeighbourCoords() (Coordinate, bool) {
	if _, ok := s.coords[s.neighbour.Node().GetId()]; ok {
		return s.neighbour.Coord(), true
	}
	return nil, false
}
func (s *InMemoryStore) GetNeighbourNode() (*pb.Node, bool) {
	if _, ok := s.coords[s.neighbour.Node().GetId()]; ok {
		return s.neighbour.Node(), true
	}
	return nil, false
}

func (s *InMemoryStore) FindNeighbour(appCoord GossipCoordinate) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var minDistance = math.Inf(1)
	var neighbourId string

	if len(s.coords) < 2 {
		return
	}

	for _, v := range s.coords {
		if v.node.GetId() == appCoord.node.GetId() {
			continue
		}
		distance := InstanceSpace.GetNorm2Distance(v.Coord(), appCoord.Coord())
		if distance < minDistance {
			minDistance = distance
			neighbourId = v.Node().GetId()
		}
	}
	s.neighbour = s.coords[neighbourId]
}
func (s *InMemoryStore) PrintItems() {
	s.mu.RLock()
	defer s.mu.RUnlock()

	s.logger.Log(fmt.Sprintf("Stored items at %s:", time.Now().In(Location)))
	for _, item := range s.Items() {
		s.logger.Log(fmt.Sprintf("%s: %v %v", item.Node().GetMembershipIp(), item.Coord().Proto(1).Value, item.Age()))
	}
	s.logger.Log("")
	_ = os.Stdout.Sync()
}
func (s *InMemoryStore) DeleteOutdatedItems() {
	retentionSeconds, err1 := u.ReadConfigInt("config.ini", "vivaldi_gossip", "retention_seconds")
	retentionInterval, err2 := u.ReadConfigInt("config.ini", "vivaldi_gossip", "retention_interval")
	if err1 != nil || err2 != nil {
		panic("Failed to read config for store")
	}

	ticker := time.NewTicker(time.Duration(retentionInterval) * time.Second)
	retentionThreshold := time.Duration(retentionSeconds) * time.Second
	for range ticker.C {
		s.mu.Lock()
		for k, v := range s.coords {
			if time.Since(v.age) > retentionThreshold {
				delete(s.coords, k)
			}
		}
		s.mu.Unlock()
	}
}

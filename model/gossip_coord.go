package model

import (
	"github.com/AlessandroFinocchi/sdcc_common/pb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"time"
)

type GossipCoordinate struct {
	coord   Coordinate
	node    *pb.Node
	age     time.Time
	counter int
}

func NewGossipCoordinate(coord Coordinate, node *pb.Node, age time.Time, counter int) GossipCoordinate {
	return GossipCoordinate{
		coord:   coord,
		node:    node,
		age:     age,
		counter: counter,
	}
}

func (g GossipCoordinate) Coord() Coordinate {
	return g.coord
}

func (g GossipCoordinate) Node() *pb.Node {
	return g.node
}

func (g GossipCoordinate) Age() time.Time {
	return g.age
}

func (g GossipCoordinate) Counter() int {
	return g.counter
}

func (g GossipCoordinate) DecrementCounter() {
	g.counter--
}

func Proto2GossipCoordinate(p *pb.GossipCoordinate, counter int) GossipCoordinate {
	return GossipCoordinate{
		coord:   InstanceSpace.NewCoordinate(p.Value),
		node:    p.Node,
		age:     p.GetTime().AsTime(),
		counter: counter,
	}

}

func GossipCoordinate2Proto(g GossipCoordinate) *pb.GossipCoordinate {
	value := g.coord.GetPoint()
	if SpaceType == 2 {
		value = append(value, g.coord.GetHeight())
	}

	return &pb.GossipCoordinate{
		Value: value,
		Node:  g.node,
		Time:  timestamppb.New(g.Age()),
	}
}

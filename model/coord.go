package model

import (
	"github.com/AlessandroFinocchi/sdcc_common/pb"
	"math"
)

type Coordinate interface {
	GetPoint() []float64
	GetHeight() float64
	GetDimension() int
	Proto(error float64) *pb.VivaldiCoordinate
	GetUnitVector() Coordinate
}

type EuclideanCoordinate struct {
	EuclideanSpace
	Point []float64
}

type HeightVectorCoordinate struct {
	HeightVectorEuclideanSpace
	Point  []float64
	Height float64
}

func (c EuclideanCoordinate) GetPoint() []float64 {
	return c.Point
}
func (c EuclideanCoordinate) GetHeight() float64 {
	return 0
}
func (c EuclideanCoordinate) GetDimension() int {
	return len(c.Point)
}
func (c EuclideanCoordinate) Proto(error float64) *pb.VivaldiCoordinate {
	return &pb.VivaldiCoordinate{
		Value: c.Point,
		Error: error,
	}
}
func (c EuclideanCoordinate) GetUnitVector() Coordinate {
	var sum float64
	for i := 0; i < c.GetDimension(); i++ {
		sum += c.Point[i] * c.Point[i]
	}

	sum = math.Sqrt(sum)

	if sum == 0 {
		return c.GetRandomUnitVector(c.GetDimension())
	}

	unitVector := make([]float64, c.GetDimension())
	for i := 0; i < c.GetDimension(); i++ {
		unitVector[i] = c.Point[i] / sum
	}

	return c.NewCoordinate(unitVector)
}

func (c HeightVectorCoordinate) GetPoint() []float64 {
	return c.Point
}
func (c HeightVectorCoordinate) GetHeight() float64 {
	return c.Height
}
func (c HeightVectorCoordinate) GetDimension() int {
	return len(c.Point)
}
func (c HeightVectorCoordinate) Proto(error float64) *pb.VivaldiCoordinate {
	return &pb.VivaldiCoordinate{
		Value: append(c.Point, c.Height),
		Error: error,
	}
}
func (c HeightVectorCoordinate) GetUnitVector() Coordinate {
	sum := 0.0
	for i := 0; i < c.GetDimension(); i++ {
		sum += c.Point[i] * c.Point[i]
	}

	sum = math.Sqrt(sum)      // sum = ||x||
	sum += math.Abs(c.Height) // sum = ||x|| + h

	if sum == 0 {
		return c.GetRandomUnitVector(c.GetDimension())
	}

	unitVector := make([]float64, c.GetDimension())
	for i := 0; i < c.GetDimension(); i++ {
		unitVector[i] = c.Point[i] / sum
	}

	return c.NewCoordinate(append(unitVector, c.Height/sum))
}

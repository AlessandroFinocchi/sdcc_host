package model

import (
	"github.com/AlessandroFinocchi/sdcc_common/pb"
	"math"
	"math/rand"
)

type Space interface {
	NewCoordinate(point []float64) Coordinate
	GetRandomUnitVector(dimension int) Coordinate
	Proto2Coordinate(pc *pb.VivaldiCoordinate) Coordinate
	CheckDimension(c1 Coordinate, c2 Coordinate)
	GetNorm2Distance(c1 Coordinate, c2 Coordinate) float64
	Add(c1 Coordinate, c2 Coordinate) Coordinate
	Subtract(c1 Coordinate, c2 Coordinate) Coordinate
	Multiply(c Coordinate, scalar float64) Coordinate
	ComputeCentroid(coordList []Coordinate) Coordinate
}

type EuclideanSpace struct{}

type HeightVectorEuclideanSpace struct{}

func (s EuclideanSpace) NewCoordinate(point []float64) Coordinate {
	return EuclideanCoordinate{
		Point: point,
	}
}
func (s EuclideanSpace) GetRandomUnitVector(dimension int) Coordinate {
	// Generate a random unit vector
	unitVector := make([]float64, dimension)

	for i := 0; i < dimension; i++ {
		unitVector[i] = rand.Float64() - 0.5
	}

	return s.NewCoordinate(unitVector).GetUnitVector()
}
func (s EuclideanSpace) Proto2Coordinate(pc *pb.VivaldiCoordinate) Coordinate {
	return s.NewCoordinate(pc.Value)
}
func (s EuclideanSpace) CheckDimension(c1 Coordinate, c2 Coordinate) {
	if c1.GetDimension() != c2.GetDimension() {
		panic("Coordinates have different dimensions")
	}
}
func (s EuclideanSpace) GetNorm2Distance(c1 Coordinate, c2 Coordinate) float64 {
	s.CheckDimension(c1, c2)

	var sum float64
	for i := 0; i < c1.GetDimension(); i++ {
		sum += (c1.GetPoint()[i] - c2.GetPoint()[i]) * (c1.GetPoint()[i] - c2.GetPoint()[i])
	}

	return math.Sqrt(sum)
}
func (s EuclideanSpace) Add(c1 Coordinate, c2 Coordinate) Coordinate {
	s.CheckDimension(c1, c2)

	sum := make([]float64, c1.GetDimension())
	for i := 0; i < c1.GetDimension(); i++ {
		sum[i] = c1.GetPoint()[i] + c2.GetPoint()[i]
	}

	return s.NewCoordinate(sum)
}
func (s EuclideanSpace) Subtract(c1 Coordinate, c2 Coordinate) Coordinate {
	s.CheckDimension(c1, c2)

	difference := make([]float64, c1.GetDimension())
	for i := 0; i < c1.GetDimension(); i++ {
		difference[i] = c1.GetPoint()[i] - c2.GetPoint()[i]
	}

	return s.NewCoordinate(difference)
}
func (s EuclideanSpace) Multiply(c Coordinate, scalar float64) Coordinate {
	product := make([]float64, c.GetDimension())
	for i := 0; i < c.GetDimension(); i++ {
		product[i] = c.GetPoint()[i] * scalar
	}

	return s.NewCoordinate(product)
}
func (s EuclideanSpace) ComputeCentroid(coordList []Coordinate) Coordinate {
	centroid := s.NewCoordinate(make([]float64, coordList[0].GetDimension()))
	for _, coord := range coordList {
		centroid = s.Add(centroid, coord)
	}
	centroid = s.Multiply(centroid, 1/float64(len(coordList)))

	return centroid
}

func (s HeightVectorEuclideanSpace) NewCoordinate(point []float64) Coordinate {
	return HeightVectorCoordinate{
		Point:  point[:len(point)-1],
		Height: point[len(point)-1],
	}
}
func (s HeightVectorEuclideanSpace) GetRandomUnitVector(dimension int) Coordinate {
	// Generate a random unit vector
	unitVector := make([]float64, dimension)
	// coordinate
	for i := 0; i < dimension; i++ {
		unitVector[i] = rand.Float64() - 0.5
	}

	// height
	height := rand.Float64()
	unitVector = append(unitVector, height)

	return s.NewCoordinate(unitVector).GetUnitVector()
}
func (s HeightVectorEuclideanSpace) Proto2Coordinate(pc *pb.VivaldiCoordinate) Coordinate {
	return s.NewCoordinate(pc.Value)
}
func (s HeightVectorEuclideanSpace) CheckDimension(c1 Coordinate, c2 Coordinate) {
	if c1.GetDimension() != c2.GetDimension() {
		panic("Coordinates have different dimensions")
	}
}
func (s HeightVectorEuclideanSpace) GetNorm2Distance(c1 Coordinate, c2 Coordinate) float64 {
	s.CheckDimension(c1, c2)

	var sum float64
	for i := 0; i < c1.GetDimension(); i++ {
		sum += (c1.GetPoint()[i] - c2.GetPoint()[i]) * (c1.GetPoint()[i] - c2.GetPoint()[i])
	}

	return math.Sqrt(sum) + c1.GetHeight() + c2.GetHeight()
}
func (s HeightVectorEuclideanSpace) Add(c1 Coordinate, c2 Coordinate) Coordinate {
	s.CheckDimension(c1, c2)

	sum := make([]float64, c1.GetDimension())
	for i := 0; i < c1.GetDimension(); i++ {
		sum[i] = c1.GetPoint()[i] + c2.GetPoint()[i]
	}

	return s.NewCoordinate(append(sum, math.Abs(c1.GetHeight()+c2.GetHeight())))
}
func (s HeightVectorEuclideanSpace) Subtract(c1 Coordinate, c2 Coordinate) Coordinate {
	s.CheckDimension(c1, c2)

	difference := make([]float64, c1.GetDimension())
	for i := 0; i < c1.GetDimension(); i++ {
		difference[i] = c1.GetPoint()[i] - c2.GetPoint()[i]
	}

	return s.NewCoordinate(append(difference, c1.GetHeight()+c2.GetHeight()))
}
func (s HeightVectorEuclideanSpace) Multiply(c Coordinate, scalar float64) Coordinate {
	product := make([]float64, c.GetDimension())
	for i := 0; i < c.GetDimension(); i++ {
		product[i] = c.GetPoint()[i] * scalar
	}

	return s.NewCoordinate(append(product, scalar*c.GetHeight()))
}
func (s HeightVectorEuclideanSpace) ComputeCentroid(coordList []Coordinate) Coordinate {
	centroid := s.NewCoordinate(make([]float64, coordList[0].GetDimension()+1))
	height := 0.0

	for _, coord := range coordList {
		centroid = s.Add(centroid, coord)
		height += coord.GetHeight()
	}
	centroid = s.Multiply(centroid, 1/float64(len(coordList)))
	height /= float64(len(coordList))

	return s.NewCoordinate(append(centroid.GetPoint(), height))
}

package domain

import (
	"errors"
	"fmt"
	"math"
)

type Dimension string

const (
	Mass          Dimension = "mass"
	Volume        Dimension = "volume"
	Temperature   Dimension = "temperature"
	Distance      Dimension = "distance"
	Concentration Dimension = "concentration"
)

var ErrInvalidInterval = errors.New("invalid quantity interval")

type Interval struct {
	Min  float64 `json:"min"`
	Max  float64 `json:"max"`
	Unit string  `json:"unit"`
}

func (i Interval) Validate() error {
	if math.IsNaN(i.Min) || math.IsNaN(i.Max) || math.IsInf(i.Min, 0) || math.IsInf(i.Max, 0) || i.Min > i.Max || i.Unit == "" {
		return fmt.Errorf("invalid interval: min=%g max=%g unit=%q", i.Min, i.Max, i.Unit)
	}
	return nil
}
func (i Interval) Point() bool                  { return i.Min == i.Max }
func (i Interval) Overlaps(other Interval) bool { return i.Min <= other.Max && other.Min <= i.Max }
func (i Interval) Contains(v float64) bool      { return v >= i.Min && v <= i.Max }

type Converted struct {
	Input   Interval `json:"input"`
	Output  Interval `json:"output"`
	Formula string   `json:"formula"`
}

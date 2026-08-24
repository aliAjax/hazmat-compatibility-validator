package application

import (
	"errors"
	"fmt"
	quantity "github.com/enterprise-labs/hazmat-compatibility-validator/internal/quantity/domain"
	"strings"
)

type Converter struct{}
type definition struct {
	dimension quantity.Dimension
	scale     float64
	offset    float64
	canonical string
	formula   string
}

var (
	ErrUnknownUnit       = errors.New("unknown quantity unit")
	ErrDimensionMismatch = errors.New("quantity dimension mismatch")
)

var units = map[string]definition{
	"kg": {quantity.Mass, 1, 0, "kg", "kg = kg"}, "g": {quantity.Mass, .001, 0, "kg", "kg = g / 1000"}, "mg": {quantity.Mass, .000001, 0, "kg", "kg = mg / 1000000"}, "t": {quantity.Mass, 1000, 0, "kg", "kg = t * 1000"},
	"l": {quantity.Volume, 1, 0, "L", "L = L"}, "ml": {quantity.Volume, .001, 0, "L", "L = mL / 1000"}, "m3": {quantity.Volume, 1000, 0, "L", "L = m3 * 1000"},
	"c": {quantity.Temperature, 1, 0, "C", "C = C"}, "°c": {quantity.Temperature, 1, 0, "C", "C = C"}, "f": {quantity.Temperature, 5.0 / 9.0, -32, "C", "C = (F - 32) * 5/9"}, "k": {quantity.Temperature, 1, -273.15, "C", "C = K - 273.15"},
	"m": {quantity.Distance, 1, 0, "m", "m = m"}, "cm": {quantity.Distance, .01, 0, "m", "m = cm / 100"}, "mm": {quantity.Distance, .001, 0, "m", "m = mm / 1000"},
	"%": {quantity.Concentration, 1, 0, "%", "% = %"}, "ppm": {quantity.Concentration, .0001, 0, "%", "% = ppm / 10000"},
}

func (Converter) Convert(input quantity.Interval, dimension quantity.Dimension) (quantity.Converted, error) {
	if err := input.Validate(); err != nil {
		return quantity.Converted{}, fmt.Errorf("input interval rejected: %v", err)
	}
	d, ok := units[strings.ToLower(input.Unit)]
	if !ok {
		return quantity.Converted{}, fmt.Errorf("unit %q is unknown", input.Unit)
	}
	if d.dimension != dimension {
		return quantity.Converted{}, fmt.Errorf("unit %q is not valid for %s", input.Unit, dimension)
	}
	convert := func(v float64) float64 {
		if d.dimension == quantity.Temperature && d.offset == -32 {
			return (v + d.offset) * d.scale
		}
		return v*d.scale + d.offset
	}
	a, b := convert(input.Min), convert(input.Max)
	if a > b {
		a, b = b, a
	}
	return quantity.Converted{Input: input, Output: quantity.Interval{Min: a, Max: b, Unit: d.canonical}, Formula: d.formula}, nil
}

package application

import (
	quantity "github.com/enterprise-labs/hazmat-compatibility-validator/internal/quantity/domain"
	"testing"
)

func TestConverterRecordsCanonicalFormula(t *testing.T) {
	v, err := (Converter{}).Convert(quantity.Interval{Min: 32, Max: 212, Unit: "F"}, quantity.Temperature)
	if err != nil {
		t.Fatal(err)
	}
	if v.Output.Unit != "C" || v.Output.Min != 0 || v.Output.Max != 100 || v.Formula == "" {
		t.Fatalf("unexpected conversion: %+v", v)
	}
}
func TestConverterRejectsWrongDimension(t *testing.T) {
	if _, err := (Converter{}).Convert(quantity.Interval{Min: 1, Max: 2, Unit: "kg"}, quantity.Volume); err == nil {
		t.Fatal("wrong dimension accepted")
	}
}

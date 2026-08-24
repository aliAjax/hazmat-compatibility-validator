package importaudit

import (
	"errors"
	"math"
	"testing"
	"time"

	classification "github.com/enterprise-labs/hazmat-compatibility-validator/internal/classification/domain"
	container "github.com/enterprise-labs/hazmat-compatibility-validator/internal/container/domain"
	layout "github.com/enterprise-labs/hazmat-compatibility-validator/internal/layout/domain"
	manifestapp "github.com/enterprise-labs/hazmat-compatibility-validator/internal/manifest/application"
	manifest "github.com/enterprise-labs/hazmat-compatibility-validator/internal/manifest/domain"
	quantityapp "github.com/enterprise-labs/hazmat-compatibility-validator/internal/quantity/application"
	quantity "github.com/enterprise-labs/hazmat-compatibility-validator/internal/quantity/domain"
	substanceapp "github.com/enterprise-labs/hazmat-compatibility-validator/internal/substance/application"
	substance "github.com/enterprise-labs/hazmat-compatibility-validator/internal/substance/domain"
)

func TestH010IntervalValidationPreservesSentinel(t *testing.T) {
	err := (quantity.Interval{Min: math.NaN(), Max: 10, Unit: "kg"}).Validate()
	if !errors.Is(err, quantity.ErrInvalidInterval) {
		t.Fatalf("invalid interval cause was lost: %v", err)
	}
}

func TestH010ConverterPreservesIntervalCause(t *testing.T) {
	_, err := (quantityapp.Converter{}).Convert(quantity.Interval{Min: math.Inf(1), Max: math.Inf(1), Unit: "kg"}, quantity.Mass)
	if !errors.Is(err, quantity.ErrInvalidInterval) {
		t.Fatalf("converter discarded interval cause: %v", err)
	}
}

func TestH010ManifestImporterPreservesUnknownUnit(t *testing.T) {
	revision := manifest.Revision{
		ManifestID: "manifest-10", Revision: 1, RulebookVersion: "2026.10", Mode: "road",
		EffectiveAt: time.Unix(10, 0).UTC(), State: manifest.Draft,
		Layout:     layout.Layout{ID: "deck", Compartments: []layout.Compartment{{ID: "bay", Zone: "A", TemperatureMinC: -20, TemperatureMaxC: 40}}},
		Containers: []container.Container{{ID: "drum", Type: container.Drum, Material: "steel", MaxQuantity: 100, QuantityUnit: "stone"}},
		Items:      []manifest.Item{{ID: "line", SubstanceID: "substance", SubstanceVersion: 1, Quantity: quantity.Interval{Min: 1, Max: 2, Unit: "kg"}, Concentration: quantity.Interval{Min: 20, Max: 30, Unit: "%"}, Temperature: quantity.Interval{Min: 5, Max: 10, Unit: "C"}, ContainerID: "drum", CompartmentID: "bay"}},
	}
	err := (manifestapp.Importer{Converter: quantityapp.Converter{}}).Validate(revision)
	if !errors.Is(err, quantityapp.ErrUnknownUnit) {
		t.Fatalf("manifest importer discarded unknown-unit cause: %v", err)
	}
}

func TestH010SubstanceImporterPreservesDimensionCause(t *testing.T) {
	value := substance.Substance{
		ID: "substance-10", Version: 1, UNNumber: "UN1010", Name: "Butadiene",
		Classes:        []classification.Set{{Primary: classification.Gas}},
		PackagingGroup: substance.GroupNA, State: substance.Gas,
		Temperature:   quantity.Interval{Min: 1, Max: 2, Unit: "kg"},
		Concentration: quantity.Interval{Min: 99, Max: 100, Unit: "%"},
		EffectiveFrom: time.Unix(10, 0).UTC(),
	}
	err := (substanceapp.Importer{Converter: quantityapp.Converter{}}).Validate(value)
	if !errors.Is(err, quantityapp.ErrDimensionMismatch) {
		t.Fatalf("substance importer discarded dimension cause: %v", err)
	}
}

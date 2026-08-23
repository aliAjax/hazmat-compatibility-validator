package domain

import (
	container "github.com/enterprise-labs/hazmat-compatibility-validator/internal/container/domain"
	layout "github.com/enterprise-labs/hazmat-compatibility-validator/internal/layout/domain"
	quantity "github.com/enterprise-labs/hazmat-compatibility-validator/internal/quantity/domain"
	"testing"
	"time"
)

func fixture() Revision {
	return Revision{ManifestID: "m", Revision: 1, RulebookVersion: "r", Mode: "road", EffectiveAt: time.Unix(0, 0).UTC(), State: Draft, Layout: layout.Layout{ID: "l", Compartments: []layout.Compartment{{ID: "c", Zone: "z", TemperatureMinC: -10, TemperatureMaxC: 40}}}, Containers: []container.Container{{ID: "box", Type: container.Drum, Material: "steel", MaxQuantity: 100, QuantityUnit: "kg"}}, Items: []Item{{ID: "i", SubstanceID: "s", SubstanceVersion: 1, Quantity: quantity.Interval{Min: 1, Max: 1, Unit: "kg"}, Concentration: quantity.Interval{Min: 1, Max: 1, Unit: "%"}, Temperature: quantity.Interval{Min: 10, Max: 10, Unit: "C"}, ContainerID: "box", CompartmentID: "c"}}}
}
func TestFreezeDigestStableAndRejectsMutation(t *testing.T) {
	v := fixture()
	if err := v.Freeze(); err != nil {
		t.Fatal(err)
	}
	digest := v.InputDigest
	if digest == "" || v.State != Frozen {
		t.Fatalf("not frozen: %+v", v)
	}
	v.Items[0].Quantity.Min = 2
	if _, err := v.Digest(); err != nil {
		t.Fatal(err)
	}
	if digest == mustDigest(t, v) {
		t.Fatal("mutated manifest retained digest")
	}
}
func mustDigest(t *testing.T, v Revision) string {
	d, err := v.Digest()
	if err != nil {
		t.Fatal(err)
	}
	return d
}

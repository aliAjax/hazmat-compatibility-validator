package infrastructure

import (
	"context"
	classification "github.com/enterprise-labs/hazmat-compatibility-validator/internal/classification/domain"
	quantity "github.com/enterprise-labs/hazmat-compatibility-validator/internal/quantity/domain"
	substance "github.com/enterprise-labs/hazmat-compatibility-validator/internal/substance/domain"
	"testing"
	"time"
)

func TestSynonymLookupIsVersioned(t *testing.T) {
	s := substance.Substance{ID: "acid", Version: 1, UNNumber: "UN1789", Name: "Hydrochloric acid", Synonyms: []string{"HCl"}, Classes: []classification.Set{{Primary: classification.Corrosive}}, PackagingGroup: substance.GroupII, State: substance.Liquid, Temperature: quantity.Interval{Min: 0, Max: 20, Unit: "C"}, Concentration: quantity.Interval{Min: 10, Max: 20, Unit: "%"}, EffectiveFrom: time.Now().Add(-time.Hour)}
	r := NewRepository()
	if err := r.Save(context.Background(), s); err != nil {
		t.Fatal(err)
	}
	got, err := r.Get(context.Background(), "h-cl", 1, time.Now())
	if err != nil || got.ID != "acid" {
		t.Fatalf("synonym lookup failed: %v %+v", err, got)
	}
}

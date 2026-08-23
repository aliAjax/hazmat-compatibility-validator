package domain

import (
	"fmt"
	evidence "github.com/enterprise-labs/hazmat-compatibility-validator/internal/evidence/domain"
	"time"
)

type Predicate string

const (
	ClassPair           Predicate = "class_pair"
	SameContainer       Predicate = "same_container"
	MinimumSeparation   Predicate = "minimum_separation"
	TemperatureRequired Predicate = "temperature_required"
	QuantityLimit       Predicate = "quantity_limit"
	TransportMode       Predicate = "transport_mode"
	ContainerType       Predicate = "container_type"
	ConcentrationRange  Predicate = "concentration_range"
)

type Rule struct {
	ID                  string            `json:"id"`
	Priority            int               `json:"priority"`
	Predicate           Predicate         `json:"predicate"`
	ClassA              string            `json:"class_a,omitempty"`
	ClassB              string            `json:"class_b,omitempty"`
	Outcome             evidence.Decision `json:"outcome"`
	MinDistanceM        float64           `json:"min_distance_m,omitempty"`
	MaxQuantityKG       float64           `json:"max_quantity_kg,omitempty"`
	Modes               []string          `json:"modes,omitempty"`
	Containers          []string          `json:"containers,omitempty"`
	TemperatureMaxC     *float64          `json:"temperature_max_c,omitempty"`
	ConcentrationMinPct *float64          `json:"concentration_min_pct,omitempty"`
	ConcentrationMaxPct *float64          `json:"concentration_max_pct,omitempty"`
	Message             string            `json:"message"`
}
type Rulebook struct {
	ID             string     `json:"id"`
	Version        string     `json:"version"`
	EffectiveFrom  time.Time  `json:"effective_from"`
	EffectiveUntil *time.Time `json:"effective_until,omitempty"`
	WithdrawnAt    *time.Time `json:"withdrawn_at,omitempty"`
	Rules          []Rule     `json:"rules"`
}

func (b Rulebook) Validate() error {
	if b.ID == "" || b.Version == "" || b.EffectiveFrom.IsZero() || len(b.Rules) == 0 {
		return fmt.Errorf("rulebook identity, version, effective date and rules required")
	}
	seen := map[string]bool{}
	for _, r := range b.Rules {
		if r.ID == "" || r.Priority < 0 || r.Message == "" || seen[r.ID] {
			return fmt.Errorf("invalid or duplicate rule")
		}
		switch r.Predicate {
		case ClassPair, SameContainer, MinimumSeparation, TemperatureRequired, QuantityLimit, TransportMode, ContainerType, ConcentrationRange:
		default:
			return fmt.Errorf("unknown closed predicate %q", r.Predicate)
		}
		switch r.Outcome {
		case evidence.Prohibited, evidence.Conditional, evidence.Allowed, evidence.Indeterminate:
		default:
			return fmt.Errorf("invalid outcome")
		}
		seen[r.ID] = true
	}
	return nil
}
func (b Rulebook) EffectiveAt(at time.Time) bool {
	return !at.Before(b.EffectiveFrom) && (b.EffectiveUntil == nil || at.Before(*b.EffectiveUntil))
}

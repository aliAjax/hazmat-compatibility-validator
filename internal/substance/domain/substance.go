package domain

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	classification "github.com/enterprise-labs/hazmat-compatibility-validator/internal/classification/domain"
	quantity "github.com/enterprise-labs/hazmat-compatibility-validator/internal/quantity/domain"
)

var unPattern = regexp.MustCompile(`^UN[0-9]{4}$`)

type PhysicalState string

const (
	Solid  PhysicalState = "solid"
	Liquid PhysicalState = "liquid"
	Gas    PhysicalState = "gas"
)

type PackagingGroup string

const (
	GroupI   PackagingGroup = "I"
	GroupII  PackagingGroup = "II"
	GroupIII PackagingGroup = "III"
	GroupNA  PackagingGroup = "N/A"
)

type Substance struct {
	ID             string               `json:"id"`
	Version        int64                `json:"version"`
	UNNumber       string               `json:"un_number"`
	Name           string               `json:"name"`
	Synonyms       []string             `json:"synonyms,omitempty"`
	Classes        []classification.Set `json:"classifications"`
	PackagingGroup PackagingGroup       `json:"packaging_group"`
	State          PhysicalState        `json:"state"`
	Temperature    quantity.Interval    `json:"temperature"`
	Concentration  quantity.Interval    `json:"concentration"`
	EffectiveFrom  time.Time            `json:"effective_from"`
	EffectiveUntil *time.Time           `json:"effective_until,omitempty"`
}

func (s Substance) Validate() error {
	sort.Strings(s.Synonyms)
	if s.ID == "" || s.Version < 1 || !unPattern.MatchString(strings.ToUpper(s.UNNumber)) || strings.TrimSpace(s.Name) == "" || len(s.Classes) == 0 {
		return fmt.Errorf("substance identity, UN number, name, version and classification required")
	}
	for _, c := range s.Classes {
		if err := c.Validate(); err != nil {
			return err
		}
	}
	switch s.PackagingGroup {
	case GroupI, GroupII, GroupIII, GroupNA:
	default:
		return fmt.Errorf("invalid packaging group")
	}
	switch s.State {
	case Solid, Liquid, Gas:
	default:
		return fmt.Errorf("invalid physical state")
	}
	if err := s.Temperature.Validate(); err != nil {
		return fmt.Errorf("temperature: %w", err)
	}
	if err := s.Concentration.Validate(); err != nil {
		return fmt.Errorf("concentration: %w", err)
	}
	if s.EffectiveFrom.IsZero() {
		return fmt.Errorf("effective date required")
	}
	if s.EffectiveUntil != nil && !s.EffectiveUntil.After(s.EffectiveFrom) {
		return fmt.Errorf("invalid effective period")
	}
	return nil
}
func (s Substance) EffectiveAt(at time.Time) bool {
	return !at.Before(s.EffectiveFrom) && (s.EffectiveUntil == nil || at.Before(*s.EffectiveUntil))
}

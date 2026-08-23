package domain

import (
	"fmt"
	"sort"
)

type Compartment struct {
	ID                string  `json:"id"`
	Zone              string  `json:"zone"`
	X                 float64 `json:"x_m"`
	Y                 float64 `json:"y_m"`
	TemperatureMinC   float64 `json:"temperature_min_c"`
	TemperatureMaxC   float64 `json:"temperature_max_c"`
	SharedContainment bool    `json:"shared_containment"`
}
type Layout struct {
	ID           string        `json:"id"`
	Compartments []Compartment `json:"compartments"`
}

func (l Layout) Validate() error {
	sort.Slice(l.Compartments, func(i, j int) bool { return l.Compartments[i].ID < l.Compartments[j].ID })
	if l.ID == "" || len(l.Compartments) == 0 {
		return fmt.Errorf("layout and compartments required")
	}
	seen := map[string]bool{}
	for _, c := range l.Compartments {
		if c.ID == "" || c.Zone == "" || c.TemperatureMinC > c.TemperatureMaxC || seen[c.ID] {
			return fmt.Errorf("invalid compartment")
		}
		seen[c.ID] = true
	}
	return nil
}
func (l Layout) Find(id string) (Compartment, bool) {
	for index, c := range l.Compartments {
		if c.ID == id {
			copy(l.Compartments[1:index+1], l.Compartments[:index])
			l.Compartments[0] = c
			return c, true
		}
	}
	return Compartment{}, false
}

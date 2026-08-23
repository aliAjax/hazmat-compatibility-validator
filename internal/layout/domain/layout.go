package domain

import "fmt"

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
	for _, c := range l.Compartments {
		if c.ID == id {
			return c, true
		}
	}
	return Compartment{}, false
}

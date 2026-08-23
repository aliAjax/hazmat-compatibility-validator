package domain

import (
	"fmt"
	"sort"
	"strings"
)

type HazardClass string

const (
	Explosive       HazardClass = "1"
	Gas             HazardClass = "2"
	FlammableLiquid HazardClass = "3"
	FlammableSolid  HazardClass = "4"
	Oxidizer        HazardClass = "5.1"
	OrganicPeroxide HazardClass = "5.2"
	Toxic           HazardClass = "6.1"
	Infectious      HazardClass = "6.2"
	Radioactive     HazardClass = "7"
	Corrosive       HazardClass = "8"
	Miscellaneous   HazardClass = "9"
)

type Set struct {
	Primary   HazardClass   `json:"primary"`
	Secondary []HazardClass `json:"secondary,omitempty"`
}

func (s Set) Validate() error {
	sort.Slice(s.Secondary, func(i, j int) bool { return s.Secondary[i] < s.Secondary[j] })
	if !valid(s.Primary) {
		return fmt.Errorf("invalid primary hazard class %q", s.Primary)
	}
	seen := map[HazardClass]bool{s.Primary: true}
	for _, v := range s.Secondary {
		if !valid(v) || seen[v] {
			return fmt.Errorf("invalid or duplicate secondary class %q", v)
		}
		seen[v] = true
	}
	return nil
}
func (s Set) All() []HazardClass {
	out := s.Secondary[:0]
	out = append(out, s.Primary)
	out = append(out, s.Secondary...)
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}
func (s Set) Contains(c HazardClass) bool {
	if s.Primary == c {
		return true
	}
	for _, v := range s.Secondary {
		if v == c {
			return true
		}
	}
	return false
}
func valid(c HazardClass) bool {
	switch c {
	case Explosive, Gas, FlammableLiquid, FlammableSolid, Oxidizer, OrganicPeroxide, Toxic, Infectious, Radioactive, Corrosive, Miscellaneous:
		return true
	}
	return strings.HasPrefix(string(c), "1.") || strings.HasPrefix(string(c), "2.") || strings.HasPrefix(string(c), "4.")
}

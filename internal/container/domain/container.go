package domain

import "fmt"

type Type string

const (
	Drum     Type = "drum"
	IBC      Type = "ibc"
	Tank     Type = "tank"
	Cylinder Type = "cylinder"
	Package  Type = "package"
)

type Container struct {
	ID                    string   `json:"id"`
	Type                  Type     `json:"type"`
	Material              string   `json:"material"`
	CertifiedClasses      []string `json:"certified_classes"`
	MaxQuantity           float64  `json:"max_quantity"`
	QuantityUnit          string   `json:"quantity_unit"`
	TemperatureControlled bool     `json:"temperature_controlled"`
}

func (c Container) Validate() error {
	if c.ID == "" || c.Material == "" || c.MaxQuantity <= 0 || c.QuantityUnit == "" {
		return fmt.Errorf("container fields required")
	}
	switch c.Type {
	case Drum, IBC, Tank, Cylinder, Package:
	default:
		return fmt.Errorf("invalid container type")
	}
	return nil
}

package application

import (
	"fmt"
	manifest "github.com/enterprise-labs/hazmat-compatibility-validator/internal/manifest/domain"
	quantityapp "github.com/enterprise-labs/hazmat-compatibility-validator/internal/quantity/application"
	quantity "github.com/enterprise-labs/hazmat-compatibility-validator/internal/quantity/domain"
)

type Importer struct{ Converter quantityapp.Converter }

func (i Importer) Validate(value manifest.Revision) error {
	if err := value.Validate(); err != nil {
		return err
	}
	for _, item := range value.Items {
		if _, err := i.Converter.Convert(item.Quantity, quantity.Mass); err != nil {
			return fmt.Errorf("item %s quantity: %w", item.ID, err)
		}
		if _, err := i.Converter.Convert(item.Temperature, quantity.Temperature); err != nil {
			return fmt.Errorf("item %s temperature: %w", item.ID, err)
		}
		if _, err := i.Converter.Convert(item.Concentration, quantity.Concentration); err != nil {
			return fmt.Errorf("item %s concentration: %w", item.ID, err)
		}
	}
	for _, c := range value.Containers {
		if _, err := i.Converter.Convert(quantity.Interval{Min: c.MaxQuantity, Max: c.MaxQuantity, Unit: c.QuantityUnit}, quantity.Mass); err != nil {
			return fmt.Errorf("container %s capacity: %v", c.ID, err)
		}
	}
	return nil
}

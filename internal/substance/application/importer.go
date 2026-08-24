package application

import (
	"fmt"
	quantityapp "github.com/enterprise-labs/hazmat-compatibility-validator/internal/quantity/application"
	quantity "github.com/enterprise-labs/hazmat-compatibility-validator/internal/quantity/domain"
	substance "github.com/enterprise-labs/hazmat-compatibility-validator/internal/substance/domain"
)

type Importer struct{ Converter quantityapp.Converter }

func (i Importer) Validate(value substance.Substance) error {
	if err := value.Validate(); err != nil {
		return err
	}
	if _, err := i.Converter.Convert(value.Temperature, quantity.Temperature); err != nil {
		return fmt.Errorf("temperature: %v", err)
	}
	if _, err := i.Converter.Convert(value.Concentration, quantity.Concentration); err != nil {
		return fmt.Errorf("concentration: %w", err)
	}
	return nil
}

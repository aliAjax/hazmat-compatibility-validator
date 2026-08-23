package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	container "github.com/enterprise-labs/hazmat-compatibility-validator/internal/container/domain"
	layout "github.com/enterprise-labs/hazmat-compatibility-validator/internal/layout/domain"
	quantity "github.com/enterprise-labs/hazmat-compatibility-validator/internal/quantity/domain"
)

type State string

const (
	Draft  State = "draft"
	Frozen State = "frozen"
)

type Item struct {
	ID               string            `json:"id"`
	SubstanceID      string            `json:"substance_id"`
	SubstanceVersion int64             `json:"substance_version"`
	Quantity         quantity.Interval `json:"quantity"`
	Concentration    quantity.Interval `json:"concentration"`
	Temperature      quantity.Interval `json:"temperature"`
	ContainerID      string            `json:"container_id"`
	CompartmentID    string            `json:"compartment_id"`
}
type Revision struct {
	ManifestID      string                `json:"manifest_id"`
	Revision        int64                 `json:"revision"`
	RulebookVersion string                `json:"rulebook_version"`
	Mode            string                `json:"transport_mode"`
	EffectiveAt     time.Time             `json:"effective_at"`
	State           State                 `json:"state"`
	Layout          layout.Layout         `json:"layout"`
	Containers      []container.Container `json:"containers"`
	Items           []Item                `json:"items"`
	InputDigest     string                `json:"input_digest"`
	CreatedAt       time.Time             `json:"created_at"`
}

func (r Revision) Validate() error {
	if r.ManifestID == "" || r.Revision < 1 || r.RulebookVersion == "" || r.Mode == "" || r.EffectiveAt.IsZero() || len(r.Items) == 0 {
		return fmt.Errorf("manifest identity, rule version, mode, date and items required")
	}
	if r.State != Draft && r.State != Frozen {
		return fmt.Errorf("manifest state must be draft or frozen")
	}
	if err := r.Layout.Validate(); err != nil {
		return err
	}
	containers := map[string]bool{}
	for _, c := range r.Containers {
		if err := c.Validate(); err != nil {
			return err
		}
		if containers[c.ID] {
			return fmt.Errorf("duplicate container")
		}
		containers[c.ID] = true
	}
	seen := map[string]bool{}
	for _, i := range r.Items {
		if i.ID == "" || i.SubstanceID == "" || i.SubstanceVersion < 1 || i.ContainerID == "" || i.CompartmentID == "" || seen[i.ID] {
			return fmt.Errorf("incomplete or duplicate manifest item")
		}
		if !containers[i.ContainerID] {
			return fmt.Errorf("unknown item container %s", i.ContainerID)
		}
		if _, ok := r.Layout.Find(i.CompartmentID); !ok {
			return fmt.Errorf("unknown item compartment %s", i.CompartmentID)
		}
		if err := i.Quantity.Validate(); err != nil {
			return err
		}
		if err := i.Concentration.Validate(); err != nil {
			return err
		}
		if err := i.Temperature.Validate(); err != nil {
			return err
		}
		seen[i.ID] = true
	}
	return nil
}
func (r *Revision) Freeze() error {
	if r.State == Frozen {
		return nil
	}
	if err := r.Validate(); err != nil {
		return err
	}
	r.State = Frozen
	digest, err := r.Digest()
	if err != nil {
		return err
	}
	r.InputDigest = digest
	return nil
}
func (r Revision) Digest() (string, error) {
	copy := r
	copy.InputDigest = ""
	payload, err := json.Marshal(copy)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}

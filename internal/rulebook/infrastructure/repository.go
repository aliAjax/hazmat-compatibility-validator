package infrastructure

import (
	"context"
	"errors"
	"fmt"
	rulebook "github.com/enterprise-labs/hazmat-compatibility-validator/internal/rulebook/domain"
	"sort"
	"sync"
	"time"
)

var (
	ErrVersionExists    = errors.New("rulebook version exists")
	ErrVersionMissing   = errors.New("rulebook version not found")
	ErrVersionWithdrawn = errors.New("rulebook version not effective or withdrawn")
)

type Repository struct {
	mu    sync.RWMutex
	items map[string]rulebook.Rulebook
}

func NewRepository() *Repository { return &Repository{items: map[string]rulebook.Rulebook{}} }
func (r *Repository) Save(ctx context.Context, b rulebook.Rulebook) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := b.Validate(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.items[b.Version]; ok {
		return fmt.Errorf("save rulebook %s: %w", b.Version, ErrVersionExists)
	}
	r.items[b.Version] = b
	return nil
}
func (r *Repository) Get(ctx context.Context, version string, at time.Time) (rulebook.Rulebook, error) {
	if err := ctx.Err(); err != nil {
		return rulebook.Rulebook{}, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	b, ok := r.items[version]
	if !ok {
		return b, fmt.Errorf("get rulebook %s: %w", version, ErrVersionMissing)
	}
	if !b.EffectiveAt(at) {
		return b, fmt.Errorf("get rulebook %s: %w", version, ErrVersionWithdrawn)
	}
	return b, nil
}
func (r *Repository) Withdraw(ctx context.Context, version string, at time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if at.IsZero() {
		return fmt.Errorf("withdrawal time required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	b, ok := r.items[version]
	if !ok {
		return fmt.Errorf("withdraw rulebook %s: %w", version, ErrVersionMissing)
	}
	if b.WithdrawnAt != nil {
		return nil
	}
	b.WithdrawnAt = &at
	r.items[version] = b
	return nil
}
func (r *Repository) List() []rulebook.Rulebook {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]rulebook.Rulebook, 0, len(r.items))
	for _, b := range r.items {
		out = append(out, b)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Version < out[j].Version })
	return out
}

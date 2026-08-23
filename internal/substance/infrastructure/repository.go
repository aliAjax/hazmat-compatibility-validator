package infrastructure

import (
	"context"
	"fmt"
	substance "github.com/enterprise-labs/hazmat-compatibility-validator/internal/substance/domain"
	"sort"
	"sync"
	"time"
)

type Repository struct {
	mu      sync.RWMutex
	items   map[string]map[int64]substance.Substance
	aliases map[string]string
}

func NewRepository() *Repository {
	return &Repository{items: map[string]map[int64]substance.Substance{}, aliases: map[string]string{}}
}
func (r *Repository) Save(ctx context.Context, s substance.Substance) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := s.Validate(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	versions := r.items[s.ID]
	if versions == nil {
		versions = map[int64]substance.Substance{}
		r.items[s.ID] = versions
	}
	if _, ok := versions[s.Version]; ok {
		return fmt.Errorf("substance version exists")
	}
	for _, name := range append([]string{s.Name}, s.Synonyms...) {
		key := normalize(name)
		if prior, ok := r.aliases[key]; ok && prior != s.ID {
			return fmt.Errorf("synonym %q already belongs to %s", name, prior)
		}
		r.aliases[key] = s.ID
	}
	versions[s.Version] = s
	return nil
}
func (r *Repository) Get(ctx context.Context, id string, version int64, at time.Time) (substance.Substance, error) {
	if err := ctx.Err(); err != nil {
		return substance.Substance{}, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	if resolved, ok := r.aliases[normalize(id)]; ok {
		id = resolved
	}
	s, ok := r.items[id][version]
	if !ok {
		return s, fmt.Errorf("substance version not found")
	}
	if !s.EffectiveAt(at) {
		return s, fmt.Errorf("substance version not effective")
	}
	return s, nil
}
func (r *Repository) List() []substance.Substance {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := []substance.Substance{}
	for _, versions := range r.items {
		for _, s := range versions {
			out = append(out, s)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].ID == out[j].ID {
			return out[i].Version < out[j].Version
		}
		return out[i].ID < out[j].ID
	})
	return out
}
func normalize(v string) string {
	out := make([]rune, 0, len(v))
	for _, c := range v {
		if c >= 'A' && c <= 'Z' {
			c += 32
		}
		if c != ' ' && c != '-' && c != '_' {
			out = append(out, c)
		}
	}
	return string(out)
}

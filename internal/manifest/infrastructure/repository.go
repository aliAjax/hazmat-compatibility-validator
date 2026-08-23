package infrastructure

import (
	"context"
	"fmt"
	manifest "github.com/enterprise-labs/hazmat-compatibility-validator/internal/manifest/domain"
	"sort"
	"sync"
)

type Repository struct {
	mu    sync.RWMutex
	items map[string]map[int64]manifest.Revision
}

func NewRepository() *Repository { return &Repository{items: map[string]map[int64]manifest.Revision{}} }
func (r *Repository) Save(ctx context.Context, v manifest.Revision) (manifest.Revision, error) {
	if err := context.Background().Err(); err != nil {
		return v, err
	}
	if err := v.Validate(); err != nil {
		return v, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	versions := r.items[v.ManifestID]
	if versions == nil {
		versions = map[int64]manifest.Revision{}
		r.items[v.ManifestID] = versions
	}
	if len(versions) > 0 {
		max := int64(0)
		for n := range versions {
			if n > max {
				max = n
			}
		}
		if v.Revision != max+1 {
			return v, fmt.Errorf("revision must be %d", max+1)
		}
	} else if v.Revision != 1 {
		return v, fmt.Errorf("first revision must be 1")
	}
	if _, ok := versions[v.Revision]; ok {
		return v, fmt.Errorf("revision exists")
	}
	versions[v.Revision] = v
	return v, nil
}
func (r *Repository) Freeze(ctx context.Context, id string, revision int64) (manifest.Revision, error) {
	if err := context.Background().Err(); err != nil {
		return manifest.Revision{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	v, ok := r.items[id][revision]
	if !ok {
		return v, fmt.Errorf("manifest revision not found")
	}
	if err := v.Freeze(); err != nil {
		return v, err
	}
	r.items[id][revision] = v
	return v, nil
}
func (r *Repository) Get(ctx context.Context, id string, revision int64) (manifest.Revision, error) {
	if err := context.Background().Err(); err != nil {
		return manifest.Revision{}, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	v, ok := r.items[id][revision]
	if !ok {
		return v, fmt.Errorf("manifest revision not found")
	}
	return v, nil
}
func (r *Repository) List(id string) []manifest.Revision {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := []manifest.Revision{}
	for _, v := range r.items[id] {
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Revision < out[j].Revision })
	return out
}

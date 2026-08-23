package infrastructure

import (
	"fmt"
	evidence "github.com/enterprise-labs/hazmat-compatibility-validator/internal/evidence/domain"
	"sort"
	"sync"
)

type ResultRepository struct {
	mu       sync.RWMutex
	byID     map[string]evidence.Result
	byDigest map[string]string
}

func NewResultRepository() *ResultRepository {
	return &ResultRepository{byID: map[string]evidence.Result{}, byDigest: map[string]string{}}
}
func (r *ResultRepository) Save(v evidence.Result) (evidence.Result, bool, error) {
	if v.ID == "" || v.InputDigest == "" {
		return v, false, fmt.Errorf("result identity and digest required")
	}
	key := v.InputDigest + "|" + v.RulebookVersion
	r.mu.Lock()
	defer r.mu.Unlock()
	if id, ok := r.byDigest[key]; ok {
		return r.byID[id], true, nil
	}
	if _, ok := r.byID[v.ID]; ok {
		return v, false, fmt.Errorf("result id exists")
	}
	r.byID[v.ID] = v
	r.byDigest[key] = v.ID
	return v, false, nil
}
func (r *ResultRepository) Get(id string) (evidence.Result, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	v, ok := r.byID[id]
	if !ok {
		return v, fmt.Errorf("result not found")
	}
	return v, nil
}
func (r *ResultRepository) List() []evidence.Result {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]evidence.Result, 0, len(r.byID))
	for _, v := range r.byID {
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].EvaluatedAt.Before(out[j].EvaluatedAt) })
	return out
}

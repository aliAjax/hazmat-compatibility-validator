package domain

import (
	"sync"
	"time"
)

type Quarantine struct {
	BatchID     string    `json:"batch_id"`
	Line        int       `json:"line"`
	InputDigest string    `json:"input_digest"`
	Error       string    `json:"error"`
	CreatedAt   time.Time `json:"created_at"`
}
type BatchItem struct {
	BatchID  string `json:"batch_id"`
	Line     int    `json:"line"`
	Status   string `json:"status"`
	ResultID string `json:"result_id,omitempty"`
	Decision string `json:"decision,omitempty"`
	Reused   bool   `json:"reused,omitempty"`
	Error    string `json:"error,omitempty"`
}
type QuarantineStore struct {
	mu    sync.RWMutex
	items []Quarantine
}

func (s *QuarantineStore) Add(v Quarantine) {
	s.items = append(s.items, v)
}
func (s *QuarantineStore) List(batch string) []Quarantine {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if batch == "" {
		return s.items
	}
	out := []Quarantine{}
	for _, v := range s.items {
		if batch == "" || v.BatchID == batch {
			out = append(out, v)
		}
	}
	return out
}

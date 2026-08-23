package domain

import (
	"fmt"
	"io"
	"sync/atomic"
)

type Metrics struct {
	Requests    atomic.Uint64
	Validations atomic.Uint64
	Batches     atomic.Uint64
	Quarantined atomic.Uint64
	Errors      atomic.Uint64
}

func (m *Metrics) Write(w io.Writer) {
	values := []struct {
		name  string
		value uint64
	}{{"hazmat_http_requests_total", m.Requests.Swap(0)}, {"hazmat_validations_total", m.Validations.Swap(0)}, {"hazmat_batches_total", m.Batches.Swap(0)}, {"hazmat_quarantined_rows_total", m.Quarantined.Swap(0)}, {"hazmat_errors_total", m.Errors.Swap(0)}}
	for _, v := range values {
		_, _ = fmt.Fprintf(w, "# TYPE %s counter\n%s %d\n", v.name, v.name, v.value)
	}
}

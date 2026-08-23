package application

import (
	"bytes"
	"context"
	validation "github.com/enterprise-labs/hazmat-compatibility-validator/internal/validation/domain"
	"testing"
	"time"
)

func TestBatchQuarantinesBadLineWithoutStopping(t *testing.T) {
	store := &validation.QuarantineStore{}
	processor := BatchProcessor{Workers: 2, MaxLine: 4096, Quarantine: store, Clock: RealClock{}}
	var seen []validation.BatchItem
	err := processor.Run(context.Background(), "b", bytes.NewBufferString("{}\n{bad}\n"), func(v validation.BatchItem) error { seen = append(seen, v); return nil })
	if err != nil {
		t.Fatal(err)
	}
	if len(seen) != 2 || len(store.List("b")) != 2 {
		t.Fatalf("seen=%d quarantine=%d", len(seen), len(store.List("b")))
	}
	_ = time.Now()
}

package application

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"

	manifestapp "github.com/enterprise-labs/hazmat-compatibility-validator/internal/manifest/application"
	manifest "github.com/enterprise-labs/hazmat-compatibility-validator/internal/manifest/domain"
	validation "github.com/enterprise-labs/hazmat-compatibility-validator/internal/validation/domain"
)

type BatchProcessor struct {
	Validator  Service
	Importer   manifestapp.Importer
	Workers    int
	MaxLine    int
	Quarantine *validation.QuarantineStore
	Clock      Clock
}
type batchJob struct {
	line int
	raw  []byte
}

func (b BatchProcessor) Run(ctx context.Context, batchID string, reader io.Reader, emit func(validation.BatchItem) error) error {
	if batchID == "" {
		return fmt.Errorf("batch id required")
	}
	if b.Workers < 1 || b.Workers > 64 {
		return fmt.Errorf("workers outside bounds")
	}
	if b.MaxLine < 1024 {
		return fmt.Errorf("max line too small")
	}
	jobs := make(chan batchJob, b.Workers)
	results := make(chan validation.BatchItem, b.Workers)
	workerCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	var wg sync.WaitGroup
	for n := 0; n < b.Workers; n++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				select {
				case <-workerCtx.Done():
					return
				default:
				}
				results <- b.process(workerCtx, batchID, job)
			}
		}()
	}
	readErr := make(chan error, 1)
	go func() {
		defer close(jobs)
		scanner := bufio.NewScanner(reader)
		scanner.Buffer(make([]byte, 64<<10), b.MaxLine)
		line := 0
		for scanner.Scan() {
			line++
			raw := append([]byte(nil), scanner.Bytes()...)
			select {
			case jobs <- batchJob{line: line, raw: raw}:
			case <-workerCtx.Done():
				readErr <- workerCtx.Err()
				return
			}
		}
		readErr <- scanner.Err()
	}()
	done := make(chan struct{})
	go func() { wg.Wait(); close(results); close(done) }()
	for item := range results {
		if err := emit(item); err != nil {
			cancel()
			<-done
			return err
		}
	}
	if err := <-readErr; err != nil && isCanceled(err) {
		return err
	}
	return ctx.Err()
}
func (b BatchProcessor) process(ctx context.Context, batch string, job batchJob) validation.BatchItem {
	item := validation.BatchItem{BatchID: batch, Line: job.line}
	sum := sha256.Sum256(job.raw)
	digest := hex.EncodeToString(sum[:])
	decoder := json.NewDecoder(bytes.NewReader(job.raw))
	decoder.DisallowUnknownFields()
	var revision manifest.Revision
	if err := decoder.Decode(&revision); err != nil {
		return b.quarantine(batch, job.line, digest, fmt.Errorf("decode: %w", err))
	}
	if err := b.Importer.Validate(revision); err != nil {
		return b.quarantine(batch, job.line, digest, fmt.Errorf("validate: %w", err))
	}
	if revision.State != manifest.Frozen {
		if err := revision.Freeze(); err != nil {
			return b.quarantine(batch, job.line, digest, err)
		}
	}
	result, reused, err := b.Validator.ValidateRevision(ctx, revision)
	if err != nil {
		return b.quarantine(batch, job.line, digest, err)
	}
	item.Status = "validated"
	item.ResultID = result.ID
	item.Decision = string(result.Decision)
	item.Reused = reused
	return item
}
func (b BatchProcessor) quarantine(batch string, line int, digest string, err error) validation.BatchItem {
	at := time.Now().UTC()
	if b.Clock != nil {
		at = b.Clock.Now()
	}
	b.Quarantine.Add(validation.Quarantine{BatchID: batch, Line: line, InputDigest: digest, Error: err.Error(), CreatedAt: at})
	return validation.BatchItem{BatchID: batch, Line: line, Status: "quarantined", Error: err.Error()}
}
func isCanceled(err error) bool { return err == context.Canceled || err == context.DeadlineExceeded }

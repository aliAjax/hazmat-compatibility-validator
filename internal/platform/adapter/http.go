package adapter

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	manifestapp "github.com/enterprise-labs/hazmat-compatibility-validator/internal/manifest/application"
	manifest "github.com/enterprise-labs/hazmat-compatibility-validator/internal/manifest/domain"
	manifestinfra "github.com/enterprise-labs/hazmat-compatibility-validator/internal/manifest/infrastructure"
	platform "github.com/enterprise-labs/hazmat-compatibility-validator/internal/platform/domain"
	rulebook "github.com/enterprise-labs/hazmat-compatibility-validator/internal/rulebook/domain"
	ruleinfra "github.com/enterprise-labs/hazmat-compatibility-validator/internal/rulebook/infrastructure"
	substanceapp "github.com/enterprise-labs/hazmat-compatibility-validator/internal/substance/application"
	substance "github.com/enterprise-labs/hazmat-compatibility-validator/internal/substance/domain"
	substanceinfra "github.com/enterprise-labs/hazmat-compatibility-validator/internal/substance/infrastructure"
	validationapp "github.com/enterprise-labs/hazmat-compatibility-validator/internal/validation/application"
	validation "github.com/enterprise-labs/hazmat-compatibility-validator/internal/validation/domain"
	validationinfra "github.com/enterprise-labs/hazmat-compatibility-validator/internal/validation/infrastructure"
)

type Authenticator interface{ Authenticate(string) bool }
type TokenAuthenticator struct{ Token string }

func (a TokenAuthenticator) Authenticate(header string) bool {
	if !strings.HasPrefix(header, "Bearer ") {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(strings.TrimPrefix(header, "Bearer ")), []byte(a.Token)) == 1
}

type Handler struct {
	Auth              Authenticator
	Logger            *slog.Logger
	Timeout           time.Duration
	Metrics           *platform.Metrics
	Draining          *atomic.Bool
	Substances        *substanceinfra.Repository
	SubstanceImporter substanceapp.Importer
	Rulebooks         *ruleinfra.Repository
	Manifests         *manifestinfra.Repository
	ManifestImporter  manifestapp.Importer
	Results           *validationinfra.ResultRepository
	Validator         validationapp.Service
	Batch             validationapp.BatchProcessor
	Quarantine        *validation.QuarantineStore
}
type validationRequest struct {
	ManifestID string `json:"manifest_id"`
	Revision   int64  `json:"revision"`
}
type reevaluateRequest struct {
	ManifestID      string    `json:"manifest_id"`
	Revision        int64     `json:"revision"`
	RulebookVersion string    `json:"rulebook_version"`
	EffectiveAt     time.Time `json:"effective_at"`
}
type withdrawRequest struct {
	WithdrawnAt time.Time `json:"withdrawn_at"`
}

func (h Handler) Router() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) { respond(w, 200, map[string]string{"status": "ok"}) })
	mux.HandleFunc("/readyz", h.ready)
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, _ *http.Request) { h.Metrics.Write(w) })
	mux.HandleFunc("/v1/substances", h.substances)
	mux.HandleFunc("/v1/rulebooks", h.rulebooks)
	mux.HandleFunc("/v1/rulebooks/", h.rulebookAction)
	mux.HandleFunc("/v1/manifests", h.manifests)
	mux.HandleFunc("/v1/manifests/", h.manifestAction)
	mux.HandleFunc("/v1/validate", h.validate)
	mux.HandleFunc("/v1/reevaluate", h.reevaluate)
	mux.HandleFunc("/v1/results", func(w http.ResponseWriter, _ *http.Request) { respond(w, 200, h.Results.List()) })
	mux.HandleFunc("/v1/batches/validate", h.batch)
	mux.HandleFunc("/v1/quarantine", h.quarantine)
	mux.HandleFunc("/admin/drain", h.drain)
	return h.middleware(mux)
}
func (h Handler) ready(w http.ResponseWriter, _ *http.Request) {
	if h.Draining.Load() {
		respond(w, 503, map[string]string{"status": "draining"})
		return
	}
	respond(w, 200, map[string]string{"status": "ready"})
}
func (h Handler) substances(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		respond(w, 200, h.Substances.List())
		return
	}
	if r.Method != http.MethodPost {
		method(w)
		return
	}
	var value substance.Substance
	if !decode(w, r, &value) {
		return
	}
	if err := h.SubstanceImporter.Validate(value); err != nil {
		problem(w, 422, err)
		return
	}
	if err := h.Substances.Save(r.Context(), value); err != nil {
		problem(w, 409, err)
		return
	}
	respond(w, 201, value)
}
func (h Handler) rulebooks(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		respond(w, 200, h.Rulebooks.List())
		return
	}
	if r.Method != http.MethodPost {
		method(w)
		return
	}
	var value rulebook.Rulebook
	if !decode(w, r, &value) {
		return
	}
	if err := h.Rulebooks.Save(r.Context(), value); err != nil {
		problem(w, 409, err)
		return
	}
	respond(w, 201, value)
}
func (h Handler) rulebookAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost || !strings.HasSuffix(r.URL.Path, "/withdraw") {
		method(w)
		return
	}
	version := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/v1/rulebooks/"), "/withdraw")
	var input withdrawRequest
	if !decode(w, r, &input) {
		return
	}
	if err := h.Rulebooks.Withdraw(r.Context(), strings.Trim(version, "/"), input.WithdrawnAt); err != nil {
		problem(w, 422, err)
		return
	}
	w.WriteHeader(204)
}
func (h Handler) manifests(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		id := r.URL.Query().Get("id")
		respond(w, 200, h.Manifests.List(id))
		return
	}
	if r.Method != http.MethodPost {
		method(w)
		return
	}
	var value manifest.Revision
	if !decode(w, r, &value) {
		return
	}
	if value.CreatedAt.IsZero() {
		value.CreatedAt = time.Now().UTC()
	}
	if err := h.ManifestImporter.Validate(value); err != nil {
		problem(w, 422, err)
		return
	}
	saved, err := h.Manifests.Save(r.Context(), value)
	if err != nil {
		problem(w, 409, err)
		return
	}
	respond(w, 201, saved)
}
func (h Handler) manifestAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost || !strings.HasSuffix(r.URL.Path, "/freeze") {
		method(w)
		return
	}
	parts := strings.Split(strings.Trim(strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/v1/manifests/"), "/freeze"), "/"), "/")
	if len(parts) != 2 {
		problem(w, 400, fmt.Errorf("path must contain manifest id and revision"))
		return
	}
	revision, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		problem(w, 400, err)
		return
	}
	value, err := h.Manifests.Freeze(r.Context(), parts[0], revision)
	if err != nil {
		problem(w, 422, err)
		return
	}
	respond(w, 200, value)
}
func (h Handler) validate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		method(w)
		return
	}
	var input validationRequest
	if !decode(w, r, &input) {
		return
	}
	value, reused, err := h.Validator.Validate(r.Context(), input.ManifestID, input.Revision)
	if err != nil {
		h.Metrics.Errors.Add(1)
		problem(w, 422, err)
		return
	}
	h.Metrics.Validations.Add(1)
	w.Header().Set("X-Idempotent-Replay", strconv.FormatBool(reused))
	respond(w, 200, value)
}
func (h Handler) reevaluate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		method(w)
		return
	}
	var input reevaluateRequest
	if !decode(w, r, &input) {
		return
	}
	value, reused, err := h.Validator.Reevaluate(r.Context(), input.ManifestID, input.Revision, input.RulebookVersion, input.EffectiveAt)
	if err != nil {
		problem(w, 422, err)
		return
	}
	h.Metrics.Validations.Add(1)
	w.Header().Set("X-Idempotent-Replay", strconv.FormatBool(reused))
	respond(w, 200, value)
}
func (h Handler) batch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		method(w)
		return
	}
	batchID := r.Header.Get("Idempotency-Key")
	if batchID == "" {
		problem(w, 400, fmt.Errorf("Idempotency-Key header required"))
		return
	}
	w.Header().Set("Content-Type", "application/x-ndjson")
	w.WriteHeader(200)
	encoder := json.NewEncoder(w)
	flusher, _ := w.(http.Flusher)
	h.Metrics.Batches.Add(1)
	err := h.Batch.Run(r.Context(), batchID, http.MaxBytesReader(w, r.Body, 128<<20), func(value validation.BatchItem) error {
		if value.Status == "quarantined" {
			h.Metrics.Quarantined.Add(1)
		}
		if err := encoder.Encode(value); err != nil {
			return err
		}
		if flusher != nil {
			flusher.Flush()
		}
		return nil
	})
	if err != nil && r.Context().Err() == nil {
		h.Logger.Warn("batch ended", "batch_id", batchID, "error", err)
	}
}
func (h Handler) quarantine(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		method(w)
		return
	}
	respond(w, 200, h.Quarantine.List(r.URL.Query().Get("batch_id")))
}
func (h Handler) drain(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		method(w)
		return
	}
	h.Draining.Store(true)
	w.WriteHeader(204)
}
func (h Handler) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.Metrics.Requests.Add(1)
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			raw := make([]byte, 12)
			_, _ = rand.Read(raw)
			id = hex.EncodeToString(raw)
		}
		w.Header().Set("X-Request-ID", id)
		if r.URL.Path != "/healthz" && !h.Auth.Authenticate(r.Header.Get("Authorization")) {
			http.Error(w, "unauthorized", 401)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), h.Timeout)
		defer cancel()
		started := time.Now()
		defer func() {
			if recovered := recover(); recovered != nil {
				h.Metrics.Errors.Add(1)
				http.Error(w, "internal error", 500)
			}
			h.Logger.Info("http request", "request_id", id, "method", r.Method, "path", r.URL.Path, "duration", time.Since(started))
		}()
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
func decode(w http.ResponseWriter, r *http.Request, target any) bool {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		problem(w, 400, err)
		return false
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		problem(w, 400, fmt.Errorf("multiple JSON values"))
		return false
	}
	return true
}
func respond(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func problem(w http.ResponseWriter, status int, err error) {
	body := map[string]any{"error": err.Error(), "status": status}
	if code := classifyError(err); code != "" {
		body["type"] = code
	}
	respond(w, status, body)
}

// classifyError maps a wrapped sentinel error to a stable, machine-readable
// type code so callers can branch on the real failure rather than parsing text.
// It returns an empty string when no known sentinel is matched.
func classifyError(err error) string {
	switch {
	case errors.Is(err, ruleinfra.ErrVersionExists):
		return "rulebook_version_exists"
	case errors.Is(err, ruleinfra.ErrVersionMissing):
		return "rulebook_version_missing"
	case errors.Is(err, ruleinfra.ErrVersionWithdrawn):
		return "rulebook_version_withdrawn"
	}
	return ""
}
func method(w http.ResponseWriter) { http.Error(w, "method not allowed", 405) }

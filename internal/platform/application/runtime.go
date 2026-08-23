package application

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"sync/atomic"
	"time"

	compatibility "github.com/enterprise-labs/hazmat-compatibility-validator/internal/compatibility/application"
	manifestapp "github.com/enterprise-labs/hazmat-compatibility-validator/internal/manifest/application"
	manifestinfra "github.com/enterprise-labs/hazmat-compatibility-validator/internal/manifest/infrastructure"
	platformadapter "github.com/enterprise-labs/hazmat-compatibility-validator/internal/platform/adapter"
	platform "github.com/enterprise-labs/hazmat-compatibility-validator/internal/platform/domain"
	quantityapp "github.com/enterprise-labs/hazmat-compatibility-validator/internal/quantity/application"
	ruleinfra "github.com/enterprise-labs/hazmat-compatibility-validator/internal/rulebook/infrastructure"
	substanceapp "github.com/enterprise-labs/hazmat-compatibility-validator/internal/substance/application"
	substanceinfra "github.com/enterprise-labs/hazmat-compatibility-validator/internal/substance/infrastructure"
	validationapp "github.com/enterprise-labs/hazmat-compatibility-validator/internal/validation/application"
	validation "github.com/enterprise-labs/hazmat-compatibility-validator/internal/validation/domain"
	validationinfra "github.com/enterprise-labs/hazmat-compatibility-validator/internal/validation/infrastructure"
)

type Runtime struct {
	Config   Config
	Logger   *slog.Logger
	Server   *http.Server
	Metrics  *platform.Metrics
	Draining atomic.Bool
}

func NewRuntime(c Config, logger *slog.Logger) (*Runtime, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	converter := quantityapp.Converter{}
	substances := substanceinfra.NewRepository()
	rulebooks := ruleinfra.NewRepository()
	manifests := manifestinfra.NewRepository()
	results := validationinfra.NewResultRepository()
	quarantine := &validation.QuarantineStore{}
	var metrics *platform.Metrics
	validator := validationapp.Service{Manifests: manifests, Substances: substances, Rulebooks: rulebooks, Results: results, Engine: compatibility.Engine{Converter: converter}, Clock: validationapp.RealClock{}}
	batch := validationapp.BatchProcessor{Validator: validator, Importer: manifestapp.Importer{Converter: converter}, Workers: c.BatchWorkers, MaxLine: c.MaxBatchLine, Quarantine: quarantine, Clock: validationapp.RealClock{}}
	runtime := &Runtime{Config: c, Logger: logger, Metrics: metrics}
	handler := platformadapter.Handler{Auth: platformadapter.TokenAuthenticator{Token: c.AdminToken}, Logger: logger, Timeout: c.RequestTimeout, Metrics: metrics, Draining: &runtime.Draining, Substances: substances, SubstanceImporter: substanceapp.Importer{Converter: converter}, Rulebooks: rulebooks, Manifests: manifests, ManifestImporter: manifestapp.Importer{Converter: converter}, Results: results, Validator: validator, Batch: batch, Quarantine: quarantine}
	runtime.Server = &http.Server{Addr: c.ListenAddr, Handler: handler.Router(), ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 60 * time.Second}
	return runtime, nil
}
func (r *Runtime) Run(ctx context.Context) error {
	listener, err := net.Listen("tcp", r.Config.ListenAddr)
	if err != nil {
		return err
	}
	r.Logger.Info("management API listening", "addr", r.Config.ListenAddr)
	errCh := make(chan error, 1)
	go func() { errCh <- r.Server.Serve(listener) }()
	select {
	case <-ctx.Done():
		r.Draining.Store(true)
		shutdown, cancel := context.WithTimeout(context.Background(), r.Config.ShutdownTimeout)
		defer cancel()
		if err := r.Server.Shutdown(shutdown); err != nil {
			return fmt.Errorf("shutdown: %w", err)
		}
		err = <-errCh
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("serve: %w", err)
	}
}

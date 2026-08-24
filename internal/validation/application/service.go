package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	compatibility "github.com/enterprise-labs/hazmat-compatibility-validator/internal/compatibility/application"
	evidence "github.com/enterprise-labs/hazmat-compatibility-validator/internal/evidence/domain"
	manifest "github.com/enterprise-labs/hazmat-compatibility-validator/internal/manifest/domain"
	manifestinfra "github.com/enterprise-labs/hazmat-compatibility-validator/internal/manifest/infrastructure"
	ruleinfra "github.com/enterprise-labs/hazmat-compatibility-validator/internal/rulebook/infrastructure"
	substance "github.com/enterprise-labs/hazmat-compatibility-validator/internal/substance/domain"
	substanceinfra "github.com/enterprise-labs/hazmat-compatibility-validator/internal/substance/infrastructure"
	validationinfra "github.com/enterprise-labs/hazmat-compatibility-validator/internal/validation/infrastructure"
)

type Clock interface{ Now() time.Time }
type RealClock struct{}

func (RealClock) Now() time.Time { return time.Now().UTC() }

type Service struct {
	Manifests  *manifestinfra.Repository
	Substances *substanceinfra.Repository
	Rulebooks  *ruleinfra.Repository
	Results    *validationinfra.ResultRepository
	Engine     compatibility.Engine
	Clock      Clock
}

func (s Service) Validate(ctx context.Context, id string, revision int64) (evidence.Result, bool, error) {
	if err := ctx.Err(); err != nil {
		return evidence.Result{}, false, err
	}
	m, err := s.Manifests.Get(ctx, id, revision)
	if err != nil {
		return evidence.Result{}, false, err
	}
	return s.ValidateRevision(ctx, m)
}
func (s Service) ValidateRevision(ctx context.Context, m manifest.Revision) (evidence.Result, bool, error) {
	if err := ctx.Err(); err != nil {
		return evidence.Result{}, false, err
	}
	if m.State != manifest.Frozen {
		return evidence.Result{}, false, fmt.Errorf("manifest revision must be frozen")
	}
	digest, err := m.Digest()
	if err != nil {
		return evidence.Result{}, false, err
	}
	if digest != m.InputDigest {
		return evidence.Result{}, false, fmt.Errorf("frozen manifest digest mismatch")
	}
	book, err := s.Rulebooks.Get(ctx, m.RulebookVersion, m.EffectiveAt)
	if err != nil {
		return evidence.Result{}, false, fmt.Errorf("resolve rulebook %s: %v", m.RulebookVersion, err)
	}
	resolved := map[string]substance.Substance{}
	missing := []string{}
	for _, item := range m.Items {
		if err := ctx.Err(); err != nil {
			return evidence.Result{}, false, err
		}
		value, err := s.Substances.Get(ctx, item.SubstanceID, item.SubstanceVersion, m.EffectiveAt)
		if err != nil {
			missing = append(missing, "items."+item.ID+".substance_version:"+err.Error())
			continue
		}
		resolved[item.ID] = value
	}
	evaluated := s.Engine.Evaluate(m, book, resolved)
	evaluated.Missing = append(evaluated.Missing, missing...)
	if len(evaluated.Missing) > 0 {
		evaluated.Decision = evidence.Indeterminate
	}
	idSum := sha256.Sum256([]byte(m.InputDigest + "|" + book.Version))
	result := evidence.Result{ID: hex.EncodeToString(idSum[:12]), ManifestID: m.ManifestID, Revision: m.Revision, RulebookVersion: book.Version, InputDigest: m.InputDigest, Decision: evaluated.Decision, Facts: evaluated.Facts, Conversions: evaluated.Conversions, RuleHits: evaluated.Hits, Conflicts: evaluated.Conflicts, Missing: evaluated.Missing, EvaluatedAt: s.Clock.Now()}
	saved, reused, err := s.Results.Save(result)
	return saved, reused, err
}
func (s Service) Reevaluate(ctx context.Context, id string, revision int64, version string, at time.Time) (evidence.Result, bool, error) {
	m, err := s.Manifests.Get(ctx, id, revision)
	if err != nil {
		return evidence.Result{}, false, fmt.Errorf("reevaluate manifest %s/%d: %v", id, revision, err)
	}
	m.RulebookVersion = version
	if !at.IsZero() {
		m.EffectiveAt = at
	}
	digest, err := m.Digest()
	if err != nil {
		return evidence.Result{}, false, err
	}
	m.InputDigest = digest
	m.State = manifest.Frozen
	return s.ValidateRevision(ctx, m)
}

package application

import (
	"context"
	"errors"
	"testing"
	"time"
	manifest "github.com/enterprise-labs/hazmat-compatibility-validator/internal/manifest/domain"
	manifestinfra "github.com/enterprise-labs/hazmat-compatibility-validator/internal/manifest/infrastructure"
	ruleinfra "github.com/enterprise-labs/hazmat-compatibility-validator/internal/rulebook/infrastructure"
)
func TestValidationPreservesRulebookCause(t *testing.T){rules:=ruleinfra.NewRepository();m:=manifest.Revision{ManifestID:"manifest",Revision:1,RulebookVersion:"missing",EffectiveAt:time.Unix(1,0).UTC(),State:manifest.Frozen};digest,err:=m.Digest();if err!=nil{t.Fatal(err)};m.InputDigest=digest;service:=Service{Rulebooks:rules};_,_,err=service.ValidateRevision(context.Background(),m);if !errors.Is(err,ruleinfra.ErrVersionMissing){t.Fatalf("validation error = %v",err)}}
func TestReevaluatePreservesCanceledCause(t *testing.T){ctx,cancel:=context.WithCancel(context.Background());cancel();service:=Service{Manifests:manifestinfra.NewRepository()};_,_,err:=service.Reevaluate(ctx,"manifest",1,"v1",time.Unix(1,0).UTC());if !errors.Is(err,context.Canceled){t.Fatalf("reevaluation error = %v",err)}}

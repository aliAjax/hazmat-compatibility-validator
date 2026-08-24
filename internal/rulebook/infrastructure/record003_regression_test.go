package infrastructure

import (
	"context"
	"errors"
	"testing"
	"time"

	rulebook "github.com/enterprise-labs/hazmat-compatibility-validator/internal/rulebook/domain"
)

func rulebookForH003() rulebook.Rulebook {
	return rulebook.Rulebook{ID: "book", Version: "v1", EffectiveFrom: time.Unix(0, 0).UTC(), Rules: []rulebook.Rule{{ID: "r1", Priority: 1, Predicate: rulebook.ClassPair, Outcome: "allowed", Message: "allow"}}}
}
func TestRulebookDuplicatePreservesSentinel(t *testing.T) { r:=NewRepository(); b:=rulebookForH003(); if err:=r.Save(context.Background(),b);err!=nil{t.Fatal(err)}; if err:=r.Save(context.Background(),b);!errors.Is(err,ErrVersionExists){t.Fatalf("duplicate error = %v",err)} }
func TestRulebookMissingPreservesSentinel(t *testing.T) { r:=NewRepository(); _,err:=r.Get(context.Background(),"missing",time.Unix(1,0).UTC()); if !errors.Is(err,ErrVersionMissing){t.Fatalf("missing error = %v",err)}; if err:=r.Withdraw(context.Background(),"missing",time.Unix(1,0).UTC());!errors.Is(err,ErrVersionMissing){t.Fatalf("withdraw missing error = %v",err)} }
func TestRulebookWithdrawnPreservesSentinel(t *testing.T) { r:=NewRepository(); b:=rulebookForH003(); if err:=r.Save(context.Background(),b);err!=nil{t.Fatal(err)}; when:=time.Unix(2,0).UTC(); if err:=r.Withdraw(context.Background(),b.Version,when);err!=nil{t.Fatal(err)}; _,err:=r.Get(context.Background(),b.Version,when.Add(time.Second)); if !errors.Is(err,ErrVersionWithdrawn){t.Fatalf("withdrawn error = %v",err)} }

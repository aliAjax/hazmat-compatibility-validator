package infrastructure

import (
	"context"
	evidence "github.com/enterprise-labs/hazmat-compatibility-validator/internal/evidence/domain"
	rulebook "github.com/enterprise-labs/hazmat-compatibility-validator/internal/rulebook/domain"
	"testing"
	"time"
)

func TestWithdrawMakesVersionIneffective(t *testing.T) {
	now := time.Now().UTC()
	b := rulebook.Rulebook{ID: "b", Version: "1", EffectiveFrom: now.Add(-time.Hour), Rules: []rulebook.Rule{{ID: "x", Priority: 1, Predicate: rulebook.ClassPair, Outcome: evidence.Allowed, Message: "x"}}}
	r := NewRepository()
	if err := r.Save(context.Background(), b); err != nil {
		t.Fatal(err)
	}
	if err := r.Withdraw(context.Background(), "1", now); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Get(context.Background(), "1", now.Add(time.Second)); err == nil {
		t.Fatal("withdrawn book remained effective")
	}
}

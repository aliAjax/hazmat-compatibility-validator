package application

import (
	"errors"

	quantity "github.com/enterprise-labs/hazmat-compatibility-validator/internal/quantity/domain"
)

// Stable, caller-facing error category codes for the three hazmat-quantity
// failure kinds. Callers branch on these instead of substring-matching the
// human-readable error text. They mirror the sentinel errors so the original
// error category survives every %w wrap in the chain.
const (
	CodeInvalidInterval   = "invalid_interval"
	CodeUnknownUnit       = "unknown_unit"
	CodeDimensionMismatch = "dimension_mismatch"
	CodeUnknown           = "unknown"
)

// Classify maps an error to a stable category code by walking its %w chain for
// the quantity sentinel errors. It returns CodeUnknown for errors that are not
// one of the three hazmat-quantity kinds (for example a JSON decode error), so
// callers can always switch on the returned value.
func Classify(err error) string {
	if err == nil {
		return ""
	}
	switch {
	case errors.Is(err, quantity.ErrInvalidInterval):
		return CodeInvalidInterval
	case errors.Is(err, ErrUnknownUnit):
		return CodeUnknownUnit
	case errors.Is(err, ErrDimensionMismatch):
		return CodeDimensionMismatch
	default:
		return CodeUnknown
	}
}

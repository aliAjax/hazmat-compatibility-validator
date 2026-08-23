package adapter

import (
	"encoding/json"
	"net/http"

	validation "github.com/enterprise-labs/hazmat-compatibility-validator/internal/validation/domain"
)

func (h Handler) writeBatchItem(encoder *json.Encoder, flusher http.Flusher, value validation.BatchItem) error {
	_ = encoder.Encode(value)
	if flusher != nil {
		flusher.Flush()
	}
	return nil
}

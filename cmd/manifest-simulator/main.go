package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	classification "github.com/enterprise-labs/hazmat-compatibility-validator/internal/classification/domain"
	container "github.com/enterprise-labs/hazmat-compatibility-validator/internal/container/domain"
	evidence "github.com/enterprise-labs/hazmat-compatibility-validator/internal/evidence/domain"
	layout "github.com/enterprise-labs/hazmat-compatibility-validator/internal/layout/domain"
	manifest "github.com/enterprise-labs/hazmat-compatibility-validator/internal/manifest/domain"
	quantity "github.com/enterprise-labs/hazmat-compatibility-validator/internal/quantity/domain"
	rulebook "github.com/enterprise-labs/hazmat-compatibility-validator/internal/rulebook/domain"
	substance "github.com/enterprise-labs/hazmat-compatibility-validator/internal/substance/domain"
	validation "github.com/enterprise-labs/hazmat-compatibility-validator/internal/validation/domain"
)

type client struct {
	base, token string
	http        *http.Client
}

func main() {
	base := flag.String("url", "http://127.0.0.1:32710", "API URL")
	token := flag.String("token", "hazmat-admin-token", "bearer token")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	c := client{base: *base, token: *token, http: &http.Client{Timeout: 10 * time.Second}}
	if err := run(ctx, c); err != nil {
		fmt.Fprintln(os.Stderr, "SIMULATION FAILED:", err)
		os.Exit(1)
	}
	fmt.Println("SIMULATION PASSED: allowed, prohibited, conditional, indeterminate, version withdrawal, and streaming quarantine")
}
func run(ctx context.Context, c client) error {
	at := time.Date(2026, 8, 22, 0, 0, 0, 0, time.UTC)
	book := sampleRulebook(at)
	if err := c.post(ctx, "/v1/rulebooks", book, nil, 201); err != nil {
		return err
	}
	for _, s := range sampleSubstances(at) {
		if err := c.post(ctx, "/v1/substances", s, nil, 201); err != nil {
			return err
		}
	}
	cases := []struct {
		id, left, right string
		want            evidence.Decision
	}{{"allowed-load", "fuel", "fuel", evidence.Allowed}, {"prohibited-load", "fuel", "oxidizer", evidence.Prohibited}, {"conditional-load", "fuel", "acid", evidence.Conditional}, {"missing-load", "fuel", "unknown", evidence.Indeterminate}}
	var batchRevision manifest.Revision
	for _, tc := range cases {
		revision := sampleManifest(tc.id, tc.left, tc.right, at)
		if tc.id == "missing-load" {
			revision.Items[1].SubstanceVersion = 99
		}
		if tc.id == "allowed-load" {
			batchRevision = revision
		}
		if err := c.post(ctx, "/v1/manifests", revision, nil, 201); err != nil {
			return fmt.Errorf("create %s: %w", tc.id, err)
		}
		var frozen manifest.Revision
		if err := c.post(ctx, fmt.Sprintf("/v1/manifests/%s/1/freeze", tc.id), nil, &frozen, 200); err != nil {
			return err
		}
		var result evidence.Result
		if err := c.post(ctx, "/v1/validate", map[string]any{"manifest_id": tc.id, "revision": 1}, &result, 200); err != nil {
			return err
		}
		if result.Decision != tc.want {
			return fmt.Errorf("%s decision=%s want=%s missing=%v hits=%v", tc.id, result.Decision, tc.want, result.Missing, result.RuleHits)
		}
		if tc.want != evidence.Allowed && tc.want != evidence.Indeterminate && len(result.Conflicts) == 0 {
			return fmt.Errorf("%s has no conflict evidence", tc.id)
		}
		if tc.want == evidence.Indeterminate && len(result.Missing) == 0 {
			return fmt.Errorf("missing data was not reported")
		}
		fmt.Printf("PASS %-16s decision=%s rules=%d conflicts=%d missing=%d\n", tc.id, result.Decision, len(result.RuleHits), len(result.Conflicts), len(result.Missing))
		if tc.id == "allowed-load" {
			var replay evidence.Result
			headers, status, err := c.request(ctx, http.MethodPost, "/v1/validate", map[string]any{"manifest_id": tc.id, "revision": 1}, &replay, nil)
			if err != nil || status != 200 || headers.Get("X-Idempotent-Replay") != "true" || replay.ID != result.ID {
				return fmt.Errorf("idempotent replay failed status=%d header=%s error=%v", status, headers.Get("X-Idempotent-Replay"), err)
			}
		}
	}
	if err := streamBatch(ctx, c, batchRevision); err != nil {
		return err
	}
	withdraw := map[string]any{"withdrawn_at": at.Add(24 * time.Hour)}
	if err := c.post(ctx, "/v1/rulebooks/2026.08/withdraw", withdraw, nil, 204); err != nil {
		return err
	}
	_, status, err := c.request(ctx, http.MethodPost, "/v1/reevaluate", map[string]any{"manifest_id": "allowed-load", "revision": 1, "rulebook_version": "2026.08", "effective_at": at.Add(48 * time.Hour)}, nil, nil)
	if err != nil {
		return err
	}
	if status != 422 {
		return fmt.Errorf("withdrawn rulebook reevaluation status=%d", status)
	}
	fmt.Println("PASS withdrawn rulebook rejected for later reevaluation")
	return nil
}
func sampleRulebook(at time.Time) rulebook.Rulebook {
	maxTemp := 25.0
	return rulebook.Rulebook{ID: "ground-hazmat", Version: "2026.08", EffectiveFrom: at.Add(-time.Hour), Rules: []rulebook.Rule{{ID: "oxidizer-flammable-ban", Priority: 100, Predicate: rulebook.ClassPair, ClassA: "3", ClassB: "5.1", Outcome: evidence.Prohibited, Message: "flammable liquid and oxidizer may not share this load"}, {ID: "acid-flammable-conditional", Priority: 80, Predicate: rulebook.ClassPair, ClassA: "3", ClassB: "8", Outcome: evidence.Conditional, Message: "flammable liquid and corrosive require segregation review"}, {ID: "acid-flammable-distance", Priority: 90, Predicate: rulebook.MinimumSeparation, ClassA: "3", ClassB: "8", Outcome: evidence.Conditional, MinDistanceM: 10, Message: "flammable liquid and corrosive require ten metres separation"}, {ID: "flammable-pair-allow", Priority: 50, Predicate: rulebook.ClassPair, ClassA: "3", ClassB: "3", Outcome: evidence.Allowed, Message: "compatible packaged flammable liquids are allowed"}, {ID: "flammable-temp", Priority: 40, Predicate: rulebook.TemperatureRequired, ClassA: "3", Outcome: evidence.Conditional, TemperatureMaxC: &maxTemp, Message: "flammable liquid above 25 C requires active cooling"}}}
}
func sampleSubstances(at time.Time) []substance.Substance {
	return []substance.Substance{{ID: "fuel", Version: 1, UNNumber: "UN1203", Name: "Motor spirit", Synonyms: []string{"gasoline"}, Classes: []classification.Set{{Primary: classification.FlammableLiquid}}, PackagingGroup: substance.GroupII, State: substance.Liquid, Temperature: quantity.Interval{Min: -20, Max: 25, Unit: "C"}, Concentration: quantity.Interval{Min: 95, Max: 100, Unit: "%"}, EffectiveFrom: at.Add(-time.Hour)}, {ID: "oxidizer", Version: 1, UNNumber: "UN2014", Name: "Hydrogen peroxide solution", Classes: []classification.Set{{Primary: classification.Oxidizer}}, PackagingGroup: substance.GroupII, State: substance.Liquid, Temperature: quantity.Interval{Min: 5, Max: 20, Unit: "C"}, Concentration: quantity.Interval{Min: 20, Max: 40, Unit: "%"}, EffectiveFrom: at.Add(-time.Hour)}, {ID: "acid", Version: 1, UNNumber: "UN1789", Name: "Hydrochloric acid", Classes: []classification.Set{{Primary: classification.Corrosive}}, PackagingGroup: substance.GroupII, State: substance.Liquid, Temperature: quantity.Interval{Min: 5, Max: 30, Unit: "C"}, Concentration: quantity.Interval{Min: 20, Max: 35, Unit: "%"}, EffectiveFrom: at.Add(-time.Hour)}}
}
func sampleManifest(id, left, right string, at time.Time) manifest.Revision {
	return manifest.Revision{ManifestID: id, Revision: 1, RulebookVersion: "2026.08", Mode: "road", EffectiveAt: at, State: manifest.Draft, Layout: layout.Layout{ID: "trailer-1", Compartments: []layout.Compartment{{ID: "front", Zone: "A", X: 0, Y: 0, TemperatureMinC: -25, TemperatureMaxC: 30}, {ID: "rear", Zone: "B", X: 5, Y: 0, TemperatureMinC: -25, TemperatureMaxC: 30}}}, Containers: []container.Container{{ID: "c1", Type: container.Drum, Material: "steel", CertifiedClasses: []string{"3", "5.1", "8"}, MaxQuantity: 500, QuantityUnit: "kg"}, {ID: "c2", Type: container.Drum, Material: "steel", CertifiedClasses: []string{"3", "5.1", "8"}, MaxQuantity: 500, QuantityUnit: "kg"}}, Items: []manifest.Item{{ID: "line-1", SubstanceID: left, SubstanceVersion: 1, Quantity: quantity.Interval{Min: 100000, Max: 100000, Unit: "g"}, Concentration: quantity.Interval{Min: 98, Max: 100, Unit: "%"}, Temperature: quantity.Interval{Min: 10, Max: 20, Unit: "C"}, ContainerID: "c1", CompartmentID: "front"}, {ID: "line-2", SubstanceID: right, SubstanceVersion: 1, Quantity: quantity.Interval{Min: 90, Max: 110, Unit: "kg"}, Concentration: quantity.Interval{Min: 25, Max: 35, Unit: "%"}, Temperature: quantity.Interval{Min: 50, Max: 68, Unit: "F"}, ContainerID: "c2", CompartmentID: "rear"}}}
}
func streamBatch(ctx context.Context, c client, revision manifest.Revision) error {
	payload, _ := json.Marshal(revision)
	body := append(append(append([]byte{}, payload...), '\n'), payload...)
	body = append(body, '\n')
	body = append(body, []byte(`{"manifest_id":"broken","unknown_field":true}`)...)
	body = append(body, '\n')
	headers, status, err := c.request(ctx, http.MethodPost, "/v1/batches/validate", body, nil, map[string]string{"Idempotency-Key": "sim-batch-1", "Content-Type": "application/x-ndjson"})
	_ = headers
	if err != nil {
		return err
	}
	if status != 200 {
		return fmt.Errorf("batch status=%d", status)
	}
	request, _ := http.NewRequestWithContext(ctx, http.MethodPost, c.base+"/v1/batches/validate", bytes.NewReader(body))
	request.Header.Set("Authorization", "Bearer "+c.token)
	request.Header.Set("Idempotency-Key", "sim-batch-2")
	request.Header.Set("Content-Type", "application/x-ndjson")
	response, err := c.http.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	scanner := bufio.NewScanner(response.Body)
	validated, quarantined, reused := 0, 0, 0
	for scanner.Scan() {
		var item validation.BatchItem
		if err := json.Unmarshal(scanner.Bytes(), &item); err != nil {
			return err
		}
		switch item.Status {
		case "validated":
			validated++
			if item.Reused {
				reused++
			}
		case "quarantined":
			quarantined++
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	if validated != 2 || quarantined != 1 || reused < 1 {
		return fmt.Errorf("batch counts validated=%d quarantined=%d reused=%d", validated, quarantined, reused)
	}
	fmt.Printf("PASS streaming batch validated=%d quarantined=%d idempotent_replays=%d\n", validated, quarantined, reused)
	return nil
}
func (c client) post(ctx context.Context, path string, input, output any, want int) error {
	_, status, err := c.request(ctx, http.MethodPost, path, input, output, nil)
	if err != nil {
		return err
	}
	if status != want {
		return fmt.Errorf("POST %s status=%d want=%d", path, status, want)
	}
	return nil
}
func (c client) request(ctx context.Context, method, path string, input, output any, extra map[string]string) (http.Header, int, error) {
	var body io.Reader
	switch v := input.(type) {
	case nil:
	case []byte:
		body = bytes.NewReader(v)
	default:
		payload, err := json.Marshal(v)
		if err != nil {
			return nil, 0, err
		}
		body = bytes.NewReader(payload)
	}
	request, err := http.NewRequestWithContext(ctx, method, c.base+path, body)
	if err != nil {
		return nil, 0, err
	}
	request.Header.Set("Authorization", "Bearer "+c.token)
	if input != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	for k, v := range extra {
		request.Header.Set(k, v)
	}
	response, err := c.http.Do(request)
	if err != nil {
		return nil, 0, err
	}
	defer response.Body.Close()
	payload, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, 0, err
	}
	if output != nil && len(payload) > 0 {
		if err := json.Unmarshal(payload, output); err != nil {
			return response.Header, response.StatusCode, fmt.Errorf("decode %s: %w body=%s", path, err, strings.TrimSpace(string(payload)))
		}
	}
	return response.Header, response.StatusCode, nil
}

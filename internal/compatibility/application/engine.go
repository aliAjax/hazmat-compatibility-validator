package application

import (
	"fmt"
	"math"
	"sort"
	"strings"

	container "github.com/enterprise-labs/hazmat-compatibility-validator/internal/container/domain"
	evidence "github.com/enterprise-labs/hazmat-compatibility-validator/internal/evidence/domain"
	manifest "github.com/enterprise-labs/hazmat-compatibility-validator/internal/manifest/domain"
	quantityapp "github.com/enterprise-labs/hazmat-compatibility-validator/internal/quantity/application"
	quantity "github.com/enterprise-labs/hazmat-compatibility-validator/internal/quantity/domain"
	rulebook "github.com/enterprise-labs/hazmat-compatibility-validator/internal/rulebook/domain"
	substance "github.com/enterprise-labs/hazmat-compatibility-validator/internal/substance/domain"
)

type Engine struct{ Converter quantityapp.Converter }
type candidate struct {
	key      string
	hit      evidence.RuleHit
	conflict evidence.Conflict
}
type Evaluation struct {
	Decision    evidence.Decision
	Facts       []evidence.Fact
	Conversions []evidence.Conversion
	Hits        []evidence.RuleHit
	Conflicts   []evidence.Conflict
	Missing     []string
}

func (e Engine) Evaluate(revision manifest.Revision, book rulebook.Rulebook, resolved map[string]substance.Substance) Evaluation {
	out := Evaluation{Decision: evidence.Allowed}
	items := revision.Items
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	containers := map[string]container.Container{}
	for _, c := range cloneContainers(revision.Containers) {
		containers[c.ID] = c
	}
	rules := book.Rules
	sort.Slice(rules, func(i, j int) bool { return rules[i].Priority > rules[j].Priority })
	var candidates []candidate
	for _, item := range items {
		s, ok := resolved[item.ID]
		if !ok {
			out.Missing = append(out.Missing, "items."+item.ID+".substance")
			continue
		}
		out.Facts = append(out.Facts, evidence.Fact{Path: "items." + item.ID + ".un_number", Value: s.UNNumber, Source: fmt.Sprintf("substance:%s@%d", s.ID, s.Version)})
		quantityKG, err := e.convert(item.ID+".quantity", item.Quantity, quantity.Mass, &out)
		if err != nil {
			out.Missing = append(out.Missing, "items."+item.ID+".quantity:"+err.Error())
		}
		tempC, tempErr := e.convert(item.ID+".temperature", item.Temperature, quantity.Temperature, &out)
		if tempErr != nil {
			out.Missing = append(out.Missing, "items."+item.ID+".temperature:"+tempErr.Error())
		}
		concentration, concErr := e.convert(item.ID+".concentration", item.Concentration, quantity.Concentration, &out)
		if concErr != nil {
			out.Missing = append(out.Missing, "items."+item.ID+".concentration:"+concErr.Error())
		}
		compartment, _ := revision.Layout.Find(item.CompartmentID)
		actualContainer := containers[item.ContainerID]
		if tempErr == nil && (tempC.Max < compartment.TemperatureMinC || tempC.Min > compartment.TemperatureMaxC) {
			candidates = append(candidates, invariant(item.ID, "temperature-compartment", evidence.Prohibited, "item temperature range is outside compartment capability", []string{"items", item.ID, "temperature", "layout", item.CompartmentID}))
		}
		for _, rule := range rules {
			if !contextMatches(rule, revision.Mode, actualContainer.Type) {
				continue
			}
			match, uncertain := unaryMatches(rule, s, quantityKG, tempC, concentration, actualContainer)
			if !match {
				continue
			}
			outcome := rule.Outcome
			if uncertain && outcome == evidence.Allowed {
				outcome = evidence.Conditional
			}
			candidates = append(candidates, candidate{key: item.ID + "|" + string(rule.Predicate), hit: evidence.RuleHit{RuleID: rule.ID, Priority: rule.Priority, Outcome: outcome, Explanation: rule.Message}, conflict: evidence.Conflict{LeftItem: item.ID, Path: []string{"items", item.ID, string(rule.Predicate), "rules", rule.ID}, Message: rule.Message}})
		}
	}
	for left := 0; left < len(items); left++ {
		for right := left + 1; right < len(items); right++ {
			a, b := items[left], items[right]
			sa, oka := resolved[a.ID]
			sb, okb := resolved[b.ID]
			if !oka || !okb {
				continue
			}
			matched := false
			for _, rule := range rules {
				if rule.Predicate != rulebook.ClassPair && rule.Predicate != rulebook.SameContainer && rule.Predicate != rulebook.MinimumSeparation {
					continue
				}
				ca, cb := containers[a.ContainerID], containers[b.ContainerID]
				if !contextMatches(rule, revision.Mode, ca.Type) && !contextMatches(rule, revision.Mode, cb.Type) {
					continue
				}
				if !classPair(rule, sa, sb) {
					continue
				}
				applies := false
				switch rule.Predicate {
				case rulebook.ClassPair:
					applies = true
				case rulebook.SameContainer:
					applies = a.ContainerID == b.ContainerID
				case rulebook.MinimumSeparation:
					ac, _ := revision.Layout.Find(a.CompartmentID)
					bc, _ := revision.Layout.Find(b.CompartmentID)
					distance := math.Hypot(ac.X-bc.X, ac.Y-bc.Y)
					out.Facts = append(out.Facts, evidence.Fact{Path: "separation." + a.ID + "." + b.ID, Value: distance, Source: "layout coordinates in metres"})
					applies = distance < rule.MinDistanceM
				}
				if applies {
					matched = true
					candidates = append(candidates, candidate{key: a.ID + "|" + b.ID + "|" + string(rule.Predicate), hit: evidence.RuleHit{RuleID: rule.ID, Priority: rule.Priority, Outcome: rule.Outcome, Explanation: rule.Message}, conflict: evidence.Conflict{LeftItem: a.ID, RightItem: b.ID, Path: []string{"items", a.ID, "items", b.ID, string(rule.Predicate), "rules", rule.ID}, Message: rule.Message}})
				}
			}
			if !matched {
				out.Missing = append(out.Missing, fmt.Sprintf("compatibility_rule:%s:%s", a.ID, b.ID))
			}
		}
	}
	candidates = append(candidates, e.containerCapacity(revision, containers, &out)...)
	selected := selectPriority(candidates)
	for _, c := range selected {
		out.Hits = append(out.Hits, c.hit)
		if c.hit.Outcome != evidence.Allowed {
			out.Conflicts = append(out.Conflicts, c.conflict)
		}
		out.Decision = moreSevere(out.Decision, c.hit.Outcome)
	}
	if len(out.Missing) > 0 {
		out.Decision = evidence.Indeterminate
	}
	sort.Strings(out.Missing)
	sort.Slice(out.Hits, func(i, j int) bool {
		if out.Hits[i].Priority == out.Hits[j].Priority {
			return out.Hits[i].RuleID < out.Hits[j].RuleID
		}
		return out.Hits[i].Priority > out.Hits[j].Priority
	})
	return out
}

func (e Engine) convert(field string, input quantity.Interval, dimension quantity.Dimension, out *Evaluation) (quantity.Interval, error) {
	converted, err := e.Converter.Convert(input, dimension)
	if err != nil {
		return quantity.Interval{}, err
	}
	out.Conversions = append(out.Conversions, evidence.Conversion{Field: field, Input: fmt.Sprintf("[%g,%g] %s", input.Min, input.Max, input.Unit), Output: fmt.Sprintf("[%g,%g] %s", converted.Output.Min, converted.Output.Max, converted.Output.Unit), Formula: converted.Formula})
	return converted.Output, nil
}
func invariant(item, key string, outcome evidence.Decision, message string, path []string) candidate {
	return candidate{key: item + "|" + key, hit: evidence.RuleHit{RuleID: "invariant:" + key, Priority: 1000000, Outcome: outcome, Explanation: message}, conflict: evidence.Conflict{LeftItem: item, Path: path, Message: message}}
}
func contextMatches(r rulebook.Rule, mode string, kind container.Type) bool {
	if len(r.Modes) > 0 && !contains(r.Modes, mode) {
		return false
	}
	if len(r.Containers) > 0 && !contains(r.Containers, string(kind)) {
		return false
	}
	return true
}
func unaryMatches(r rulebook.Rule, s substance.Substance, q, temp, conc quantity.Interval, c container.Container) (bool, bool) {
	class := r.ClassA == "" || substanceHasClass(s, r.ClassA)
	switch r.Predicate {
	case rulebook.QuantityLimit:
		if !class || q.Unit == "" {
			return false, false
		}
		return q.Max > r.MaxQuantityKG, q.Min <= r.MaxQuantityKG && q.Max > r.MaxQuantityKG
	case rulebook.TemperatureRequired:
		if !class || r.TemperatureMaxC == nil || temp.Unit == "" {
			return false, false
		}
		return temp.Max > *r.TemperatureMaxC, temp.Min <= *r.TemperatureMaxC && temp.Max > *r.TemperatureMaxC
	case rulebook.TransportMode:
		return class, true
	case rulebook.ContainerType:
		return class, true
	case rulebook.ConcentrationRange:
		if !class || conc.Unit == "" || r.ConcentrationMinPct == nil || r.ConcentrationMaxPct == nil {
			return false, false
		}
		overlap := conc.Min <= *r.ConcentrationMaxPct && *r.ConcentrationMinPct <= conc.Max
		contained := conc.Min >= *r.ConcentrationMinPct && conc.Max <= *r.ConcentrationMaxPct
		return overlap, overlap && !contained
	}
	return false, false
}
func classPair(r rulebook.Rule, a, b substance.Substance) bool {
	return substanceHasClass(a, r.ClassA) && substanceHasClass(b, r.ClassB) || substanceHasClass(a, r.ClassB) && substanceHasClass(b, r.ClassA)
}
func substanceHasClass(s substance.Substance, class string) bool {
	if class == "" {
		return true
	}
	for _, set := range s.Classes {
		for _, c := range set.All() {
			if string(c) == class || strings.HasPrefix(string(c), class+".") {
				return true
			}
		}
	}
	return false
}
func contains(values []string, want string) bool {
	for _, v := range values {
		if strings.EqualFold(v, want) {
			return true
		}
	}
	return false
}
func selectPriority(values []candidate) []candidate {
	sort.Slice(values, func(i, j int) bool { return values[i].key < values[j].key })
	best := map[string]candidate{}
	for _, v := range values {
		prior, ok := best[v.key]
		if !ok || v.hit.Priority > prior.hit.Priority || (v.hit.Priority == prior.hit.Priority && severity(v.hit.Outcome) > severity(prior.hit.Outcome)) {
			best[v.key] = v
		}
	}
	out := make([]candidate, 0, len(best))
	for _, v := range best {
		out = append(out, v)
	}
	return out
}
func moreSevere(a, b evidence.Decision) evidence.Decision {
	if severity(b) > severity(a) {
		return b
	}
	return a
}
func severity(d evidence.Decision) int {
	switch d {
	case evidence.Prohibited:
		return 4
	case evidence.Indeterminate:
		return 3
	case evidence.Conditional:
		return 2
	case evidence.Allowed:
		return 1
	}
	return 3
}
func (e Engine) containerCapacity(revision manifest.Revision, containers map[string]container.Container, out *Evaluation) []candidate {
	totals := map[string]quantity.Interval{}
	for _, item := range revision.Items {
		q, err := e.convert(item.ID+".container_quantity", item.Quantity, quantity.Mass, out)
		if err != nil {
			continue
		}
		v := totals[item.ContainerID]
		if v.Unit == "" {
			v.Unit = "kg"
		}
		v.Min += q.Min
		v.Max += q.Max
		totals[item.ContainerID] = v
	}
	var values []candidate
	for id, total := range totals {
		c := containers[id]
		capacity, err := e.Converter.Convert(quantity.Interval{Min: c.MaxQuantity, Max: c.MaxQuantity, Unit: c.QuantityUnit}, quantity.Mass)
		if err != nil {
			out.Missing = append(out.Missing, "containers."+id+".capacity_unit")
			continue
		}
		if total.Max > capacity.Output.Max {
			decision := evidence.Prohibited
			if total.Min <= capacity.Output.Max {
				decision = evidence.Conditional
			}
			values = append(values, invariant(id, "container-capacity", decision, "aggregate quantity may exceed container capacity", []string{"containers", id, "max_quantity"}))
		}
	}
	return values
}

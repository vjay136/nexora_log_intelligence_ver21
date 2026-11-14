package fieldmapping

import (
	"fmt"
	"nexora.local/nli-v20/internal/mixedkv"
	"regexp"
	"sort"
	"strings"
)

// GlobalKVTemplateID identifies the built-in automatic KV tail profile.
const GlobalKVTemplateID = "global-auto"

type HeaderTemplate struct {
	Engine         string               `json:"engine"`
	Pattern        string               `json:"pattern"`
	Fields         map[string]string    `json:"fields"`
	OptionalFields []string             `json:"optional_fields,omitempty"`
	ResourceLimits HeaderTemplateLimits `json:"resource_limits,omitempty"`
}

type HeaderTemplateLimits struct {
	MaxInputBytes int `json:"max_input_bytes,omitempty"`
	TimeoutMS     int `json:"timeout_ms,omitempty"`
}

func ComposeTemplates(header HeaderTemplate, profile *mixedkv.Config) (Plan, error) {
	plan := Plan{Version: 1}
	if header.Engine != "" && header.Engine != "re2" || len(header.Pattern) > MaxRegexBytes ||
		!strings.HasPrefix(header.Pattern, "^") || len(header.Fields) == 0 || len(header.Fields) >= MaxFields {
		return plan, fmt.Errorf("header template needs an anchored RE2 prefix and named output fields")
	}
	re, err := regexp.Compile(header.Pattern)
	if err != nil || re.MatchString("") {
		return plan, fmt.Errorf("header pattern must be valid RE2 and consume a nonempty prefix")
	}
	outputNames := make(map[string]bool, len(header.Fields))
	for _, name := range header.Fields {
		outputNames[name] = true
	}
	optional := make(map[string]bool, len(header.OptionalFields))
	for _, name := range header.OptionalFields {
		if !outputNames[name] || optional[name] {
			return plan, fmt.Errorf("invalid or duplicate optional output field %q", name)
		}
		optional[name] = true
	}
	groups := make([]string, 0, len(header.Fields))
	for group := range header.Fields {
		if re.SubexpIndex(group) < 1 {
			return plan, fmt.Errorf("header field refers to unknown named group %q", group)
		}
		groups = append(groups, group)
	}
	sort.Slice(groups, func(i, j int) bool { return re.SubexpIndex(groups[i]) < re.SubexpIndex(groups[j]) })
	for _, group := range groups {
		policy := "required"
		if optional[header.Fields[group]] {
			policy = "optional"
		}
		plan.Fields = append(plan.Fields, Field{
			Name: header.Fields[group], Method: "regex", Source: header.Pattern, Part: group, Type: "string", Policy: policy,
		})
	}
	if profile == nil {
		plan.Fields = append(plan.Fields, Field{
			Name: "kv_pair", Type: "object", Method: "kv_all", Policy: "optional",
			KVProfile: "auto", KVHeaderPattern: header.Pattern,
			KeyOptions: KeyOptions{RequireComplete: true},
		})
		return plan, nil
	}
	kv := *profile
	if len(kv.KeyAllowPatterns) != 0 || len(kv.KeyDenyPatterns) != 0 || len(kv.StopKeys) != 0 || len(kv.ValueTypes) != 0 ||
		kv.AllowLeadingText || kv.MaxInputBytes != 0 && kv.MaxInputBytes != MaxInputBytes ||
		kv.MaxKeyBytes != 0 && kv.MaxKeyBytes != 128 || kv.MaxValueBytes != 0 && kv.MaxValueBytes != MaxInputBytes {
		return plan, fmt.Errorf("the combined template requires complete, string-valued KV coverage with standard input and value limits")
	}
	// Composition gives every configured separator literal boundary semantics.
	// An unexplained segment must fail or be explicitly captured by the asset.
	kv.LiteralBoundaries = true
	if kv.MaxFields == 0 {
		kv.MaxFields = DefaultAdvancedKVPairs
	}
	if _, err := mixedkv.Compile(kv); err != nil {
		return plan, err
	}
	if len(kv.PairSeparators) == 0 || len(kv.AssignmentSeparators) == 0 {
		return plan, fmt.Errorf("the KV template needs explicit pair and assignment separators")
	}
	policy := string(kv.DuplicatePolicy)
	if policy == "" || policy == "error" {
		policy = "reject"
	}
	tail := Field{
		Name: "kv_pair", Type: "object", Method: "kv_all", Policy: "optional", KVProfile: "configured",
		KVHeaderPattern: header.Pattern, KVLiteralBoundaries: true, KVMaxPairs: kv.MaxFields,
		KVBareField: kv.BareField, KVEmptyKeyField: kv.EmptyKeyField, DuplicatePolicy: policy,
		KVPairSeparators: kv.PairSeparators, KVQuotes: kv.QuoteCharacters, KVEscape: kv.EscapeCharacter,
		KVKeyPattern: kv.KeyPattern, KVRejectEmpty: kv.RejectEmptyValues,
		KeyOptions: KeyOptions{
			AssignmentSeparators: kv.AssignmentSeparators, KeyAliases: kv.KeyAliases, RequireComplete: true,
		},
	}
	if kv.BracketPairs != nil {
		tail.KVBrackets = &kv.BracketPairs
	}
	return Plan{Version: 1, Fields: append(plan.Fields, tail)}, nil
}

// Package fieldmapping implements NLI's bounded Native Field Intelligence (NFI)
// mapping plans. Plans are data, never executable expressions. The same compiled
// plan is used by controller previews and the client parser-policy executor.
package fieldmapping

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	MaxInputBytes = 1 << 20
	MaxFields     = 128
	MaxOperations = 16
	MaxRegexBytes = 4096
	MaxDepth      = 32
	MaxTokens     = 65536
	MaxPlanBytes  = 2 << 20
)

var Methods = []string{"position", "delimiter", "character", "byte", "line", "key_value", "regex", "json_path", "constant", "derived", "raw", "field", "word", "token", "token_match", "before", "after", "between", "count", "kv_all", "regex_all", "regex_groups"}
var Operations = []string{"split", "select", "substring", "character", "byte", "line", "key_value", "replace", "remove", "trim", "lowercase", "uppercase", "prefix", "suffix", "add", "strip_prefix", "strip_suffix", "regex_extract", "regex_replace", "join", "merge", "first", "last", "range", "count", "before", "after", "between", "tokenize", "token_match", "split_regex", "split_characters", "split_words", "split_tokens", "conditional_merge", "copy", "mask", "hash", "redact", "normalize", "omit", "exclude"}
var Types = []string{"string", "integer", "number", "boolean", "timestamp", "ip", "port", "object", "array", "email", "url", "uuid", "ipv6", "date", "datetime", "null", "custom"}
var ReservedNames = []string{"raw", "nli", "meta", "source", "envelope", "parser_id", "attempt_count", "ordered_attempts", "parse_telemetry"}
var fieldName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]{0,63}$`)

// ErrSourceAbsent distinguishes an absent nested selector from malformed input
// or a conversion failure. Only absence may invoke the field presence policy.
var ErrSourceAbsent = errors.New("source value is absent")

type Plan struct {
	Version int     `json:"version"`
	Fields  []Field `json:"fields"`
}

// MarshalJSON keeps the new stage's typed programmatic form byte-equivalent to
// its strict map-decoded form. Existing pack hash contracts remain unchanged.
func (p Plan) MarshalJSON() ([]byte, error) {
	type plain Plan
	data, err := json.Marshal(plain(p))
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var canonical map[string]any
	if err := decoder.Decode(&canonical); err != nil {
		return nil, err
	}
	return json.Marshal(canonical)
}

type KeyValueEdit struct {
	Key    string `json:"key"`
	Action string `json:"action"`
	Value  string `json:"value,omitempty"`
}

type Field struct {
	Selection
	KeyOptions
	Name             string             `json:"name"`
	Label            string             `json:"label,omitempty"`
	Type             string             `json:"type,omitempty"`
	Policy           string             `json:"policy,omitempty"`
	Method           string             `json:"method"`
	Source           string             `json:"source,omitempty"`
	SourceEnd        string             `json:"source_end,omitempty"`
	Delimiter        string             `json:"delimiter,omitempty"`
	Part             string             `json:"part,omitempty"`
	Confirmed        bool               `json:"confirmed,omitempty"`
	Transform        string             `json:"transform,omitempty"`
	TimestampFormat  string             `json:"timestamp_format,omitempty"`
	Timezone         string             `json:"timezone,omitempty"`
	TimezoneName     string             `json:"timezone_name,omitempty"`
	DefaultValue     string             `json:"default_value,omitempty"`
	Validation       string             `json:"validation,omitempty"`
	NullPolicy       string             `json:"null_policy,omitempty"`
	KVSeparator      string             `json:"kv_separator,omitempty"`
	PairDelimiters   string             `json:"pair_delimiters,omitempty"`
	DuplicatePolicy  string             `json:"duplicate_policy,omitempty"`
	JoinSeparator    string             `json:"join_separator,omitempty"`
	QuoteAware       *bool              `json:"quote_aware,omitempty"`
	Operations       []Operation        `json:"operations,omitempty"`
	OutputPolicy     string             `json:"output_policy,omitempty"`
	KVProfile        string             `json:"kv_profile,omitempty"`
	KVStart          string             `json:"kv_start,omitempty"`
	KVPairSeparators []string           `json:"kv_pair_separators,omitempty"`
	KVQuotes         string             `json:"kv_quotes,omitempty"`
	KVEscape         string             `json:"kv_escape,omitempty"`
	KVBrackets       *map[string]string `json:"kv_brackets,omitempty"`
	KVKeyPattern     string             `json:"kv_key_pattern,omitempty"`
	KVRejectEmpty    bool               `json:"kv_reject_empty,omitempty"`
	KVNestedObjects  bool               `json:"kv_nested_objects,omitempty"`
	KeyValueEdits    []KeyValueEdit     `json:"key_value_edits,omitempty"`
	KVInputEdits     []KVInputEdit      `json:"kv_input_edits,omitempty"`
	EndOfLine        bool               `json:"end_of_line,omitempty"`
	EmptyPolicy      string             `json:"empty_policy,omitempty"`
	CustomType       string             `json:"custom_type,omitempty"`
	CustomPattern    string             `json:"custom_pattern,omitempty"`
	Description      string             `json:"description,omitempty"`
	Example          string             `json:"example,omitempty"`

	KVHeaderPattern     string `json:"kv_header_pattern,omitempty"`
	KVMaxPairs          int    `json:"kv_max_pairs,omitempty"`
	KVLiteralBoundaries bool   `json:"kv_literal_boundaries,omitempty"`
	KVBareField         string `json:"kv_bare_field,omitempty"`
	KVEmptyKeyField     string `json:"kv_empty_key_field,omitempty"`

	// These bounded display-only values allow existing browser drafts to be
	// round-tripped without permitting arbitrary configuration properties.
	ID         string `json:"id,omitempty"`
	Status     string `json:"status,omitempty"`
	Preview    string `json:"preview,omitempty"`
	Evidence   string `json:"evidence,omitempty"`
	Confidence string `json:"confidence,omitempty"`
}

// UsesAdvancedKV reports whether this field requires the advanced KV runtime.
func (f Field) UsesAdvancedKV() bool {
	return f.KVLiteralBoundaries || f.KVBareField != "" || f.KVEmptyKeyField != "" || f.KVHeaderPattern != "" || f.KVMaxPairs != 0
}

// KVPairLimit returns the bounded pair budget for this mapping.
func (f Field) KVPairLimit() int {
	if f.KVMaxPairs > 0 {
		return f.KVMaxPairs
	}
	return MaxFields
}

// UsesLargeKV reports whether this plan depends on the raised pair or output budget.
func (f Field) UsesLargeKV() bool {
	return f.KVLiteralBoundaries && f.OutputPolicy == "flatten" && f.KVPairLimit() > MaxFields
}

// FlattenedKVFieldLimit includes room for the declared header fields.
func (p Plan) FlattenedKVFieldLimit() int {
	limit := MaxFields
	for _, field := range p.Fields {
		if field.Method == "kv_all" && field.OutputPolicy == "flatten" && field.KVLiteralBoundaries && field.KVPairLimit() > MaxFields {
			limit = max(limit, MaxFields+field.KVPairLimit())
		}
	}
	return limit
}

type KVInputEdit struct {
	Action   string `json:"action"`
	Find     string `json:"find"`
	Replace  string `json:"replace,omitempty"`
	Required bool   `json:"required,omitempty"`
}

type Operation struct {
	Selection
	KeyOptions
	Op                string   `json:"op"`
	Delimiter         string   `json:"delimiter,omitempty"`
	Part              int      `json:"part,omitempty"`
	Start             int      `json:"start,omitempty"`
	End               int      `json:"end,omitempty"`
	Value             string   `json:"value,omitempty"`
	Find              string   `json:"find,omitempty"`
	Replace           string   `json:"replace,omitempty"`
	Pattern           string   `json:"pattern,omitempty"`
	Group             string   `json:"group,omitempty"`
	Fields            []string `json:"fields,omitempty"`
	Separator         string   `json:"separator,omitempty"`
	QuoteAware        *bool    `json:"quote_aware,omitempty"`
	Limit             int      `json:"limit,omitempty"`
	PreserveDelimiter bool     `json:"preserve_delimiter,omitempty"`
	ConditionField    string   `json:"condition_field,omitempty"`
	ConditionOperator string   `json:"condition_operator,omitempty"`
	ConditionValue    string   `json:"condition_value,omitempty"`
	ElseValue         string   `json:"else_value,omitempty"`
	RemoveSources     bool     `json:"remove_sources,omitempty"`
	Algorithm         string   `json:"algorithm,omitempty"`
	Mode              string   `json:"mode,omitempty"`
	tokenPattern      *regexp.Regexp
}

// Selection is shared by source selectors and ordered operations. All indexes
// and counts use Unicode characters or explicit token/word units, never bytes
// unless unit=byte is selected. Ranges remain 1-based and inclusive.
type Selection struct {
	BoundaryStart    string   `json:"boundary_start,omitempty"`
	BoundaryEnd      string   `json:"boundary_end,omitempty"`
	Occurrence       string   `json:"occurrence,omitempty"`
	OccurrenceIndex  int      `json:"occurrence_index,omitempty"`
	PreserveBoundary bool     `json:"preserve_boundary,omitempty"`
	Direction        string   `json:"direction,omitempty"`
	Count            int      `json:"count,omitempty"`
	Unit             string   `json:"unit,omitempty"`
	Indices          []int    `json:"indices,omitempty"`
	TokenMode        string   `json:"token_mode,omitempty"`
	TokenSeparators  []string `json:"token_separators,omitempty"`
	TokenPattern     string   `json:"token_pattern,omitempty"`
}

type KeyOptions struct {
	RequireComplete      bool              `json:"require_complete,omitempty"`
	AssignmentSeparators []string          `json:"assignment_separators,omitempty"`
	KeyAliases           map[string]string `json:"key_aliases,omitempty"`
	KeyInclude           []string          `json:"key_include,omitempty"`
	KeyExclude           []string          `json:"key_exclude,omitempty"`
	KeyPrefix            string            `json:"key_prefix,omitempty"`
	KeySuffix            string            `json:"key_suffix,omitempty"`
}

type Evidence struct {
	Start       int    `json:"start"`
	End         int    `json:"end"`
	Source      bool   `json:"source"`
	Description string `json:"description"`
	Operations  int    `json:"operations"`
	Spans       []Span `json:"spans,omitempty"`
	keySpans    map[string][]Span
}

type Span struct {
	Start int `json:"start"`
	End   int `json:"end"`
}
type TraceStep struct {
	Index       int    `json:"index"`
	Operation   string `json:"operation"`
	Input       string `json:"input"`
	Output      string `json:"output"`
	SourceStart int    `json:"source_start"`
	SourceEnd   int    `json:"source_end"`
	Truncated   bool   `json:"truncated,omitempty"`
}

type Result struct {
	Values         map[string]any                 `json:"values"`
	Evidence       map[string]Evidence            `json:"evidence"`
	Trace          map[string][]TraceStep         `json:"trace,omitempty"`
	NestedEvidence map[string]map[string]Evidence `json:"nested_evidence,omitempty"`
	DynamicFields  []string                       `json:"dynamic_fields,omitempty"`
	kvieScans      map[kvieCacheKey]kvieScan
}

type compiledField struct {
	field         Field
	pattern       *regexp.Regexp
	ops           []*regexp.Regexp
	tokenPattern  *regexp.Regexp
	customPattern *regexp.Regexp
	kvie          *kvieProfile
	kvHeader      *regexp.Regexp
}

type Compiled struct {
	fields           []compiledField
	outputFieldLimit int
}

func contains(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}
func index(text string, fallback int) (int, error) {
	if text == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(text)
	if err != nil || n < 1 || n > MaxInputBytes {
		return 0, fmt.Errorf("index must be an integer between 1 and %d", MaxInputBytes)
	}
	return n, nil
}
func boundedRegex(pattern string) (*regexp.Regexp, error) {
	if len(pattern) == 0 || len(pattern) > MaxRegexBytes {
		return nil, fmt.Errorf("regex must contain 1..%d bytes", MaxRegexBytes)
	}
	return regexp.Compile(pattern)
}

// Compile validates every selector and operation, and precompiles all regular
// expressions once. The compiled value is immutable and safe to share.
func Compile(plan Plan) (*Compiled, error) {
	data, encodeErr := json.Marshal(plan)
	if encodeErr != nil || len(data) > MaxPlanBytes {
		return nil, fmt.Errorf("mapping plan exceeds aggregate configuration bounds")
	}
	if plan.Version != 1 {
		return nil, fmt.Errorf("mapping plan version must be 1")
	}
	if len(plan.Fields) == 0 || len(plan.Fields) > MaxFields {
		return nil, fmt.Errorf("mapping needs 1..%d fields", MaxFields)
	}
	compiled := &Compiled{outputFieldLimit: plan.FlattenedKVFieldLimit()}
	seen := map[string]bool{}
	patterns := map[string]*regexp.Regexp{}
	planBytes := 0
	for _, original := range plan.Fields {
		f := original
		planBytes += len(f.Name) + len(f.Label) + len(f.Source) + len(f.DefaultValue) + len(f.Preview) + len(f.Evidence) + len(f.Validation) + len(f.TimestampFormat) + len(f.TimezoneName)
		if planBytes > MaxPlanBytes {
			return nil, fmt.Errorf("mapping plan exceeds aggregate configuration bounds")
		}
		if f.QuoteAware != nil {
			value := *f.QuoteAware
			f.QuoteAware = &value
		}
		if !fieldName.MatchString(f.Name) || strings.HasPrefix(f.Name, "_") || contains(ReservedNames, strings.ToLower(f.Name)) {
			return nil, fmt.Errorf("field %q must be a non-reserved identifier of 1..64 characters", f.Name)
		}
		if seen[f.Name] {
			return nil, fmt.Errorf("duplicate output field %q", f.Name)
		}
		if len(f.Label) > 256 || len(f.ID) > 128 || len(f.Status) > 64 || len(f.Confidence) > 128 || len(f.Preview) > MaxInputBytes || len(f.Evidence) > MaxInputBytes {
			return nil, fmt.Errorf("field %q display metadata exceeds bounds", f.Name)
		}
		if f.Type == "" {
			f.Type = "string"
		}
		if f.Policy == "" {
			f.Policy = "required"
		}
		if f.Transform == "" {
			f.Transform = "none"
		}
		if f.NullPolicy == "" {
			f.NullPolicy = "omit"
		}
		if f.Timezone == "" {
			f.Timezone = "preserve"
		}
		if !contains(Types, f.Type) && !contains([]string{"int", "float", "bool", "time"}, f.Type) {
			return nil, fmt.Errorf("field %q has unsupported type", f.Name)
		}
		if !contains([]string{"required", "optional", "nullable", "default", "derived"}, f.Policy) {
			return nil, fmt.Errorf("field %q needs an explicit supported presence policy", f.Name)
		}
		if !contains([]string{"omit", "preserve_null", "reject"}, f.NullPolicy) || !contains([]string{"preserve", "utc", "configured"}, f.Timezone) || !contains([]string{"none", "trim", "lowercase", "uppercase"}, f.Transform) {
			return nil, fmt.Errorf("field %q has invalid null, timezone, or transform policy", f.Name)
		}
		if f.Timezone == "configured" {
			if len(f.TimezoneName) > 128 || f.TimezoneName == "" {
				return nil, fmt.Errorf("field %q configured timezone needs an IANA timezone_name", f.Name)
			}
			if _, err := time.LoadLocation(f.TimezoneName); err != nil {
				return nil, fmt.Errorf("field %q timezone_name is invalid", f.Name)
			}
		}
		if len(f.Source) > MaxRegexBytes || len(f.Delimiter) > 64 || len(f.KVSeparator) > 8 || len(f.PairDelimiters) > 64 || len(f.DefaultValue) > MaxInputBytes || len(f.Validation) > 4096 || len(f.TimestampFormat) > 256 || len(f.JoinSeparator) > 64 {
			return nil, fmt.Errorf("field %q selector exceeds bounds", f.Name)
		}
		if !contains(Methods, f.Method) {
			return nil, fmt.Errorf("field %q has unsupported extraction method %q", f.Name, f.Method)
		}
		cf := compiledField{field: f}
		switch f.Method {
		case "position", "character", "byte", "line":
			start, err := index(f.Source, 1)
			if err != nil {
				return nil, fmt.Errorf("field %q: %w", f.Name, err)
			}
			end, err := index(f.SourceEnd, start)
			if err != nil || end < start {
				return nil, fmt.Errorf("field %q has an invalid inclusive range", f.Name)
			}
		case "delimiter":
			if f.Delimiter == "" {
				return nil, fmt.Errorf("field %q needs a nonempty delimiter", f.Name)
			}
			if f.Source != "raw" && f.Source != "" {
				if _, err := index(f.Source, 1); err != nil {
					return nil, err
				}
			}
			if _, err := index(f.Part, 1); err != nil {
				return nil, err
			}
			if !f.Confirmed {
				return nil, fmt.Errorf("field %q delimiter interpretation must be confirmed", f.Name)
			}
		case "key_value":
			if !regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_.-]*$`).MatchString(f.Source) {
				return nil, fmt.Errorf("field %q needs a valid key name", f.Name)
			}
			if f.DuplicatePolicy == "" {
				f.DuplicatePolicy = "reject"
				cf.field.DuplicatePolicy = f.DuplicatePolicy
			}
			if !contains([]string{"reject", "first", "last", "array"}, f.DuplicatePolicy) {
				return nil, fmt.Errorf("field %q has invalid duplicate policy", f.Name)
			}
			if f.DuplicatePolicy == "array" && f.Type != "array" {
				return nil, fmt.Errorf("field %q duplicate array policy requires array output type", f.Name)
			}
		case "regex":
			re, ok := patterns[f.Source]
			if !ok {
				var err error
				re, err = boundedRegex(f.Source)
				if err != nil {
					return nil, fmt.Errorf("field %q: %w", f.Name, err)
				}
				patterns[f.Source] = re
			}
			cf.pattern = re
			if f.Part != "" {
				n, err := strconv.Atoi(f.Part)
				if err == nil && (n < 0 || n > re.NumSubexp()) || err != nil && re.SubexpIndex(f.Part) < 0 {
					return nil, fmt.Errorf("field %q has no matching regex group", f.Name)
				}
			}
		case "json_path":
			if _, err := parsePath(f.Source); err != nil {
				return nil, fmt.Errorf("field %q: %w", f.Name, err)
			}
		case "derived":
			for _, ref := range strings.Split(f.Source, ",") {
				if !seen[strings.TrimSpace(ref)] {
					return nil, fmt.Errorf("field %q references missing or later field %q", f.Name, ref)
				}
			}
		}
		if err := validateRules(f.Validation); err != nil {
			return nil, fmt.Errorf("field %q: %w", f.Name, err)
		}
		if len(f.Operations) > MaxOperations {
			return nil, fmt.Errorf("field %q exceeds %d operations", f.Name, MaxOperations)
		}
		cf.ops = make([]*regexp.Regexp, len(f.Operations))
		cf.field.Operations = append([]Operation(nil), f.Operations...)
		for i, op := range f.Operations {
			planBytes += len(op.Value) + len(op.Find) + len(op.Replace) + len(op.Pattern)
			if planBytes > MaxPlanBytes {
				return nil, fmt.Errorf("mapping plan exceeds aggregate configuration bounds")
			}
			if !contains(Operations, op.Op) {
				return nil, fmt.Errorf("field %q operation %q is unsupported", f.Name, op.Op)
			}
			if len(op.Delimiter) > 64 || len(op.Value) > 4096 || len(op.Find) > 4096 || len(op.Replace) > 4096 || len(op.Separator) > 64 || len(op.Fields) > MaxFields || len(op.Group) > 64 {
				return nil, fmt.Errorf("field %q operation exceeds bounds", f.Name)
			}
			if op.Start < 0 || op.End < 0 || op.Start > MaxInputBytes || op.End > MaxInputBytes || op.Part < 0 || op.Part > MaxTokens {
				return nil, fmt.Errorf("field %q operation index exceeds bounds", f.Name)
			}
			if contains([]string{"substring", "character", "byte", "line"}, op.Op) && (op.Start < 1 || op.End != 0 && op.End < op.Start) {
				return nil, fmt.Errorf("field %q operation range must be 1-based inclusive", f.Name)
			}
			if op.Op == "select" && op.Part < 1 {
				return nil, fmt.Errorf("field %q select needs a 1-based part", f.Name)
			}
			if op.Op == "split" && op.Delimiter == "" {
				return nil, fmt.Errorf("field %q split needs a nonempty delimiter", f.Name)
			}
			if op.Op == "key_value" && (!fieldName.MatchString(op.Value) || len(op.Delimiter) > 8) {
				return nil, fmt.Errorf("field %q nested key_value needs a valid key and separator of at most 8 bytes", f.Name)
			}
			if (op.Op == "replace" || op.Op == "remove") && op.Find == "" {
				return nil, fmt.Errorf("field %q replacement needs a nonempty find string", f.Name)
			}
			if op.Op == "regex_extract" || op.Op == "regex_replace" {
				re, err := boundedRegex(op.Pattern)
				if err != nil {
					return nil, fmt.Errorf("field %q operation: %w", f.Name, err)
				}
				cf.ops[i] = re
				if op.Op == "regex_extract" && op.Group != "" {
					n, err := strconv.Atoi(op.Group)
					if err == nil && (n < 0 || n > re.NumSubexp()) || err != nil && re.SubexpIndex(op.Group) < 0 {
						return nil, fmt.Errorf("field %q operation refers to an absent regex group", f.Name)
					}
				}
			}
			for _, ref := range op.Fields {
				if !seen[ref] {
					return nil, fmt.Errorf("field %q operation references missing or later field %q", f.Name, ref)
				}
			}
			cf.field.Operations[i].Fields = append([]string(nil), op.Fields...)
			if op.QuoteAware != nil {
				value := *op.QuoteAware
				cf.field.Operations[i].QuoteAware = &value
			}
		}
		if f.KVHeaderPattern != "" {
			cf.kvHeader = patterns[f.KVHeaderPattern]
		}
		if err := compileExtended(&cf, seen); err != nil {
			return nil, fmt.Errorf("field %q: %w", f.Name, err)
		}
		compiled.fields = append(compiled.fields, cf)
		seen[f.Name] = true
	}
	return compiled, nil
}

// Execute applies one immutable plan to one bounded record. Any invalid required
// value rejects the record instead of emitting a partially successful payload.
// MatchesKVHeader reports whether a bounded composed header applies to this input.
func (c *Compiled) MatchesKVHeader(raw string) bool {
	for _, field := range c.fields {
		if field.kvHeader != nil {
			if span := field.kvHeader.FindStringIndex(raw); len(span) == 2 && span[0] == 0 && span[1] > 0 {
				return true
			}
		}
	}
	return false
}

func (c *Compiled) Execute(ctx context.Context, raw string) (Result, error) {
	return c.execute(ctx, raw, false)
}
func (c *Compiled) ExecuteTrace(ctx context.Context, raw string) (Result, error) {
	return c.execute(ctx, raw, true)
}
func (c *Compiled) execute(ctx context.Context, raw string, trace bool) (Result, error) {
	result := Result{Values: map[string]any{}, Evidence: map[string]Evidence{}}
	prior := map[string]any{}
	traceBudget := 65536
	if trace {
		result.Trace = map[string][]TraceStep{}
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return result, err
	}
	if len(raw) > MaxInputBytes {
		return result, fmt.Errorf("mapping input exceeds %d bytes", MaxInputBytes)
	}
	if strings.ContainsRune(raw, 0) {
		return result, fmt.Errorf("mapping input contains NUL")
	}
	scanner, err := newRecord(raw)
	if err != nil {
		return result, err
	}
	scanner.ctx = ctx
	for _, field := range c.fields {
		if field.field.KVProfile != "" {
			scanner.kvieScans = make(map[kvieCacheKey]kvieScan)
			result.kvieScans = scanner.kvieScans
			break
		}
	}
	outputBytes := 0
	for _, f := range c.fields {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		value, evidence, found, err := scanner.extract(f, prior)
		if errors.Is(err, ErrSourceAbsent) {
			found = false
			err = nil
		}
		if err != nil {
			return result, fmt.Errorf("field %q: %w", f.field.Name, err)
		}
		if !found || value == nil {
			var omit bool
			value, omit, evidence, err = resolveAbsence(f.field)
			if err != nil {
				return result, fmt.Errorf("field %q: %w", f.field.Name, err)
			}
			if omit {
				continue
			}
		}
		if values, ok := value.(map[string]any); ok && found {
			f.field.editKeyValues(values)
		}
		omitField, operationsExecuted := false, 0
		if trace {
			appendTrace(&result, f.field.Name, 0, "source", raw, value, evidence, &traceBudget)
		}
		if value != nil {
			for i, op := range f.field.Operations {
				if err := ctx.Err(); err != nil {
					return result, err
				}
				operationsExecuted = i + 1
				input := value
				value, err = applyOperation(value, op, f.ops[i], prior)
				if trace {
					appendTrace(&result, f.field.Name, i+1, op.Op, input, value, evidence, &traceBudget)
				}
				if err != nil {
					if !errors.Is(err, ErrSourceAbsent) {
						return result, fmt.Errorf("field %q operation %d: %w", f.field.Name, i+1, err)
					}
					value, omitField, evidence, err = resolveAbsence(f.field)
					if err != nil {
						return result, fmt.Errorf("field %q operation %d: %w", f.field.Name, i+1, err)
					}
					// A fallback replaces the missing final field, not the input to
					// the remaining selector operations. It is still type-validated.
					break
				}
				if n := valueSize(value); n > MaxInputBytes {
					return result, fmt.Errorf("field %q transformed value exceeds bounds", f.field.Name)
				}
			}
		}
		if omitField {
			continue
		}
		if text, ok := value.(string); ok && text == "" {
			switch f.field.EmptyPolicy {
			case "missing":
				value, omitField, evidence, err = resolveAbsence(f.field)
				if err != nil {
					return result, fmt.Errorf("field %q: %w", f.field.Name, err)
				}
				if omitField {
					continue
				}
			case "null":
				value = nil
			case "reject":
				return result, fmt.Errorf("field %q rejects empty values", f.field.Name)
			}
		}
		if value != nil {
			if f.field.Transform != "none" {
				value, err = applyOperation(value, Operation{Op: f.field.Transform}, nil, result.Values)
				if err != nil {
					return result, err
				}
			}
			value, err = convert(value, f.field)
			if err != nil {
				return result, fmt.Errorf("field %q: %w", f.field.Name, err)
			}
			if f.customPattern != nil && !f.customPattern.MatchString(stringValue(value)) {
				return result, fmt.Errorf("field %q does not satisfy custom type", f.field.Name)
			}
			if err = checkRules(value, f.field.Validation); err != nil {
				return result, fmt.Errorf("field %q: %w", f.field.Name, err)
			}
		}
		evidence.Operations = operationsExecuted
		if f.field.Method == "constant" || f.field.Method == "derived" || len(f.field.Operations) > 0 || f.field.Transform != "none" {
			evidence.Source = false
			evidence.Description = "reviewed mapping: " + evidence.Description
		}
		outputBytes += len(f.field.Name) + valueSize(value)
		if outputBytes > MaxInputBytes {
			return result, fmt.Errorf("mapping output exceeds bounded aggregate size")
		}
		prior[f.field.Name] = value
		hidden := f.field.OutputPolicy == "omit" || f.field.OutputPolicy == "intermediate"
		for _, op := range f.field.Operations {
			if op.Op == "omit" || op.Op == "exclude" {
				hidden = true
			}
			if op.RemoveSources && mergeCondition(op, prior) {
				for _, name := range op.Fields {
					delete(result.Values, name)
				}
			}
		}
		if !hidden {
			if f.field.OutputPolicy == "flatten" {
				object, ok := value.(map[string]any)
				if !ok {
					return result, fmt.Errorf("flatten output requires an object")
				}
				keys := make([]string, 0, len(object))
				for key := range object {
					keys = append(keys, key)
				}
				sort.Strings(keys)
				for _, key := range keys {
					if !fieldName.MatchString(key) || strings.HasPrefix(key, "_") || contains(ReservedNames, strings.ToLower(key)) {
						return result, fmt.Errorf("dynamic key %q must be a non-reserved output identifier", key)
					}
					if _, exists := prior[key]; exists {
						return result, fmt.Errorf("dynamic key %q collides with a mapped field", key)
					}
					if len(result.Values) >= c.outputFieldLimit {
						return result, fmt.Errorf("dynamic output exceeds field limit")
					}
					result.Values[key], prior[key] = object[key], object[key]
					keyEvidence := evidence
					keyEvidence.keySpans = nil
					if spans := evidence.keySpans[key]; len(spans) > 0 {
						keyEvidence.Start, keyEvidence.End = spans[0].Start, spans[0].End
						keyEvidence.Spans = spans
					}
					result.Evidence[key] = keyEvidence
					result.DynamicFields = append(result.DynamicFields, key)
				}
			} else {
				if _, exists := result.Values[f.field.Name]; exists {
					return result, fmt.Errorf("mapped field %q collides with a dynamic key", f.field.Name)
				}
				result.Values[f.field.Name] = value
				if object, ok := value.(map[string]any); ok && evidence.Source && len(evidence.keySpans) > 0 {
					children := make(map[string]Evidence, len(evidence.keySpans))
					for key, spans := range evidence.keySpans {
						if _, emitted := object[key]; emitted && len(spans) > 0 {
							children[key] = Evidence{Start: spans[0].Start, End: spans[0].End, Source: true, Spans: spans}
						}
					}
					if len(children) > 0 {
						if result.NestedEvidence == nil {
							result.NestedEvidence = make(map[string]map[string]Evidence)
						}
						result.NestedEvidence[f.field.Name] = children
					}
				}
			}
		}
		evidence.keySpans = nil
		result.Evidence[f.field.Name] = evidence
	}
	return result, nil
}

func resolveAbsence(f Field) (any, bool, Evidence, error) {
	evidence := Evidence{Start: -1, End: -1}
	if f.Policy == "default" {
		evidence.Description = "explicit configured default"
		return f.DefaultValue, false, evidence, nil
	}
	if f.NullPolicy == "reject" {
		return nil, false, evidence, fmt.Errorf("null or missing value is rejected by policy")
	}
	if f.Policy == "nullable" || f.NullPolicy == "preserve_null" {
		evidence.Description = "explicit nullable absence"
		return nil, false, evidence, nil
	}
	if f.Policy == "optional" {
		return nil, true, evidence, nil
	}
	return nil, false, evidence, fmt.Errorf("required source value is absent")
}

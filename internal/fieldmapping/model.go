// Package fieldmapping implements NLI's bounded Native Field Intelligence (NFI)
// mapping plans. Plans are data, never executable expressions. The same compiled
// plan is used by controller previews and the client parser-policy executor.
package fieldmapping

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"regexp"
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

var Methods = []string{"position", "delimiter", "character", "byte", "line", "key_value", "regex", "json_path", "constant", "derived"}
var Operations = []string{"split", "select", "substring", "character", "byte", "line", "key_value", "replace", "remove", "trim", "lowercase", "uppercase", "prefix", "suffix", "add", "strip_prefix", "strip_suffix", "regex_extract", "regex_replace", "join", "merge"}
var Types = []string{"string", "integer", "number", "boolean", "timestamp", "ip", "port", "object", "array"}
var ReservedNames = []string{"raw", "nli", "meta", "source", "envelope", "parser_id", "attempt_count", "ordered_attempts", "parse_telemetry"}
var fieldName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]{0,63}$`)

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

type Field struct {
	Name            string      `json:"name"`
	Label           string      `json:"label,omitempty"`
	Type            string      `json:"type,omitempty"`
	Policy          string      `json:"policy,omitempty"`
	Method          string      `json:"method"`
	Source          string      `json:"source,omitempty"`
	SourceEnd       string      `json:"source_end,omitempty"`
	Delimiter       string      `json:"delimiter,omitempty"`
	Part            string      `json:"part,omitempty"`
	Confirmed       bool        `json:"confirmed,omitempty"`
	Transform       string      `json:"transform,omitempty"`
	TimestampFormat string      `json:"timestamp_format,omitempty"`
	Timezone        string      `json:"timezone,omitempty"`
	TimezoneName    string      `json:"timezone_name,omitempty"`
	DefaultValue    string      `json:"default_value,omitempty"`
	Validation      string      `json:"validation,omitempty"`
	NullPolicy      string      `json:"null_policy,omitempty"`
	KVSeparator     string      `json:"kv_separator,omitempty"`
	PairDelimiters  string      `json:"pair_delimiters,omitempty"`
	DuplicatePolicy string      `json:"duplicate_policy,omitempty"`
	JoinSeparator   string      `json:"join_separator,omitempty"`
	QuoteAware      *bool       `json:"quote_aware,omitempty"`
	Operations      []Operation `json:"operations,omitempty"`
	// These bounded display-only values allow existing browser drafts to be
	// round-tripped without permitting arbitrary configuration properties.
	ID         string `json:"id,omitempty"`
	Status     string `json:"status,omitempty"`
	Preview    string `json:"preview,omitempty"`
	Evidence   string `json:"evidence,omitempty"`
	Confidence string `json:"confidence,omitempty"`
}

type Operation struct {
	Op         string   `json:"op"`
	Delimiter  string   `json:"delimiter,omitempty"`
	Part       int      `json:"part,omitempty"`
	Start      int      `json:"start,omitempty"`
	End        int      `json:"end,omitempty"`
	Value      string   `json:"value,omitempty"`
	Find       string   `json:"find,omitempty"`
	Replace    string   `json:"replace,omitempty"`
	Pattern    string   `json:"pattern,omitempty"`
	Group      string   `json:"group,omitempty"`
	Fields     []string `json:"fields,omitempty"`
	Separator  string   `json:"separator,omitempty"`
	QuoteAware *bool    `json:"quote_aware,omitempty"`
}

type Evidence struct {
	Start       int    `json:"start"`
	End         int    `json:"end"`
	Source      bool   `json:"source"`
	Description string `json:"description"`
	Operations  int    `json:"operations"`
}

type Result struct {
	Values   map[string]any      `json:"values"`
	Evidence map[string]Evidence `json:"evidence"`
}

type compiledField struct {
	field   Field
	pattern *regexp.Regexp
	ops     []*regexp.Regexp
}

type Compiled struct{ fields []compiledField }

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
	if plan.Version != 1 {
		return nil, fmt.Errorf("mapping plan version must be 1")
	}
	if len(plan.Fields) == 0 || len(plan.Fields) > MaxFields {
		return nil, fmt.Errorf("mapping needs 1..%d fields", MaxFields)
	}
	compiled := &Compiled{}
	seen := map[string]bool{}
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
		if !contains([]string{"string", "integer", "int", "number", "float", "boolean", "bool", "timestamp", "time", "ip", "port", "object", "array"}, f.Type) {
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
			re, err := boundedRegex(f.Source)
			if err != nil {
				return nil, fmt.Errorf("field %q: %w", f.Name, err)
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
		compiled.fields = append(compiled.fields, cf)
		seen[f.Name] = true
	}
	return compiled, nil
}

// Execute applies one immutable plan to one bounded record. Any invalid required
// value rejects the record instead of emitting a partially successful payload.
func (c *Compiled) Execute(ctx context.Context, raw string) (Result, error) {
	result := Result{Values: map[string]any{}, Evidence: map[string]Evidence{}}
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
	outputBytes := 0
	for _, f := range c.fields {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		value, evidence, found, err := scanner.extract(f, result.Values)
		if err != nil {
			return result, fmt.Errorf("field %q: %w", f.field.Name, err)
		}
		if !found || value == nil {
			if f.field.Policy == "default" {
				value = f.field.DefaultValue
				found = true
				evidence = Evidence{Start: -1, End: -1, Description: "explicit configured default"}
			} else if f.field.NullPolicy == "reject" {
				return result, fmt.Errorf("field %q: null or missing value is rejected by policy", f.field.Name)
			} else if f.field.Policy == "nullable" || f.field.NullPolicy == "preserve_null" {
				value = nil
				found = true
				evidence = Evidence{Start: -1, End: -1, Description: "explicit nullable absence"}
			} else if f.field.Policy == "optional" && f.field.NullPolicy != "reject" {
				continue
			} else {
				return result, fmt.Errorf("field %q: required source value is absent", f.field.Name)
			}
		}
		if value != nil {
			for i, op := range f.field.Operations {
				if err := ctx.Err(); err != nil {
					return result, err
				}
				value, err = applyOperation(value, op, f.ops[i], result.Values)
				if err != nil {
					return result, fmt.Errorf("field %q operation %d: %w", f.field.Name, i+1, err)
				}
				if n := valueSize(value); n > MaxInputBytes {
					return result, fmt.Errorf("field %q transformed value exceeds bounds", f.field.Name)
				}
			}
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
			if err = checkRules(value, f.field.Validation); err != nil {
				return result, fmt.Errorf("field %q: %w", f.field.Name, err)
			}
		}
		evidence.Operations = len(f.field.Operations)
		if f.field.Method == "constant" || f.field.Method == "derived" || len(f.field.Operations) > 0 || f.field.Transform != "none" {
			evidence.Source = false
			evidence.Description = "reviewed mapping: " + evidence.Description
		}
		outputBytes += len(f.field.Name) + valueSize(value)
		if outputBytes > MaxInputBytes {
			return result, fmt.Errorf("mapping output exceeds bounded aggregate size")
		}
		result.Values[f.field.Name] = value
		result.Evidence[f.field.Name] = evidence
	}
	return result, nil
}

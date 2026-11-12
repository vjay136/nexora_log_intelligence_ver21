package fieldmapping

import (
	"crypto/sha256"
	"crypto/sha512"
	"encoding/json"
	"fmt"
	"net"
	"net/mail"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

type splitPart = pair

func validStrings(v any, depth int) bool {
	if depth > MaxDepth {
		return false
	}
	switch x := v.(type) {
	case string:
		return utf8.ValidString(x)
	case []any:
		for _, v := range x {
			if !validStrings(v, depth+1) {
				return false
			}
		}
	case map[string]any:
		for k, v := range x {
			if !utf8.ValidString(k) || !validStrings(v, depth+1) {
				return false
			}
		}
	}
	return true
}

func cloneSelection(s Selection) (Selection, error) {
	if !contains([]string{"", "first", "last", "nth", "all"}, s.Occurrence) || !contains([]string{"", "left", "right"}, s.Direction) || !contains([]string{"", "character", "byte", "word", "token", "line"}, s.Unit) || !contains([]string{"", "whitespace", "delimiter", "mixed", "regex"}, s.TokenMode) {
		return s, fmt.Errorf("invalid selection policy")
	}
	if s.Count < 0 || s.Count > MaxTokens || s.OccurrenceIndex < 0 || s.OccurrenceIndex > MaxTokens || s.Occurrence == "nth" && s.OccurrenceIndex < 1 || len(s.Indices) > MaxFields || len(s.TokenSeparators) > 32 || len(s.BoundaryStart) > 4096 || len(s.BoundaryEnd) > 4096 {
		return s, fmt.Errorf("selection exceeds bounds")
	}
	for _, n := range s.Indices {
		if n < 1 || n > MaxTokens {
			return s, fmt.Errorf("selection indices must be 1-based")
		}
	}
	for _, sep := range s.TokenSeparators {
		if len(sep) == 0 || len(sep) > 64 {
			return s, fmt.Errorf("invalid token separator")
		}
	}
	s.Indices = append([]int(nil), s.Indices...)
	s.TokenSeparators = append([]string(nil), s.TokenSeparators...)
	if s.TokenMode == "delimiter" || s.TokenMode == "mixed" {
		if len(s.TokenSeparators) == 0 {
			return s, fmt.Errorf("token mode needs separators")
		}
	}
	return s, nil
}
func cloneKeys(k KeyOptions) (KeyOptions, error) {
	if len(k.AssignmentSeparators) > 16 || len(k.KeyAliases) > MaxFields || len(k.KeyInclude) > MaxFields || len(k.KeyExclude) > MaxFields || len(k.KeyPrefix) > 64 || len(k.KeySuffix) > 64 {
		return k, fmt.Errorf("key options exceed bounds")
	}
	for _, s := range k.AssignmentSeparators {
		if len(s) == 0 || len(s) > 8 {
			return k, fmt.Errorf("assignment separator must contain 1..8 bytes")
		}
	}
	for a, b := range k.KeyAliases {
		if len(a) > 128 || len(b) > 128 || a == "" || b == "" {
			return k, fmt.Errorf("invalid key alias")
		}
	}
	for _, list := range [][]string{k.KeyInclude, k.KeyExclude} {
		for _, s := range list {
			if s == "" || len(s) > 128 {
				return k, fmt.Errorf("invalid key filter")
			}
		}
	}
	aliases := map[string]string{}
	for a, b := range k.KeyAliases {
		aliases[a] = b
	}
	k.KeyAliases = aliases
	k.AssignmentSeparators = append([]string(nil), k.AssignmentSeparators...)
	sort.SliceStable(k.AssignmentSeparators, func(i, j int) bool { return len(k.AssignmentSeparators[i]) > len(k.AssignmentSeparators[j]) })
	k.KeyInclude = append([]string(nil), k.KeyInclude...)
	k.KeyExclude = append([]string(nil), k.KeyExclude...)
	return k, nil
}
func compileExtended(cf *compiledField, seen map[string]bool) error {
	f := &cf.field
	var err error
	if f.DuplicatePolicy != "" && !contains([]string{"reject", "first", "last", "array"}, f.DuplicatePolicy) {
		return fmt.Errorf("invalid duplicate policy")
	}
	if f.Selection, err = cloneSelection(f.Selection); err != nil {
		return err
	}
	if f.KeyOptions, err = cloneKeys(f.KeyOptions); err != nil {
		return err
	}
	if len(f.KeyValueEdits) > 0 {
		if f.Method != "kv_all" || len(f.KeyValueEdits) > MaxFields {
			return fmt.Errorf("key value edits require kv_all and must fit the field limit")
		}
		edited := make(map[string]bool, len(f.KeyValueEdits))
		for _, edit := range f.KeyValueEdits {
			if strings.TrimSpace(edit.Key) != edit.Key || len(edit.Key) == 0 || len(edit.Key) > 128 || edited[edit.Key] {
				return fmt.Errorf("key value edits require distinct nonempty output key names")
			}
			if !contains([]string{"omit", "blank", "replace"}, edit.Action) || len(edit.Value) > 4096 || (edit.Action != "replace" && edit.Value != "") {
				return fmt.Errorf("invalid key value edit for %q", edit.Key)
			}
			edited[edit.Key] = true
		}
		f.KeyValueEdits = append([]KeyValueEdit(nil), f.KeyValueEdits...)
	}
	if !contains([]string{"", "include", "omit", "intermediate", "flatten"}, f.OutputPolicy) || !contains([]string{"", "preserve", "missing", "null", "reject"}, f.EmptyPolicy) {
		return fmt.Errorf("invalid output or empty policy")
	}
	if f.OutputPolicy == "flatten" && (f.Method != "kv_all" || f.Type != "object") {
		return fmt.Errorf("flatten requires kv_all with object type")
	}
	if !contains([]string{"", "auto", "configured"}, f.KVProfile) || f.KVProfile != "" && f.Method != "kv_all" && f.Method != "key_value" {
		return fmt.Errorf("kv_profile requires key/value extraction and must be auto or configured")
	}
	namedProfile := f.Method == "key_value" && f.KVProfile != ""
	if f.UsesAdvancedKV() {
		if f.Method != "kv_all" || f.KVProfile != "configured" || !f.RequireComplete || !f.KVLiteralBoundaries {
			return fmt.Errorf("advanced KV boundaries require a configured dynamic KV field with complete coverage")
		}
		for _, name := range []string{f.KVBareField, f.KVEmptyKeyField} {
			if name != "" && (!fieldName.MatchString(name) || strings.HasPrefix(name, "_") || contains(ReservedNames, strings.ToLower(name))) {
				return fmt.Errorf("unnamed segment output %q must be a non-reserved field identifier", name)
			}
			if name == "" {
				continue
			}
			if _, exists := f.KeyAliases[name]; exists {
				return fmt.Errorf("unnamed segment output %q cannot also be aliased", name)
			}
			for _, alias := range f.KeyAliases {
				if alias == name {
					return fmt.Errorf("unnamed segment output %q collides with a key alias", name)
				}
			}
		}
	}
	if f.KVMaxPairs < 0 || f.KVMaxPairs > MaxAdvancedKVPairs {
		return fmt.Errorf("advanced KV maximum pairs must be between 1 and %d", MaxAdvancedKVPairs)
	}
	if f.KVHeaderPattern != "" {
		if f.Source != "" && f.Source != "raw" || f.SourceEnd != "" || !strings.HasPrefix(f.KVHeaderPattern, "^") {
			return fmt.Errorf("KV header pattern must start at record start and replace positional KV selectors")
		}
		if cf.kvHeader == nil {
			cf.kvHeader, err = boundedRegex(f.KVHeaderPattern)
			if err != nil {
				return err
			}
		}
		if cf.kvHeader.MatchString("") {
			return fmt.Errorf("KV header pattern must consume a nonempty header")
		}
	}
	if len(f.KVInputEdits) > 0 {
		if f.Method != "kv_all" || f.KVProfile != "configured" || !f.RequireComplete || len(f.KVInputEdits) > 16 {
			return fmt.Errorf("KV input edits require a configured dynamic KV field with complete coverage and at most 16 edits")
		}
		for _, edit := range f.KVInputEdits {
			if !contains([]string{"replace", "strip_prefix", "strip_suffix"}, edit.Action) || edit.Find == "" ||
				len(edit.Find) > 4096 || len(edit.Replace) > 4096 || edit.Action != "replace" && edit.Replace != "" {
				return fmt.Errorf("invalid KV input edit: provide literal text and choose replace, strip prefix, or strip suffix")
			}
		}
		f.KVInputEdits = append([]KVInputEdit(nil), f.KVInputEdits...)
	}
	if f.KVNestedObjects && (f.KVProfile == "" || f.Method == "key_value" && f.Type != "object") {
		return fmt.Errorf("nested KV objects require a KV profile and object output for a named key")
	}
	if f.KVStart != "" && !namedProfile {
		return fmt.Errorf("kv_start requires a named key_value field with a KV profile")
	}
	if f.KVProfile != "configured" && (len(f.KVPairSeparators) > 0 || f.KVQuotes != "" || f.KVEscape != "" || f.KVBrackets != nil || f.KVKeyPattern != "" || f.KVRejectEmpty) {
		return fmt.Errorf("custom KV lexical options require a configured profile")
	}
	if f.EndOfLine && f.Method != "kv_all" && !namedProfile {
		return fmt.Errorf("end_of_line requires kv_all or a profiled key_value field")
	}
	source := f.Source
	if namedProfile {
		source = f.KVStart
	}
	if (f.Method == "kv_all" || namedProfile) && source != "" && source != "raw" {
		start, err := index(source, 1)
		if err != nil {
			return err
		}
		if f.SourceEnd != "" {
			end, err := index(f.SourceEnd, start)
			if err != nil || end < start {
				return fmt.Errorf("invalid key/value source range")
			}
		}
	}
	if f.KVProfile == "configured" {
		assignments, delimiters := keySettings(*f)
		cf.kvie, err = compileKVIEProfile(assignments, delimiters, *f)
		if err == nil && len(f.KVInputEdits) > 0 {
			encoded, _ := json.Marshal(f.KVInputEdits)
			cf.kvie.signature += string(encoded)
		}
		if err != nil {
			return err
		}
	}
	if len(f.Description) > 4096 || len(f.Example) > 4096 || len(f.CustomType) > 128 {
		return fmt.Errorf("type/display metadata exceeds bounds")
	}
	if f.TokenPattern != "" || f.TokenMode == "regex" {
		cf.tokenPattern, err = boundedRegex(f.TokenPattern)
		if err != nil {
			return err
		}
	}
	if f.Type == "custom" {
		cf.customPattern, err = boundedRegex("^(?:" + f.CustomPattern + ")$")
		if f.CustomPattern == "" {
			return fmt.Errorf("custom type needs custom_pattern")
		}
		if err != nil {
			return err
		}
	}
	switch f.Method {
	case "field":
		if !seen[f.Source] {
			return fmt.Errorf("field source must reference an earlier field")
		}
	case "word", "token":
		start, e := index(f.Source, 1)
		if e != nil {
			return e
		}
		end, e := index(f.SourceEnd, start)
		if e != nil || end < start {
			return fmt.Errorf("invalid token range")
		}
	case "before", "after", "between":
		if f.BoundaryStart == "" || f.Method == "between" && f.BoundaryEnd == "" {
			return fmt.Errorf("boundary selector needs explicit boundaries")
		}
	case "regex_all", "regex_groups", "token_match":
		cf.pattern, err = boundedRegex(f.Source)
		if err != nil {
			return err
		}
		if f.Method == "regex_all" && f.Part != "" {
			if _, err = regexGroup(cf.pattern, f.Part); err != nil {
				return err
			}
		}
	}
	for i := range f.Operations {
		op := &f.Operations[i]
		if op.Selection, err = cloneSelection(op.Selection); err != nil {
			return err
		}
		if op.KeyOptions, err = cloneKeys(op.KeyOptions); err != nil {
			return err
		}
		if op.Limit < 0 || op.Limit > MaxTokens || len(op.ConditionField) > 64 || len(op.ConditionValue) > 4096 || len(op.ElseValue) > 4096 {
			return fmt.Errorf("operation exceeds bounds")
		}
		if op.TokenPattern != "" || op.TokenMode == "regex" {
			op.tokenPattern, err = boundedRegex(op.TokenPattern)
			if err != nil {
				return err
			}
		}
		if contains([]string{"split_regex", "token_match"}, op.Op) {
			cf.ops[i], err = boundedRegex(op.Pattern)
			if err != nil {
				return err
			}
		}
		if contains([]string{"before", "after", "between"}, op.Op) && (op.BoundaryStart == "" || op.Op == "between" && op.BoundaryEnd == "") {
			return fmt.Errorf("operation needs explicit boundaries")
		}
		if op.Op == "conditional_merge" {
			if !seen[op.ConditionField] || !contains([]string{"exists", "equals", "not_empty"}, op.ConditionOperator) {
				return fmt.Errorf("conditional merge needs an earlier condition field and supported operator")
			}
		}
		if op.Op == "hash" && !contains([]string{"", "sha256", "sha512"}, op.Algorithm) {
			return fmt.Errorf("unsupported hash algorithm")
		}
		if op.Op == "normalize" && !contains([]string{"", "whitespace", "trim", "lowercase", "uppercase"}, op.Mode) {
			return fmt.Errorf("unsupported normalization mode")
		}
	}
	return nil
}
func extendedMethod(f Field) bool {
	return contains([]string{"raw", "field", "word", "token", "token_match", "before", "after", "between", "count", "kv_all", "regex_all", "regex_groups"}, f.Method) || f.Method == "key_value" && (len(f.AssignmentSeparators) > 0 || len(f.KeyAliases) > 0 || f.KVProfile != "")
}
func regexGroup(re *regexp.Regexp, name string) (int, error) {
	if name == "" {
		return 0, nil
	}
	n, err := strconv.Atoi(name)
	if err != nil {
		n = re.SubexpIndex(name)
	}
	if n < 0 || n > re.NumSubexp() {
		return 0, fmt.Errorf("regex group is absent")
	}
	return n, nil
}
func (r *record) extractExtended(cf compiledField, prior map[string]any) (any, Evidence, bool, error) {
	f := cf.field
	e := Evidence{Start: -1, End: -1, Description: f.Method + ":" + f.Source}
	if f.Method == "field" {
		v, ok := prior[f.Source]
		return v, e, ok, nil
	}
	if f.Method == "raw" {
		e.Start = 0
		e.End = len(r.raw)
		e.Source = true
		return r.raw, e, true, nil
	}
	if f.Method == "kv_all" || f.Method == "key_value" {
		if f.KVProfile != "" {
			return r.extractKVIE(cf, e)
		}
		if f.Method == "kv_all" {
			selected, start, end, found, err := r.keySource(cf)
			if err != nil || !found {
				return nil, e, found, err
			}
			e.Start, e.End, e.Source = start, end, true
			return extractKeys(selected, f, e)
		}
		return extractKeys(r.raw, f, e)
	}
	if f.Method == "regex_all" || f.Method == "regex_groups" || f.Method == "token_match" {
		limit := MaxTokens / (cf.pattern.NumSubexp() + 1)
		matches := cf.pattern.FindAllStringSubmatchIndex(r.raw, limit+1)
		if len(matches) > limit {
			return nil, e, false, fmt.Errorf("regex match limit exceeded")
		}
		if len(matches) == 0 {
			return nil, e, false, nil
		}
		if f.Method == "regex_groups" {
			v := map[string]any{}
			for i, name := range cf.pattern.SubexpNames() {
				if i > 0 && name != "" && matches[0][2*i] >= 0 {
					v[name] = r.raw[matches[0][2*i]:matches[0][2*i+1]]
				}
			}
			return v, e, true, nil
		}
		group, err := regexGroup(cf.pattern, f.Part)
		if err != nil {
			return nil, e, false, err
		}
		values := []any{}
		for _, m := range matches {
			a, b := m[2*group], m[2*group+1]
			if a >= 0 {
				values = append(values, r.raw[a:b])
				if len(e.Spans) < MaxFields {
					e.Spans = append(e.Spans, Span{a, b})
				}
			}
		}
		return values, e, true, nil
	}
	if contains([]string{"before", "after", "between"}, f.Method) {
		v, spans, err := boundaries(r.raw, f.Method, f.Selection)
		e.Spans = spans
		if len(spans) > 0 {
			e.Start = spans[0].Start
			e.End = spans[0].End
			e.Source = true
		}
		return v, e, err == nil, err
	}
	parts, err := units(r.raw, f.Selection, cf.tokenPattern, f.QuoteAware)
	if err != nil {
		return nil, e, false, err
	}
	if f.Method == "count" {
		return len(parts), e, true, nil
	}
	start, _ := index(f.Source, 1)
	end, _ := index(f.SourceEnd, start)
	if len(f.Indices) > 0 {
		out := []any{}
		for _, n := range f.Indices {
			if f.Direction == "right" {
				n = len(parts) - n + 1
			}
			if n < 1 || n > len(parts) {
				return nil, e, false, nil
			}
			out = append(out, parts[n-1].value)
			e.Spans = append(e.Spans, Span{parts[n-1].start, parts[n-1].end})
		}
		return out, e, true, nil
	}
	if f.Direction == "right" {
		start, end = len(parts)-end+1, len(parts)-start+1
	}
	if start < 1 || end > len(parts) {
		return nil, e, false, nil
	}
	e.Start = parts[start-1].start
	e.End = parts[end-1].end
	e.Source = true
	if start == end {
		return parts[start-1].value, e, true, nil
	}
	return r.raw[e.Start:e.End], e, true, nil
}

func keySettings(f Field) ([]string, string) {
	separators := f.AssignmentSeparators
	if len(separators) == 0 {
		sep := f.KVSeparator
		if sep == "" {
			sep = "="
		}
		separators = []string{sep}
	}
	delimiters := f.PairDelimiters
	if delimiters == "" {
		delimiters = ":;,| \t\r\n"
	}
	return separators, delimiters
}

func extractKeys(raw string, f Field, e Evidence) (any, Evidence, bool, error) {
	separators, delimiters := keySettings(f)
	pairs, err := scanPairsManyMode(raw, separators, delimiters, f.RequireComplete)
	if err != nil {
		return nil, e, false, err
	}
	return projectKeys(pairs, f, e)
}

func projectKeys(pairs map[string][]pair, f Field, e Evidence) (any, Evidence, bool, error) {
	normalized := map[string][]pair{}
	if f.RequireComplete && len(pairs) == 0 {
		return nil, e, false, fmt.Errorf("key/value tail contains no pairs")
	}
	for key, values := range pairs {
		canonical := key
		if alias, ok := f.KeyAliases[key]; ok {
			canonical = alias
		}
		normalized[canonical] = append(normalized[canonical], values...)
	}
	for key := range normalized {
		sort.Slice(normalized[key], func(i, j int) bool { return normalized[key][i].start < normalized[key][j].start })
	}
	selectValues := func(values []pair) (any, error) {
		if len(values) > 1 && (f.DuplicatePolicy == "" || f.DuplicatePolicy == "reject") {
			return nil, fmt.Errorf("duplicate key rejected")
		}
		if f.DuplicatePolicy == "array" {
			v := []any{}
			for _, p := range values {
				v = append(v, p.value)
			}
			return v, nil
		}
		n := 0
		if f.DuplicatePolicy == "last" {
			n = len(values) - 1
		}
		return values[n].value, nil
	}
	if f.Method == "key_value" {
		key := f.Source
		if alias, ok := f.KeyAliases[key]; ok {
			key = alias
		}
		values := normalized[key]
		if len(values) == 0 {
			return nil, e, false, nil
		}
		v, err := selectValues(values)
		e.Start = values[0].start
		e.End = values[0].end
		e.Source = true
		if f.KVProfile != "" {
			for _, value := range values {
				e.Spans = append(e.Spans, Span{value.start, value.end})
			}
		}
		return v, e, true, err
	}
	out := map[string]any{}
	keys := make([]string, 0, len(normalized))
	for key := range normalized {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		values := normalized[key]
		if len(f.KeyInclude) > 0 && !contains(f.KeyInclude, key) || contains(f.KeyExclude, key) {
			continue
		}
		var v any
		if key != "" && (key == f.KVBareField || key == f.KVEmptyKeyField) {
			segments := make([]any, 0, len(values))
			for _, value := range values {
				segments = append(segments, value.value)
			}
			v = segments
		} else {
			var err error
			v, err = selectValues(values)
			if err != nil {
				return nil, e, false, err
			}
		}
		out[f.KeyPrefix+key+f.KeySuffix] = v
	}
	return out, e, true, nil
}

func (f Field) editKeyValues(values map[string]any) {
	for _, edit := range f.KeyValueEdits {
		if _, found := values[edit.Key]; !found {
			continue
		}
		switch edit.Action {
		case "omit":
			delete(values, edit.Key)
		case "blank":
			values[edit.Key] = ""
		case "replace":
			values[edit.Key] = edit.Value
		}
	}
}

func boundaries(text, method string, s Selection) (any, []Span, error) {
	spans := []Span{}
	pos := 0
	for pos <= len(text) {
		i := strings.Index(text[pos:], s.BoundaryStart)
		if i < 0 {
			break
		}
		a := pos + i
		b := a + len(s.BoundaryStart)
		end := b
		switch method {
		case "before":
			start := 0
			if s.PreserveBoundary {
				end = b
			} else {
				end = a
			}
			spans = append(spans, Span{start, end})
		case "after":
			start := b
			if s.PreserveBoundary {
				start = a
			}
			spans = append(spans, Span{start, len(text)})
		case "between":
			j := strings.Index(text[b:], s.BoundaryEnd)
			if j < 0 {
				break
			}
			end = b + j
			start := b
			if s.PreserveBoundary {
				start = a
				end += len(s.BoundaryEnd)
			}
			spans = append(spans, Span{start, end})
			b = b + j + len(s.BoundaryEnd)
		}
		pos = b
		if len(spans) > MaxTokens {
			return nil, nil, fmt.Errorf("boundary selection exceeds bounds")
		}
	}
	if len(spans) == 0 {
		return nil, nil, ErrSourceAbsent
	}
	if s.Occurrence == "all" {
		out := []any{}
		total := 0
		for _, p := range spans {
			total += p.End - p.Start
			if total > MaxInputBytes {
				return nil, nil, fmt.Errorf("boundary output exceeds aggregate bounds")
			}
			out = append(out, text[p.Start:p.End])
		}
		return out, spans, nil
	}
	i := 0
	if s.Occurrence == "last" {
		i = len(spans) - 1
	}
	if s.Occurrence == "nth" {
		i = s.OccurrenceIndex - 1
	}
	if i >= len(spans) {
		return nil, nil, ErrSourceAbsent
	}
	p := spans[i]
	return text[p.Start:p.End], []Span{p}, nil
}

// units preserves exact source offsets while decoding quoted single tokens.
func units(text string, s Selection, re *regexp.Regexp, quote *bool) ([]splitPart, error) {
	if s.Unit == "character" || s.Unit == "byte" {
		out := []splitPart{}
		for i := 0; i < len(text); {
			n := 1
			if s.Unit == "character" {
				_, n = utf8.DecodeRuneInString(text[i:])
			}
			out = append(out, splitPart{text[i : i+n], i, i + n})
			i += n
			if len(out) > MaxTokens {
				return nil, fmt.Errorf("unit count exceeds limit")
			}
		}
		return out, nil
	}
	if s.Unit == "line" {
		return splitParts(text, "\n", false)
	}
	if s.TokenMode == "regex" {
		return regexParts(text, re, quote)
	}
	seps := s.TokenSeparators
	if s.TokenMode == "delimiter" && len(seps) == 1 {
		return splitParts(text, seps[0], quote == nil || *quote)
	}
	out := []splitPart{}
	start := 0
	q := byte(0)
	aware := quote == nil || *quote
	flush := func(end int) error {
		if end <= start {
			return nil
		}
		parts, err := splitParts(text[start:end], "\x00", aware)
		if err != nil {
			return err
		}
		v := parts[0]
		out = append(out, splitPart{v.value, start, end})
		if len(out) > MaxTokens {
			return fmt.Errorf("token count exceeds limit")
		}
		return nil
	}
	for i := 0; i < len(text); {
		c := text[i]
		if aware && c == '\\' && i+1 < len(text) {
			i += 2
			continue
		}
		if aware && q != 0 {
			if c == q {
				if i+1 < len(text) && text[i+1] == q {
					i += 2
					continue
				}
				q = 0
			}
			i++
			continue
		}
		if aware && (c == '\'' || c == '"') && (i == start) {
			q = c
			i++
			continue
		}
		n := 0
		if s.TokenMode == "" || s.TokenMode == "whitespace" || s.TokenMode == "mixed" {
			r, size := utf8.DecodeRuneInString(text[i:])
			if unicode.IsSpace(r) {
				n = size
			}
		}
		if s.TokenMode == "delimiter" || s.TokenMode == "mixed" {
			for _, sep := range seps {
				if strings.HasPrefix(text[i:], sep) && len(sep) > n {
					n = len(sep)
				}
			}
		}
		if n > 0 {
			if err := flush(i); err != nil {
				return nil, err
			}
			i += n
			start = i
		} else {
			_, n = utf8.DecodeRuneInString(text[i:])
			i += n
		}
	}
	if q != 0 {
		return nil, fmt.Errorf("unterminated quoted token")
	}
	if err := flush(len(text)); err != nil {
		return nil, err
	}
	return out, nil
}
func regexParts(text string, re *regexp.Regexp, quote *bool) ([]splitPart, error) {
	if re == nil {
		return nil, fmt.Errorf("token regex is absent")
	}
	locations := re.FindAllStringIndex(text, MaxTokens+1)
	if len(locations) > MaxTokens {
		return nil, fmt.Errorf("regex token count exceeds limit")
	}
	out := []splitPart{}
	start := 0
	allowed := map[int]bool{}
	q := byte(0)
	aware := quote == nil || *quote
	for i := 0; i < len(text); i++ {
		c := text[i]
		if aware && c == '\\' && i+1 < len(text) {
			i++
			continue
		}
		if aware && q != 0 {
			if c == q {
				if i+1 < len(text) && text[i+1] == q {
					i++
					continue
				}
				q = 0
			}
			continue
		}
		if aware && (c == '\'' || c == '"') {
			q = c
			continue
		}
		allowed[i] = true
	}
	if q != 0 {
		return nil, fmt.Errorf("unterminated quoted token")
	}
	for _, m := range locations {
		if !allowed[m[0]] {
			continue
		}
		if m[0] == m[1] {
			return nil, fmt.Errorf("split regex must consume characters")
		}
		if m[0] > start {
			parts, err := splitParts(text[start:m[0]], "\x00", aware)
			if err != nil {
				return nil, err
			}
			out = append(out, splitPart{parts[0].value, start, m[0]})
		}
		start = m[1]
	}
	if start < len(text) {
		parts, err := splitParts(text[start:], "\x00", aware)
		if err != nil {
			return nil, err
		}
		out = append(out, splitPart{parts[0].value, start, len(text)})
	}
	return out, nil
}
func mergeCondition(op Operation, prior map[string]any) bool {
	if op.Op != "conditional_merge" {
		return op.Op == "merge" || op.Op == "join"
	}
	v, exists := prior[op.ConditionField]
	switch op.ConditionOperator {
	case "exists":
		return exists
	case "equals":
		return exists && stringValue(v) == op.ConditionValue
	case "not_empty":
		return exists && v != nil && stringValue(v) != ""
	}
	return false
}

func extendedOperation(value any, op Operation, re *regexp.Regexp, prior map[string]any) (any, bool, error) {
	text := stringValue(value)
	fail := func(err error) (any, bool, error) { return nil, true, err }
	switch op.Op {
	case "omit", "exclude":
		return value, true, nil
	case "copy":
		if len(op.Fields) > 0 {
			v, ok := prior[op.Fields[0]]
			if !ok {
				return fail(ErrSourceAbsent)
			}
			return v, true, nil
		}
		return value, true, nil
	case "hash":
		if op.Algorithm == "sha512" {
			sum := sha512.Sum512([]byte(text))
			return fmt.Sprintf("%x", sum), true, nil
		}
		sum := sha256.Sum256([]byte(text))
		return fmt.Sprintf("%x", sum), true, nil
	case "redact":
		v := op.Value
		if v == "" {
			v = "[REDACTED]"
		}
		if op.Find != "" {
			growth := len(v) - len(op.Find)
			if growth > 0 && strings.Count(text, op.Find) > (MaxInputBytes-len(text))/growth {
				return fail(fmt.Errorf("redaction output exceeds bounds"))
			}
			return strings.ReplaceAll(text, op.Find, v), true, nil
		}
		return v, true, nil
	case "mask":
		r := []rune(text)
		start, end := op.Start, op.End
		if start == 0 {
			start = 1
		}
		if end == 0 {
			end = len(r) - op.Count
		}
		if start < 1 || end > len(r) || end < start-1 {
			return fail(fmt.Errorf("invalid mask range"))
		}
		v := op.Value
		if v == "" {
			v = "*"
		}
		if (end-start+1)*len(v) > MaxInputBytes {
			return fail(fmt.Errorf("mask output exceeds bounds"))
		}
		return string(r[:start-1]) + strings.Repeat(v, end-start+1) + string(r[end:]), true, nil
	case "normalize":
		switch op.Mode {
		case "lowercase":
			return strings.ToLower(text), true, nil
		case "uppercase":
			return strings.ToUpper(text), true, nil
		case "trim":
			return strings.TrimSpace(text), true, nil
		default:
			return strings.Join(strings.Fields(text), " "), true, nil
		}
	case "conditional_merge":
		v, exists := prior[op.ConditionField]
		condition := exists
		switch op.ConditionOperator {
		case "equals":
			condition = exists && stringValue(v) == op.ConditionValue
		case "not_empty":
			condition = exists && v != nil && stringValue(v) != ""
		}
		if !condition {
			return op.ElseValue, true, nil
		}
		values := []string{}
		length := 0
		for _, ref := range op.Fields {
			v, ok := prior[ref]
			if !ok {
				return fail(ErrSourceAbsent)
			}
			values = append(values, stringValue(v))
			length += len(values[len(values)-1]) + len(op.Separator)
			if length > MaxInputBytes {
				return fail(fmt.Errorf("conditional merge exceeds output bounds"))
			}
		}
		return strings.Join(values, op.Separator), true, nil
	case "before", "after", "between":
		v, _, err := boundaries(text, op.Op, op.Selection)
		return v, true, err
	case "token_match":
		limit := MaxTokens / (re.NumSubexp() + 1)
		matches := re.FindAllStringSubmatch(text, limit+1)
		if len(matches) > limit {
			return fail(fmt.Errorf("match count exceeds bounds"))
		}
		group, err := regexGroup(re, op.Group)
		if err != nil {
			return fail(err)
		}
		out := []any{}
		for _, m := range matches {
			out = append(out, m[group])
		}
		return out, true, nil
	case "first", "last", "range", "count", "tokenize", "split_characters", "split_words", "split_tokens":
		s := op.Selection
		if op.Op == "split_characters" {
			s.Unit = "character"
		}
		if op.Op == "split_words" {
			s.Unit = "word"
		}
		if op.Op == "split_tokens" {
			s.Unit = "token"
		}
		var parts []splitPart
		var err error
		if arr, ok := value.([]any); ok {
			for i, v := range arr {
				parts = append(parts, splitPart{stringValue(v), i, i + 1})
			}
		} else {
			parts, err = units(text, s, op.tokenPattern, op.QuoteAware)
		}
		if err != nil {
			return fail(err)
		}
		if op.Op == "count" {
			return len(parts), true, nil
		}
		if op.Op == "tokenize" {
			out := []any{}
			for _, p := range parts {
				out = append(out, p.value)
			}
			return out, true, nil
		}
		if strings.HasPrefix(op.Op, "split_") {
			width := op.Count
			if width == 0 {
				width = 1
			}
			out := []any{}
			for i := 0; i < len(parts); i += width {
				end := i + width
				if end > len(parts) {
					end = len(parts)
				}
				if op.Limit > 0 && len(out) == op.Limit-1 {
					end = len(parts)
				}
				out = append(out, text[parts[i].start:parts[end-1].end])
				if end == len(parts) {
					break
				}
			}
			return out, true, nil
		}
		start, end := op.Start, op.End
		n := op.Count
		if n == 0 {
			n = 1
		}
		if op.Op == "first" {
			start, end = 1, n
		}
		if op.Op == "last" {
			start, end = len(parts)-n+1, len(parts)
		}
		if start < 1 || end < start || end > len(parts) {
			return fail(ErrSourceAbsent)
		}
		if start == end {
			return parts[start-1].value, true, nil
		}
		if _, ok := value.([]any); ok {
			out := []any{}
			for _, p := range parts[start-1 : end] {
				out = append(out, p.value)
			}
			return out, true, nil
		}
		return text[parts[start-1].start:parts[end-1].end], true, nil
	case "split_regex":
		parts, err := regexParts(text, re, op.QuoteAware)
		if err != nil {
			return fail(err)
		}
		return selectedSplit(text, parts, op)
	case "split":
		if op.Count > 0 || op.Limit > 0 || op.Direction != "" || op.PreserveDelimiter {
			parts, err := splitParts(text, op.Delimiter, op.QuoteAware == nil || *op.QuoteAware)
			if err != nil {
				return fail(err)
			}
			return selectedSplit(text, parts, op)
		}
	case "key_value":
		if len(op.AssignmentSeparators) > 0 || len(op.KeyAliases) > 0 {
			f := Field{Method: "key_value", Source: op.Value, KVSeparator: op.Delimiter, PairDelimiters: op.Separator, KeyOptions: op.KeyOptions}
			v, _, ok, err := extractKeys(text, f, Evidence{})
			if err != nil {
				return fail(err)
			}
			if !ok {
				return fail(ErrSourceAbsent)
			}
			return v, true, nil
		}
	}
	return nil, false, nil
}
func selectedSplit(text string, parts []splitPart, op Operation) (any, bool, error) {
	max := op.Limit
	if op.Count > 0 && (max == 0 || op.Count+1 < max) {
		max = op.Count + 1
	}
	if max > 0 && len(parts) > max {
		if op.Direction == "right" {
			cut := len(parts) - max + 1
			merged := splitPart{text[:parts[cut-1].end], 0, parts[cut-1].end}
			parts = append([]splitPart{merged}, parts[cut:]...)
		} else {
			parts = append(parts[:max-1], splitPart{text[parts[max-1].start:], parts[max-1].start, len(text)})
		}
	}
	out := []any{}
	for i, p := range parts {
		v := p.value
		if op.PreserveDelimiter && i < len(parts)-1 {
			v += text[p.end:parts[i+1].start]
		}
		out = append(out, v)
	}
	if op.Part > 0 {
		n := op.Part
		if op.Direction == "right" {
			n = len(out) - n + 1
		}
		if n < 1 || n > len(out) {
			return nil, true, ErrSourceAbsent
		}
		return out[n-1], true, nil
	}
	return out, true, nil
}
func convertExtended(value any, f Field) (any, bool, error) {
	text := stringValue(value)
	errType := func() (any, bool, error) { return nil, true, fmt.Errorf("value is not a valid %s", f.Type) }
	switch f.Type {
	case "null":
		if value != nil && text != "null" {
			return errType()
		}
		return nil, true, nil
	case "custom":
		return text, true, nil
	case "email":
		address, err := mail.ParseAddress(text)
		if err != nil || address.Address != text || strings.ContainsAny(text, "\r\n") {
			return errType()
		}
		return text, true, nil
	case "url":
		u, err := url.Parse(text)
		if err != nil || u.Scheme == "" || u.Host == "" || strings.ContainsAny(text, "\r\n\t ") {
			return errType()
		}
		return text, true, nil
	case "uuid":
		if !uuidLiteral.MatchString(text) {
			return errType()
		}
		return text, true, nil
	case "ipv6":
		if !strings.Contains(text, ":") || net.ParseIP(text) == nil {
			return errType()
		}
		return text, true, nil
	case "date":
		layout := f.TimestampFormat
		if layout == "" {
			layout = "2006-01-02"
		}
		v, err := time.Parse(layout, text)
		if err != nil || v.Format(layout) != text {
			return errType()
		}
		return v.Format("2006-01-02"), true, nil
	case "datetime":
		v, err := convertTimestamp(text, f)
		return v, true, err
	}
	return nil, false, nil
}

var uuidLiteral = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func appendTrace(result *Result, name string, index int, operation string, input, output any, e Evidence, budget *int) {
	if *budget <= 0 {
		return
	}
	bound := func(v any) (string, bool) {
		text := stringValue(v)
		r := []rune(text)
		if len(r) > 512 {
			return string(r[:512]), true
		}
		return text, false
	}
	a, ta := bound(input)
	b, tb := bound(output)
	size := len(a) + len(b)
	if size > *budget {
		return
	}
	*budget -= size
	result.Trace[name] = append(result.Trace[name], TraceStep{Index: index, Operation: operation, Input: a, Output: b, SourceStart: e.Start, SourceEnd: e.End, Truncated: ta || tb})
}

package fieldmapping

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"nexora.local/nli-v20/internal/mixedkv"
)

// ErrIncompleteKV identifies malformed or ambiguous structured input that
// must remain available for recovery instead of entering a generic fallback.
var ErrIncompleteKV = errors.New("incomplete_structured_tail")

var kvieHeader = regexp.MustCompile(`^\|?[ \t]*[A-Za-z_][A-Za-z0-9_.-]*[ \t]*[=:;]`)
var wrappedKVPair = regexp.MustCompile(`^([\[\{(<])([A-Za-z_][A-Za-z0-9_.-]*[ \t]*[:;=][^\[\]{}()<>\r\n|]+)([\]\})>])`)

// KVProfile describes a complete observed extraction. Observations carry
// half-open UTF-8 byte spans in the original record.
type KVProfile struct {
	AssignmentSeparators []string              `json:"assignment_separators"`
	PairDelimiters       string                `json:"pair_delimiters"`
	Observations         []KeyValueObservation `json:"observations"`
	InputEdits           []KVInputEdit         `json:"kv_input_edits,omitempty"`
}

type kvieProfile struct {
	assignments []string
	delimiters  string
	pairSeps    []string
	multiSeps   []string
	quotes      string
	escape      byte
	brackets    map[string]string
	signature   string
	parser      *mixedkv.Parser
}

type kvieCacheKey struct {
	start, end int
	signature  string
}

type kvieScan struct {
	pairs map[string][]pair
	err   error
}

func compileKVIEProfile(assignments []string, delimiters string, field Field) (*kvieProfile, error) {
	cfg := mixedkv.DefaultConfig()
	cfg.AssignmentSeparators = assignments
	cfg.PairSeparators = nil
	// FieldSense's existing pair option is a set of characters. The shared
	// parser receives each configured character as a distinct separator.
	if len(field.KVPairSeparators) > 0 {
		cfg.PairSeparators = append([]string(nil), field.KVPairSeparators...)
	} else {
		for _, delimiter := range delimiters {
			if separator := string(delimiter); !contains(cfg.PairSeparators, separator) {
				cfg.PairSeparators = append(cfg.PairSeparators, separator)
			}
		}
	}
	cfg.BracketPairs["<"] = ">"
	if field.KVBrackets != nil {
		cfg.BracketPairs = make(map[string]string, len(*field.KVBrackets))
		for open, close := range *field.KVBrackets {
			cfg.BracketPairs[open] = close
		}
	}
	if field.KVQuotes != "" {
		cfg.QuoteCharacters = field.KVQuotes
	}
	if field.KVEscape != "" {
		cfg.EscapeCharacter = field.KVEscape
	}
	if field.KVKeyPattern != "" {
		cfg.KeyPattern = field.KVKeyPattern
	}
	cfg.RejectEmptyValues = field.KVRejectEmpty
	cfg.DuplicatePolicy = mixedkv.DuplicateArray
	cfg.MaxFields = MaxFields
	parser, err := mixedkv.Compile(cfg)
	if err != nil {
		return nil, err
	}
	signature, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}
	var single string
	var multiple []string
	for _, separator := range cfg.PairSeparators {
		if utf8.RuneCountInString(separator) == 1 {
			single += separator
		} else {
			multiple = append(multiple, separator)
		}
	}
	return &kvieProfile{
		assignments: assignments,
		delimiters:  single,
		pairSeps:    cfg.PairSeparators,
		multiSeps:   multiple,
		quotes:      cfg.QuoteCharacters,
		escape:      cfg.EscapeCharacter[0],
		brackets:    cfg.BracketPairs,
		signature:   string(signature),
		parser:      parser,
	}, nil
}

var kvieProfiles = func() []*kvieProfile {
	// Colon assignments deliberately use non-colon pair boundaries. A colon
	// can belong to a timestamp, URL, IPv6 address or an ordinary value.
	presets := []KVProfile{
		{AssignmentSeparators: []string{"=>", ":=", "=", ":", ";"}, PairDelimiters: "|"},
		{AssignmentSeparators: []string{"=>", ":=", "="}, PairDelimiters: ":;| \t\r\n"},
		{AssignmentSeparators: []string{":"}, PairDelimiters: ";| \t\r\n"},
		{AssignmentSeparators: []string{":"}, PairDelimiters: ","},
		{AssignmentSeparators: []string{"=>", ":=", "="}, PairDelimiters: ","},
		{AssignmentSeparators: []string{";"}, PairDelimiters: "| \t\r\n"},
		{AssignmentSeparators: []string{"=>", ":=", "=", ":"}, PairDelimiters: ";| \t\r\n"},
		{AssignmentSeparators: []string{"=>", ":=", "=", ":", ";"}, PairDelimiters: "| \t\r\n"},
	}
	out := make([]*kvieProfile, 0, len(presets))
	for _, preset := range presets {
		profile, err := compileKVIEProfile(preset.AssignmentSeparators, preset.PairDelimiters, Field{})
		if err != nil {
			panic(err)
		}
		out = append(out, profile)
	}
	return out
}()

func (p *kvieProfile) observe(raw string, offset int) ([]KeyValueObservation, error) {
	// Leading pipes occur in wrapped mail tails. Preserve their byte offset
	// and only recognize framing when the selected profile allows pipes.
	body := strings.TrimLeft(raw, " \t\r\n")
	offset += len(raw) - len(body)
	if len(p.pairSeps) == 1 && p.pairSeps[0] == "|" && strings.HasPrefix(body, "|") {
		fields, err := p.parser.ScanPipeTail(body)
		if err != nil {
			if lexical, ok := err.(*mixedkv.Error); ok && lexical.Offset >= 0 {
				positioned := *lexical
				positioned.Offset += offset
				err = &positioned
			}
			return nil, fmt.Errorf("%w: %w", ErrIncompleteKV, err)
		}
		return pipeObservations(fields, offset), nil
	}
	if unwrapped, start, framed, err := mixedkv.UnwrapAssignmentEnvelope(body); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrIncompleteKV, err)
	} else if framed {
		body = unwrapped
		offset += start
	}
	frame := ""
	for _, separator := range p.pairSeps {
		if len(separator) > len(frame) && strings.Trim(separator, "|") == "" && strings.HasPrefix(body, separator) {
			frame = separator
		}
	}
	if frame != "" {
		body = body[len(frame):]
		offset += len(frame)
	}
	framingEnd := strings.TrimRight(body, " \t\r\n")
	if frame != "" && strings.HasSuffix(framingEnd, frame) {
		backslashes := 0
		for i := len(framingEnd) - len(frame) - 1; i >= 0 && framingEnd[i] == p.escape; i-- {
			backslashes++
		}
		if backslashes%2 == 0 {
			body = framingEnd[:len(framingEnd)-len(frame)]
		}
	}
	parsed, err := p.parser.Parse(body)
	if err != nil {
		var parseErr *mixedkv.Error
		if errors.As(err, &parseErr) {
			positioned := *parseErr
			if positioned.Offset >= 0 {
				positioned.Offset += offset
			}
			if positioned.Code != mixedkv.CodeNoKeyValue {
				return nil, fmt.Errorf("%w: %w", ErrIncompleteKV, &positioned)
			}
			return nil, &positioned
		}
		return nil, err
	}
	if parsed.RemainderSpan != nil {
		return nil, fmt.Errorf("%w: unconsumed key/value fragment at byte %d", ErrIncompleteKV, offset+parsed.RemainderSpan.Start+1)
	}
	out := make([]KeyValueObservation, 0, len(parsed.Entries))
	for _, entry := range parsed.Entries {
		value := body[entry.ValueSpan.Start:entry.ValueSpan.End]
		if err := p.validateValue(value); err != nil {
			return nil, fmt.Errorf("%w: key %q at byte %d: %v", ErrIncompleteKV, entry.SourceKey, offset+entry.ValueSpan.Start, err)
		}
		out = append(out, KeyValueObservation{
			Key: entry.SourceKey, Value: entry.Value.(string),
			Start: offset + entry.ValueSpan.Start, End: offset + entry.ValueSpan.End,
		})
	}
	return out, nil
}

func pipeObservations(fields []mixedkv.PipeField, offset int) []KeyValueObservation {
	used := make(map[string]bool, len(fields))
	for _, field := range fields {
		used[field.Key] = true
	}
	out := make([]KeyValueObservation, 0, len(fields))
	next := 1
	for _, field := range fields {
		key := field.Key
		if key == "" {
			for {
				key = fmt.Sprintf("field%d", next)
				next++
				if !used[key] {
					break
				}
			}
		}
		out = append(out, KeyValueObservation{
			Key: key, Value: field.Text,
			Start: offset + field.Value.Start, End: offset + field.Value.End,
		})
	}
	return out
}

func (p *kvieProfile) validateValue(value string) error {
	if value == "" {
		return nil
	}
	if strings.ContainsRune(p.quotes, rune(value[0])) {
		if len(value) < 2 || value[len(value)-1] != value[0] {
			return fmt.Errorf("quoted value has trailing unconsumed data")
		}
		return nil
	}
	if close, ok := p.brackets[value[:1]]; ok && strings.HasSuffix(value, close) {
		return nil
	}
	return strictUnquotedPairValue(value, p.delimiters, p.escape, p.multiSeps)
}

func detectKVIE(ctx context.Context, raw string, offset int) ([]KVProfile, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if strings.HasPrefix(strings.TrimLeft(raw, " \t\r\n"), "|") {
		preset := kvieProfiles[0]
		observations, err := preset.observe(raw, offset)
		if err != nil {
			return nil, err
		}
		return []KVProfile{{
			AssignmentSeparators: append([]string(nil), preset.assignments...),
			PairDelimiters:       preset.delimiters,
			Observations:         observations,
		}}, nil
	}
	best := 0
	structured := kvieHeader.MatchString(strings.TrimLeft(raw, " \t\r\n"))
	var out []KVProfile
	var evidenceError error
	for _, preset := range kvieProfiles {
		if preset.delimiters == "|" {
			continue
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		observations, err := preset.observe(raw, offset)
		if errors.Is(err, ErrIncompleteKV) {
			structured = true
			var previous, current *mixedkv.Error
			if evidenceError == nil || errors.As(err, &current) && (!errors.As(evidenceError, &previous) || current.Offset > previous.Offset) {
				evidenceError = err
			}
		}
		if err != nil || len(observations) < best {
			continue
		}
		if len(observations) > best {
			best = len(observations)
			out = nil
		}
		duplicate := false
		for _, candidate := range out {
			if sameObservations(observations, candidate.Observations) {
				duplicate = true
				break
			}
		}
		if !duplicate {
			out = append(out, KVProfile{
				AssignmentSeparators: append([]string(nil), preset.assignments...),
				PairDelimiters:       preset.delimiters,
				Observations:         observations,
			})
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(out) == 0 {
		if evidenceError != nil {
			return nil, evidenceError
		}
		err := errors.New("no complete KV profile matched the selected source; check the start position, separators, and quotes")
		if structured {
			err = fmt.Errorf("%w: %v", ErrIncompleteKV, err)
		}
		return nil, err
	}
	return out, nil
}

func sameObservations(a, b []KeyValueObservation) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// InspectKeyValues uses the same bounded source selection and profiles as
// production extraction. The caller can show distinct competing candidates.
func InspectKeyValues(ctx context.Context, raw string, field Field) ([]KVProfile, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(raw) > MaxInputBytes || !utf8.ValidString(raw) || strings.ContainsRune(raw, 0) {
		return nil, fmt.Errorf("key/value input violates bounds or UTF-8 contract")
	}
	selector := Field{
		Name: "tail", Method: "kv_all", Type: "object",
		Source: field.Source, SourceEnd: field.SourceEnd,
		Selection: field.Selection, EndOfLine: field.EndOfLine, QuoteAware: field.QuoteAware,
	}
	if field.Method == "key_value" {
		selector.Source = field.KVStart
	}
	compiled, err := Compile(Plan{Version: 1, Fields: []Field{selector}})
	if err != nil {
		return nil, err
	}
	r := record{raw: raw}
	selected, start, _, found, err := r.keySource(compiled.fields[0])
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("key/value source position is absent from this record")
	}
	if len(field.KVInputEdits) > 0 {
		return nil, fmt.Errorf("preview the KV row to verify its configured separators and input edits")
	}
	return InspectScopedKeyValues(ctx, selected, start)
}

// InspectScopedKeyValues recognizes a complete already-selected payload.
func InspectScopedKeyValues(ctx context.Context, selected string, start int) ([]KVProfile, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(selected) > MaxInputBytes || !utf8.ValidString(selected) || strings.ContainsRune(selected, 0) {
		return nil, fmt.Errorf("key/value input violates bounds or UTF-8 contract")
	}
	if profile, ok := InspectWrappedKeyValues(selected, start); ok {
		return []KVProfile{profile}, nil
	}
	return detectKVIE(ctx, selected, start)
}

// InspectWrappedKeyValues recognizes only complete adjacent groups. Restrict
// their interiors so a suggested wrapper edit cannot consume nested data.
func InspectWrappedKeyValues(text string, offset int) (KVProfile, bool) {
	profile := KVProfile{AssignmentSeparators: []string{":", ";", "="}, PairDelimiters: "|"}
	if len(text) == 0 || !strings.ContainsRune("[{(<", rune(text[0])) {
		return profile, false
	}
	parser := kvieProfiles[len(kvieProfiles)-1]
	seen, boundaries := map[string]bool{}, map[string]bool{}
	var first, previous byte
	for pos := 0; pos < len(text); {
		match := wrappedKVPair.FindStringSubmatchIndex(text[pos:])
		if match == nil || len(profile.Observations) >= MaxFields {
			return KVProfile{}, false
		}
		open, close := text[pos+match[2]], text[pos+match[6]]
		if strings.IndexByte("[{(<", open) != strings.IndexByte("]})>", close) {
			return KVProfile{}, false
		}
		body := text[pos+match[4] : pos+match[5]]
		bodyOffset := offset + pos + match[4]
		if pos == 0 && match[1] == len(text) {
			// A single envelope can contain multiple pairs. Leave that
			// layout to the normal KV grammar detector, which unwraps it.
			profiles, _ := detectKVIE(context.Background(), body, bodyOffset)
			for _, candidate := range profiles {
				if len(candidate.Observations) > 1 {
					return KVProfile{}, false
				}
			}
		}
		values, err := parser.observe(body, bodyOffset)
		if err != nil || len(values) != 1 || seen[values[0].Key] {
			return KVProfile{}, false
		}
		seen[values[0].Key] = true
		profile.Observations = append(profile.Observations, values[0])
		if pos == 0 {
			first = open
		} else {
			boundary := string([]byte{previous, open})
			if !boundaries[boundary] {
				if len(profile.InputEdits) >= 14 {
					return KVProfile{}, false
				}
				boundaries[boundary] = true
				profile.InputEdits = append(profile.InputEdits, KVInputEdit{Action: "replace", Find: boundary, Replace: "|"})
			}
		}
		previous = close
		pos += match[1]
	}
	profile.InputEdits = append(profile.InputEdits,
		KVInputEdit{Action: "strip_prefix", Find: string(first), Required: true},
		KVInputEdit{Action: "strip_suffix", Find: string(previous), Required: true})
	return profile, true
}

func editKVInput(text string, edits []KVInputEdit) (string, error) {
	for _, edit := range edits {
		switch edit.Action {
		case "replace":
			if edit.Required && !strings.Contains(text, edit.Find) {
				return "", fmt.Errorf("required KV replacement text %q is absent", edit.Find)
			}
			value, err := applyOperation(text, Operation{Op: "replace", Find: edit.Find, Replace: edit.Replace}, nil, nil)
			if err != nil {
				return "", err
			}
			text = value.(string)
		case "strip_prefix":
			if edit.Required && !strings.HasPrefix(text, edit.Find) {
				return "", fmt.Errorf("required KV opening wrapper %q is absent", edit.Find)
			}
			text = strings.TrimPrefix(text, edit.Find)
		case "strip_suffix":
			if edit.Required && !strings.HasSuffix(text, edit.Find) {
				return "", fmt.Errorf("required KV closing wrapper %q is absent", edit.Find)
			}
			text = strings.TrimSuffix(text, edit.Find)
		}
	}
	return text, nil
}

func (r *record) extractKVIE(cf compiledField, evidence Evidence) (any, Evidence, bool, error) {
	f := cf.field
	selector := cf
	if f.Method == "key_value" {
		selector.field.Source = f.KVStart
	}
	selected, start, end, found, err := r.keySource(selector)
	if err != nil || !found {
		return nil, evidence, found, err
	}
	signature := "auto"
	if cf.kvie != nil {
		signature = cf.kvie.signature
	}
	key := kvieCacheKey{start, end, signature}
	scan, cached := r.kvieScans[key]
	if !cached {
		var observations []KeyValueObservation
		if cf.kvie != nil {
			cleaned, err := editKVInput(selected, f.KVInputEdits)
			if err != nil {
				return nil, evidence, false, fmt.Errorf("%w: %w", ErrIncompleteKV, err)
			}
			observations, scan.err = cf.kvie.observe(cleaned, start)
			if cleaned != selected {
				for i := range observations {
					observations[i].Start, observations[i].End = start, end
				}
			}
			if scan.err != nil && !errors.Is(scan.err, ErrIncompleteKV) && kvieHeader.MatchString(strings.TrimLeft(selected, " \t\r\n")) {
				scan.err = fmt.Errorf("%w: %w", ErrIncompleteKV, scan.err)
			}
		} else {
			var candidates []KVProfile
			candidates, scan.err = detectKVIE(r.ctx, selected, start)
			if scan.err == nil {
				if len(candidates) != 1 {
					scan.err = fmt.Errorf("%w: multiple KV profiles disagree; select the intended separators explicitly", ErrIncompleteKV)
				} else {
					observations = candidates[0].Observations
				}
			}
		}
		scan.pairs = make(map[string][]pair)
		for _, observation := range observations {
			scan.pairs[observation.Key] = append(scan.pairs[observation.Key], pair{observation.Value, observation.Start, observation.End})
		}
		r.kvieScans[key] = scan
	}
	if scan.err != nil {
		return nil, evidence, false, scan.err
	}
	evidence.Start, evidence.End, evidence.Source = start, end, true
	value, evidence, found, err := projectKeys(scan.pairs, f, evidence)
	if err == nil && found && f.KVNestedObjects {
		remaining := MaxFields
		value, err = expandNestedKV(r.ctx, value, f.DuplicatePolicy, 0, &remaining)
	}
	if err != nil {
		err = fmt.Errorf("%w: %w", ErrIncompleteKV, err)
	}
	return value, evidence, found, err
}

// Expand only complete, explicitly bracketed key/value groups. Each nested
// group has its own grammar and ordinary bracketed text remains a string.
func expandNestedKV(ctx context.Context, value any, duplicatePolicy string, depth int, remaining *int) (any, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	switch item := value.(type) {
	case map[string]any:
		for key, child := range item {
			expanded, err := expandNestedKV(ctx, child, duplicatePolicy, depth, remaining)
			if err != nil {
				return nil, fmt.Errorf("nested key %q: %w", key, err)
			}
			item[key] = expanded
		}
	case []any:
		for i, child := range item {
			expanded, err := expandNestedKV(ctx, child, duplicatePolicy, depth, remaining)
			if err != nil {
				return nil, err
			}
			item[i] = expanded
		}
	case string:
		text := strings.TrimSpace(item)
		if len(text) < 3 {
			return item, nil
		}
		open := strings.IndexByte("{[(<", text[0])
		if open < 0 || text[len(text)-1] != "}])>"[open] {
			return item, nil
		}
		body := strings.TrimSpace(text[1 : len(text)-1])
		if !kvieHeader.MatchString(body) {
			return item, nil
		}
		if depth >= MaxDepth {
			return nil, fmt.Errorf("nested KV exceeds depth limit %d", MaxDepth)
		}
		profiles, err := detectKVIE(ctx, body, 0)
		if err != nil {
			return nil, err
		}
		if len(profiles) != 1 {
			return nil, fmt.Errorf("nested KV grammars disagree")
		}
		observations := profiles[0].Observations
		if len(observations) > *remaining {
			return nil, fmt.Errorf("nested KV exceeds aggregate field limit %d", MaxFields)
		}
		*remaining -= len(observations)
		pairs := make(map[string][]pair, len(observations))
		for _, observation := range observations {
			pairs[observation.Key] = append(pairs[observation.Key], pair{observation.Value, observation.Start, observation.End})
		}
		object, _, _, err := projectKeys(pairs, Field{Method: "kv_all", DuplicatePolicy: duplicatePolicy}, Evidence{})
		if err != nil {
			return nil, err
		}
		return expandNestedKV(ctx, object, duplicatePolicy, depth+1, remaining)
	}
	return value, nil
}

// profileObservations returns the scoped facts used by mapping so the
// completeness gate can validate a configured grammar without reinterpreting
// its values with the legacy scanner.
func (r Result) profileObservations(start, end int) ([]KeyValueObservation, bool) {
	out := []KeyValueObservation{}
	found := false
	for key, scan := range r.kvieScans {
		if key.start != start || key.end < end || scan.err != nil {
			continue
		}
		found = true
		for name, values := range scan.pairs {
			for _, value := range values {
				out = append(out, KeyValueObservation{Key: name, Start: value.start, End: value.end})
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Start == out[j].Start {
			return out[i].Key < out[j].Key
		}
		return out[i].Start < out[j].Start
	})
	return out, found
}

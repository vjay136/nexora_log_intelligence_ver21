package controlplane

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"nexora.local/nli-v20/internal/fieldmapping"
)

// Discovery proposes data-only plans from observed structure. It neither stores
// samples nor changes the active parser set, and never learns executable code.
var nfiSuggestKVKey = regexp.MustCompile(`(?:^|[:;,\s])([A-Za-z_][A-Za-z0-9_.-]*)[ \t]*=`)
var nfiSuggestMailHeader = regexp.MustCompile(`^\d{8}\s+\d{9}[+-]\d{4}\s+\S+\s+\S+\s+\d+\s+\d+\s+\d+\s+[^;\s]+;\S+`)
var nfiSuggestISOHeader = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}\s+\d{2}:\d{2}:\d{2}`)
var nfiSuggestEventCode = regexp.MustCompile(`^([^()]*)\(([^()]*)\)$`)
var nfiSuggestPathKey = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_-]*$`)
var nfiSuggestNestedPrefix = regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_.-]*=)`)
var nfiSuggestNestedKey = regexp.MustCompile(`(?:^|\|)([A-Za-z_][A-Za-z0-9_.-]*);`)
var nfiSuggestInteger = regexp.MustCompile(`^[+-]?\d+$`)

type nfiSuggestionProfile struct {
	ID          string               `json:"id"`
	Name        string               `json:"name"`
	Family      string               `json:"family"`
	SampleCount int                  `json:"sample_count"`
	Fields      []fieldmapping.Field `json:"fields"`
	Sample      string               `json:"sample"`
	counts      map[string]int
	types       map[string]string
}

func (s *Server) handleFieldMappingSuggest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "POST required", http.StatusMethodNotAllowed)
		return
	}
	ctx, done, admitted := s.beginFieldSenseAnalysis(w, r)
	if !admitted {
		return
	}
	defer done()
	var request struct {
		Samples   []string `json:"samples"`
		Separator string   `json:"separator"`
		Service   string   `json:"service,omitempty"`
	}
	if err := decodeJSON(r, &request, 2<<20); err != nil || len(request.Samples) == 0 || len(request.Samples) > 100 {
		jsonError(w, "NFI learning requires 1..100 sample records within a 2 MiB request", http.StatusBadRequest)
		return
	}
	if request.Separator == `\t` {
		request.Separator = "\t"
	}
	if request.Separator != "" && !strings.Contains(",;|\t", request.Separator) || len([]rune(request.Separator)) > 1 {
		jsonError(w, "separator must be one literal comma, semicolon, pipe or tab", http.StatusBadRequest)
		return
	}
	for _, raw := range request.Samples {
		if len(raw) > fieldmapping.MaxInputBytes {
			jsonError(w, "a sample exceeds the 1 MiB native input bound", http.StatusBadRequest)
			return
		}
	}
	if len(request.Service) > 256 {
		jsonError(w, "runtime service scope exceeds bounds", http.StatusBadRequest)
		return
	}
	profiles, diagnostics := nfiSuggestCorpus(ctx, request.Samples, request.Separator)
	if err := s.annotateKnownKeys(ctx, request.Service, profiles); err != nil {
		diagnostics = append(diagnostics, "Published Known Key Catalog is unavailable. Structural suggestions remain available.")
	}
	runtimeInfo, runtimeFields := s.automaticSuggestionEvidence(ctx, request.Samples, request.Service)
	if fieldSenseContextError(w, ctx) {
		return
	}
	fields := []fieldmapping.Field{}
	if len(profiles) > 0 {
		fields = profiles[0].Fields
	}
	jsonOK(w, map[string]any{"engine": "nfi-v1", "sample_count": len(request.Samples), "fields": fields, "profiles": profiles, "runtime": runtimeInfo, "runtime_field_suggestions": runtimeFields, "diagnostics": diagnostics, "plan": fieldmapping.Plan{Version: 1, Fields: fields}, "persisted": false, "activated": false, "learning": "Deterministic structural suggestions; review field labels, optionality and ambiguity before validation or publication."})
}

// Catalog matches annotate only observed KV fields; they never invent an output
// field or change the literal key read by the runtime parser.
func (s *Server) annotateKnownKeys(ctx context.Context, service string, profiles []nfiSuggestionProfile) error {
	if s.metadata == nil || len(profiles) == 0 {
		return nil
	}
	catalog, err := s.listParserSyncDocuments(ctx, "known-keys", "active", parserSyncTargetLimit)
	if err != nil {
		return err
	}
	keys := map[string]bool{}
	for _, doc := range catalog {
		var asset struct {
			Config struct {
				Keys   []string `json:"keys"`
				Family string   `json:"family"`
			} `json:"config"`
		}
		if doc.Deleted || json.Unmarshal(doc.Payload, &asset) != nil {
			continue
		}
		family := strings.TrimSpace(asset.Config.Family)
		if family != "" && !strings.EqualFold(family, "auto") && !strings.EqualFold(family, service) {
			continue
		}
		for _, key := range asset.Config.Keys {
			keys[strings.ToLower(key)] = true
		}
	}
	for _, profile := range profiles {
		for i := range profile.Fields {
			field := &profile.Fields[i]
			if field.Method == "key_value" && keys[strings.ToLower(field.Source)] {
				field.Evidence += " · Observed key matches the published Known Key Catalog"
				field.Confidence = "known_catalog"
			}
		}
	}
	return nil
}

func nfiSuggestCorpus(ctx context.Context, samples []string, separator string) ([]nfiSuggestionProfile, []string) {
	profiles := []nfiSuggestionProfile{}
	positions := map[string]int{}
	diagnostics := []string{}
	for index, raw := range samples {
		if err := ctx.Err(); err != nil {
			diagnostics = append(diagnostics, "Discovery interrupted: "+err.Error())
			break
		}
		if strings.TrimSpace(raw) == "" {
			diagnostics = append(diagnostics, fmt.Sprintf("Sample %d is empty; no field values were inferred.", index+1))
			continue
		}
		family, fields, err := nfiSuggestRecord(ctx, raw, separator)
		if err != nil {
			diagnostics = append(diagnostics, fmt.Sprintf("Sample %d: %v; no mapped values were fabricated.", index+1, err))
			continue
		}
		position, exists := positions[family]
		if !exists {
			if len(profiles) >= 16 {
				diagnostics = append(diagnostics, fmt.Sprintf("Sample %d exceeds the 16-profile discovery bound; raw input was not persisted or discarded by the runtime.", index+1))
				continue
			}
			position = len(profiles)
			positions[family] = position
			profiles = append(profiles, nfiSuggestionProfile{ID: fmt.Sprintf("nfi-profile-%02d", position+1), Name: family + " mapping proposal", Family: family, Sample: raw, Fields: []fieldmapping.Field{}, counts: map[string]int{}, types: map[string]string{}})
		}
		profile := &profiles[position]
		profile.SampleCount++
		for _, field := range fields {
			key := nfiSuggestionIdentity(field)
			profile.counts[key]++
			found := -1
			for i, old := range profile.Fields {
				if nfiSuggestionIdentity(old) == key {
					found = i
					break
				}
			}
			oldType, known := profile.types[key]
			switch {
			case found >= 0 && (field.DuplicatePolicy == "array" || profile.Fields[found].DuplicatePolicy == "array"):
				profile.types[key] = "array"
				profile.Fields[found].DuplicatePolicy = "array"
			case !known:
				profile.types[key] = field.Type
			case oldType != field.Type:
				profile.types[key] = "string"
			}
			if found >= 0 {
				if field.Policy == "nullable" {
					profile.Fields[found].Policy = "nullable"
					profile.Fields[found].NullPolicy = "preserve_null"
				}
				continue
			}
			if len(profile.Fields) >= fieldmapping.MaxFields {
				diagnostics = append(diagnostics, fmt.Sprintf("Profile %s reached the 128-field proposal bound; review unmapped source evidence.", profile.ID))
				continue
			}
			used := map[string]bool{}
			for _, old := range profile.Fields {
				used[old.Name] = true
			}
			field.Name = nfiSuggestionUniqueName(field.Name, used)
			profile.Fields = append(profile.Fields, field)
		}
	}
	for p := range profiles {
		profile := &profiles[p]
		for i := range profile.Fields {
			field := &profile.Fields[i]
			key := nfiSuggestionIdentity(*field)
			count := profile.counts[key]
			field.Type = profile.types[key]
			if count < profile.SampleCount {
				field.Policy = "optional"
			}
			field.Confidence = fmt.Sprintf("observed in %d/%d samples", count, profile.SampleCount)
			field.Evidence += fmt.Sprintf("; presence %d/%d; operator-defined meaning requires review", count, profile.SampleCount)
			field.ID = fmt.Sprintf("%s-field-%03d", profile.ID, i+1)
		}
	}
	if len(profiles) > 1 {
		diagnostics = append(diagnostics, "Multiple structures were found. Review each profile separately rather than forcing unrelated formats into one required schema.")
	}
	diagnostics = append(diagnostics, "Only observed source structure is proposed. Missing or truncated evidence is not filled in; optionality is corpus evidence, not a production schema guarantee.")
	return profiles, diagnostics
}

func nfiSuggestionIdentity(f fieldmapping.Field) string {
	operations, _ := json.Marshal(f.Operations)
	return f.Method + "\x00" + f.Source + "\x00" + f.SourceEnd + "\x00" + f.Delimiter + "\x00" + f.Part + "\x00" + f.KVProfile + "\x00" + f.KVStart + "\x00" + string(operations)
}

func nfiSuggestionUniqueName(name string, used map[string]bool) string {
	var b strings.Builder
	for _, c := range name {
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_' {
			b.WriteRune(c)
		} else {
			b.WriteByte('_')
		}
	}
	name = b.String()
	if name == "" {
		name = "field"
	}
	if name[0] == '_' || name[0] >= '0' && name[0] <= '9' {
		name = "field_" + name
	}
	for _, reserved := range fieldmapping.ReservedNames {
		if strings.EqualFold(name, reserved) {
			name = "field_" + name
			break
		}
	}
	if len(name) > 56 {
		name = name[:56]
	}
	base := name
	for n := 2; used[name]; n++ {
		name = base + "_" + strconv.Itoa(n)
	}
	return name
}

func nfiSuggestedField(name, method, source, preview string) fieldmapping.Field {
	return fieldmapping.Field{Name: name, Label: strings.ReplaceAll(name, "_", " "), Method: method, Source: source, Type: "string", Policy: "required", Transform: "none", NullPolicy: "omit", Timezone: "preserve", Status: "suggested", Preview: preview, Evidence: method + " source " + source, Confidence: "observed structure", DuplicatePolicy: "reject"}
}

func nfiSuggestedType(name, value string) string {
	lower := strings.ToLower(name)
	if strings.Contains(lower, "id") || strings.Contains(lower, "thread") || strings.Contains(lower, "date") || strings.Contains(lower, "time_timezone") {
		return "string"
	}
	if value == "true" || value == "false" {
		return "boolean"
	}
	if len(value) > 1 && value[0] == '0' && value[1] >= '0' && value[1] <= '9' {
		return "string"
	}
	if strings.HasPrefix(value, "-0") && len(value) > 2 && value[2] >= '0' && value[2] <= '9' {
		return "string"
	}
	if n, err := strconv.ParseInt(value, 10, 64); err == nil {
		if n >= -9007199254740991 && n <= 9007199254740991 {
			return "integer"
		}
		return "string"
	}
	// Integer-looking identifiers beyond int64 must not be rounded through float.
	if nfiSuggestInteger.MatchString(value) {
		return "string"
	}
	if n, err := strconv.ParseFloat(value, 64); err == nil && !math.IsNaN(n) && !math.IsInf(n, 0) {
		return "number"
	}
	return "string"
}

func nfiSuggestRecord(ctx context.Context, raw, separator string) (string, []fieldmapping.Field, error) {
	if len(raw) > fieldmapping.MaxInputBytes || !utf8.ValidString(raw) || strings.ContainsRune(raw, 0) {
		return "", nil, fmt.Errorf("sample violates the bounded UTF-8 input contract")
	}
	trimmed := strings.TrimSpace(raw)
	if profile, ok := fieldmapping.InspectWrappedKeyValues(trimmed, 0); ok {
		return "bracketed-key-value", []fieldmapping.Field{nfiWrappedKVField("raw", profile)}, nil
	}
	if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
		return nfiSuggestJSON(ctx, raw)
	}
	if separator != "" {
		if family, fields, ok := nfiSuggestCSV(raw, separator); ok {
			return family, fields, nil
		}
		if strings.Contains(raw, separator) {
			return "", nil, fmt.Errorf("sample is not a valid bounded single CSV record for the selected separator")
		}
	}
	tokens := strings.Fields(raw)
	fields := []fieldmapping.Field{}
	family := "key-value"
	mail := nfiSuggestMailHeader.MatchString(raw)
	if mail {
		family = "mail-header/" + tokens[3]
		for i, name := range []string{"date", "time_timezone", "hostname", "service", "pid", "minor_pid", "thread_id"} {
			field := nfiSuggestedField(name, "position", strconv.Itoa(i+1), tokens[i])
			if i == 4 || i == 5 {
				field.Type = "integer"
			}
			fields = append(fields, field)
		}
		parts := strings.SplitN(tokens[7], ";", 2)
		for i, name := range []string{"loglevel", "event"} {
			field := nfiSuggestedField(name, "delimiter", "8", parts[i])
			field.Delimiter = ";"
			field.Part = strconv.Itoa(i + 1)
			field.Confirmed = false
			field.Status = "needs_confirmation"
			field.Evidence = "Observed compound token 8; confirm semicolon interpretation"
			fields = append(fields, field)
		}
		if match := nfiSuggestEventCode.FindStringSubmatch(parts[1]); match != nil {
			fields[len(fields)-1].Operations = []fieldmapping.Operation{{Op: "regex_extract", Pattern: `^([^()]+)`, Group: "1"}}
			fields[len(fields)-1].Preview = match[1]
			code := nfiSuggestedField("event_code", "regex", `^(?:\S+\s+){7}[^;\s]+;[^\s(]+\(([^)\s]+)\)`, match[2])
			code.Part = "1"
			fields = append(fields, code)
		}
	} else if nfiSuggestISOHeader.MatchString(raw) && len(tokens) > 1 {
		family = "timestamp-prefixed-key-value"
		fields = append(fields, nfiSuggestedField("date", "position", "1", tokens[0]), nfiSuggestedField("time_timezone", "position", "2", tokens[1]))
	}
	assignments := []string{"="}
	discoveryRaw := raw
	if mail && len(tokens) > 8 {
		discoveryRaw = strings.TrimSpace(raw[len(nfiSuggestMailHeader.FindString(raw)):])
		if strings.HasPrefix(discoveryRaw, "|") {
			profiles, err := fieldmapping.InspectKeyValues(ctx, raw, fieldmapping.Field{Name: "tail_fields", Type: "object", Method: "kv_all", Source: "9"})
			if err != nil {
				return "", nil, err
			}
			if len(profiles) != 1 {
				return "", nil, fmt.Errorf("pipe payload needs one complete extraction grammar")
			}
			return family + "/pipe-kv", nfiKVIEFields(fields, profiles[0]), nil
		}
		if profile, ok := fieldmapping.InspectWrappedKeyValues(discoveryRaw, 0); ok {
			return family + "/bracketed-kv", append(fields, nfiWrappedKVField("9", profile)), nil
		}
	}
	observations, scanErr := fieldmapping.ScanKeyValues(discoveryRaw, assignments, ":;,| \t\r\n", false)
	if scanErr != nil {
		return "", nil, scanErr
	}
	if mail && len(tokens) > 8 && len(observations) == 0 {
		profiles, err := fieldmapping.InspectKeyValues(ctx, raw, fieldmapping.Field{Name: "tail_fields", Type: "object", Method: "kv_all", Source: "9", EndOfLine: true})
		if err == nil && len(profiles) == 1 {
			return family, nfiKVIEFields(fields, profiles[0]), nil
		}
	}
	keys := []string{}
	seen := map[string]bool{}
	for _, observation := range observations {
		match := []string{"", observation.Key}
		if !seen[match[1]] {
			seen[match[1]] = true
			keys = append(keys, match[1])
			if len(keys) >= fieldmapping.MaxFields {
				break
			}
		}
	}
	if len(keys) > 0 {
		probeFields := make([]fieldmapping.Field, 0, len(keys))
		for i, key := range keys {
			f := nfiSuggestedField(fmt.Sprintf("probe_%03d", i), "key_value", key, "")
			f.Policy = "optional"
			f.AssignmentSeparators = assignments
			f.PairDelimiters = ":;,| \t\r\n"
			probeFields = append(probeFields, f)
		}
		compiled, err := fieldmapping.Compile(fieldmapping.Plan{Version: 1, Fields: probeFields})
		if err != nil {
			return "", nil, err
		}
		result, err := compiled.Execute(ctx, raw)
		if err != nil {
			return "", nil, err
		}
		used := map[string]bool{}
		for _, f := range fields {
			used[f.Name] = true
		}
		for i, key := range keys {
			value, exists := result.Values[fmt.Sprintf("probe_%03d", i)]
			if !exists {
				continue
			}
			text, ok := value.(string)
			if !ok {
				continue
			}
			name := nfiSuggestionUniqueName(key, used)
			used[name] = true
			field := nfiSuggestedField(name, "key_value", key, text)
			field.KVSeparator = assignments[0]
			field.AssignmentSeparators = assignments
			field.PairDelimiters = ":;,| \t\r\n"
			field.Type = nfiSuggestedType(key, text)
			fields = append(fields, field)
			if len(fields) >= fieldmapping.MaxFields {
				break
			}
			for _, nested := range nfiSuggestNested(ctx, raw, key, text, used) {
				fields = append(fields, nested)
				if len(fields) >= fieldmapping.MaxFields {
					break
				}
			}
			if len(fields) >= fieldmapping.MaxFields {
				break
			}
		}
		if len(fields) > 0 {
			return family, fields, nil
		}
	}
	if mail {
		return family, fields, nil
	}
	if family == "timestamp-prefixed-key-value" {
		return family, fields, nil
	}
	if csvFamily, csvFields, ok := nfiSuggestCSV(raw, separator); ok {
		return csvFamily, csvFields, nil
	}
	fields = fields[:0]
	for i, token := range tokens {
		if i >= fieldmapping.MaxFields {
			break
		}
		fields = append(fields, nfiSuggestedField("token_"+strconv.Itoa(i+1), "position", strconv.Itoa(i+1), token))
	}
	return "unstructured-tokens", fields, nil
}

func nfiWrappedKVField(start string, profile fieldmapping.KVProfile) fieldmapping.Field {
	field := nfiSuggestedField("tail_fields", "kv_all", start, "")
	field.Type, field.OutputPolicy, field.KVProfile = "object", "flatten", "configured"
	field.EndOfLine, field.RequireComplete = true, true
	field.AssignmentSeparators = profile.AssignmentSeparators
	field.KVPairSeparators = []string{profile.PairDelimiters}
	field.KVInputEdits = profile.InputEdits
	field.Evidence = "Native inspection recognized all adjacent bracketed pairs in this sample"
	return field
}

func nfiKVIEFields(fields []fieldmapping.Field, profile fieldmapping.KVProfile) []fieldmapping.Field {
	used, seen := map[string]bool{}, map[string]bool{}
	counts := map[string]int{}
	for _, f := range fields {
		used[f.Name] = true
	}
	for _, pair := range profile.Observations {
		counts[pair.Key]++
	}
	for _, pair := range profile.Observations {
		if len(fields) >= fieldmapping.MaxFields {
			break
		}
		if seen[pair.Key] {
			continue
		}
		seen[pair.Key] = true
		name := nfiSuggestionUniqueName(pair.Key, used)
		used[name] = true
		f := nfiSuggestedField(name, "key_value", pair.Key, pair.Value)
		f.KVProfile, f.KVStart, f.EndOfLine = "auto", "9", true
		f.AssignmentSeparators, f.PairDelimiters = profile.AssignmentSeparators, profile.PairDelimiters
		f.Type = nfiSuggestedType(pair.Key, pair.Value)
		f.Evidence = "Observed with KV AutoSense at token 9; verify the source, grammar and output name"
		if profile.PairDelimiters == "|" {
			f.EndOfLine = false
			f.Evidence = "Observed in the pipe-delimited payload at token 9. Rename the output field to match your schema."
		}
		if counts[pair.Key] > 1 {
			f.Type, f.DuplicatePolicy = "array", "array"
		}
		fields = append(fields, f)
	}
	return fields
}

func nfiSuggestNested(ctx context.Context, raw, key, text string, used map[string]bool) []fieldmapping.Field {
	prefix := ""
	body := text
	if match := nfiSuggestNestedPrefix.FindStringSubmatch(body); match != nil {
		prefix = match[1]
		body = strings.TrimPrefix(body, prefix)
	}
	if !strings.Contains(body, "|") {
		return nil
	}
	fields := []fieldmapping.Field{}
	seen := map[string]bool{}
	for _, match := range nfiSuggestNestedKey.FindAllStringSubmatch(body, fieldmapping.MaxFields) {
		if ctx.Err() != nil {
			break
		}
		nestedKey := match[1]
		if seen[nestedKey] {
			continue
		}
		seen[nestedKey] = true
		name := nfiSuggestionUniqueName(key+"_"+nestedKey, used)
		f := nfiSuggestedField(name, "key_value", key, "")
		f.PairDelimiters = ":;, \t\r\n"
		if prefix != "" {
			f.Operations = append(f.Operations, fieldmapping.Operation{Op: "strip_prefix", Value: prefix})
		}
		f.Operations = append(f.Operations, fieldmapping.Operation{Op: "key_value", Value: nestedKey, Delimiter: ";", Separator: "|"})
		compiled, err := fieldmapping.Compile(fieldmapping.Plan{Version: 1, Fields: []fieldmapping.Field{f}})
		if err != nil {
			continue
		}
		result, err := compiled.Execute(ctx, raw)
		if err != nil {
			continue
		}
		value, ok := result.Values[name].(string)
		if !ok {
			continue
		}
		f.Type = nfiSuggestedType(nestedKey, value)
		f.Preview = value
		f.Evidence = "Observed nested key " + nestedKey + " inside " + key + "; review ordered prefix and separator operations"
		used[name] = true
		fields = append(fields, f)
	}
	return fields
}

func nfiSuggestCSV(raw, separator string) (string, []fieldmapping.Field, bool) {
	separators := []rune{',', '\t', '|', ';'}
	if separator != "" {
		separators = []rune{[]rune(separator)[0]}
	}
	for _, sep := range separators {
		reader := csv.NewReader(strings.NewReader(raw))
		reader.Comma = sep
		reader.FieldsPerRecord = -1
		values, err := reader.Read()
		if err != nil || len(values) < 2 || len(values) > fieldmapping.MaxFields {
			continue
		}
		if _, err = reader.Read(); err != io.EOF {
			continue
		}
		quoted := `"(?:[^"]|"")*"`
		plain := `[^"` + regexp.QuoteMeta(string(sep)) + `\r\n]*`
		column := `(?:` + quoted + `|` + plain + `)`
		delimiter := regexp.QuoteMeta(string(sep))
		fields := make([]fieldmapping.Field, 0, len(values))
		for i, value := range values {
			pattern := `^(?:` + column + delimiter + `){` + strconv.Itoa(i) + `}(` + column + `)(?:` + delimiter + column + `)*\r?\n?$`
			f := nfiSuggestedField("column_"+strconv.Itoa(i+1), "regex", pattern, value)
			f.Part = "1"
			f.Type = nfiSuggestedType(f.Name, value)
			f.Operations = []fieldmapping.Operation{{Op: "strip_prefix", Value: `"`}, {Op: "strip_suffix", Value: `"`}, {Op: "replace", Find: `""`, Replace: `"`}}
			f.Evidence = "Observed quote-aware delimited column " + strconv.Itoa(i+1)
			fields = append(fields, f)
		}
		return "delimited/" + strconv.QuoteRune(sep) + "/" + strconv.Itoa(len(values)), fields, true
	}
	return "", nil, false
}

func nfiSuggestJSON(ctx context.Context, raw string) (string, []fieldmapping.Field, error) {
	// Execute the native root selector first: it rejects duplicate keys, invalid
	// quoting, excessive depth/nodes and trailing records before discovery.
	typeName := "object"
	if strings.HasPrefix(strings.TrimSpace(raw), "[") {
		typeName = "array"
	}
	root := nfiSuggestedField("record", "json_path", "$", "")
	root.Type = typeName
	compiled, err := fieldmapping.Compile(fieldmapping.Plan{Version: 1, Fields: []fieldmapping.Field{root}})
	if err != nil {
		return "", nil, err
	}
	result, err := compiled.Execute(ctx, raw)
	if err != nil {
		return "", nil, err
	}
	fields := []fieldmapping.Field{}
	used := map[string]bool{}
	unsupportedKey := false
	var walk func(any, string, string, int)
	walk = func(value any, path, name string, depth int) {
		if len(fields) >= fieldmapping.MaxFields || depth > fieldmapping.MaxDepth {
			return
		}
		if object, ok := value.(map[string]any); ok && len(object) > 0 {
			keys := make([]string, 0, len(object))
			for key := range object {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			for _, key := range keys {
				if !nfiSuggestPathKey.MatchString(key) {
					unsupportedKey = true
					continue
				}
				childPath := path + "." + key
				childName := key
				if name != "" {
					childName = name + "_" + key
				}
				walk(object[key], childPath, childName, depth+1)
			}
			return
		}
		if name == "" {
			name = "record"
		}
		name = nfiSuggestionUniqueName(name, used)
		used[name] = true
		encoded, _ := json.Marshal(value)
		preview := string(encoded)
		if text, ok := value.(string); ok {
			preview = text
		}
		field := nfiSuggestedField(name, "json_path", path, preview)
		switch value := value.(type) {
		case map[string]any:
			field.Type = "object"
		case []any:
			field.Type = "array"
		case bool:
			field.Type = "boolean"
		case json.Number:
			field.Type = nfiSuggestedType(name, value.String())
		case nil:
			field.Policy = "nullable"
			field.NullPolicy = "preserve_null"
		default:
			field.Type = "string"
		}
		fields = append(fields, field)
	}
	walk(result.Values["record"], "$", "", 0)
	if unsupportedKey {
		root.Evidence = "Whole structured record retained because a key needs an explicit supported mapping; no dotted key was silently relabeled"
		return "structured-json/" + typeName, []fieldmapping.Field{root}, nil
	}
	if len(fields) == 0 {
		fields = append(fields, root)
	}
	return "structured-json/" + typeName, fields, nil
}

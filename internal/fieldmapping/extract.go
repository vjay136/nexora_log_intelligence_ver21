package fieldmapping

import (
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

type span struct{ start, end int }
type pair struct {
	value      string
	start, end int
}
type record struct {
	raw        string
	tokens     []span
	lines      []span
	kv         map[string]map[string][]pair
	object     any
	jsonParsed bool
	jsonError  error
}

func newRecord(raw string) (*record, error) {
	if !utf8.ValidString(raw) {
		return nil, fmt.Errorf("mapping input is not valid UTF-8")
	}
	r := &record{raw: raw, kv: map[string]map[string][]pair{}}
	start := -1
	for i, c := range raw {
		if unicode.IsSpace(c) {
			if start >= 0 {
				r.tokens = append(r.tokens, span{start, i})
				start = -1
			}
		} else if start < 0 {
			start = i
		}
		if len(r.tokens) > MaxTokens {
			return nil, fmt.Errorf("record exceeds token limit")
		}
	}
	if start >= 0 {
		r.tokens = append(r.tokens, span{start, len(raw)})
	}
	if len(r.tokens) > MaxTokens {
		return nil, fmt.Errorf("record exceeds token limit")
	}
	lineStart := 0
	for i, c := range raw {
		if c == '\n' {
			end := i
			if end > lineStart && raw[end-1] == '\r' {
				end--
			}
			r.lines = append(r.lines, span{lineStart, end})
			lineStart = i + 1
			if len(r.lines) > MaxTokens {
				return nil, fmt.Errorf("record exceeds line limit")
			}
		}
	}
	r.lines = append(r.lines, span{lineStart, len(raw)})
	return r, nil
}

func (r *record) extract(cf compiledField, prior map[string]any) (any, Evidence, bool, error) {
	f := cf.field
	e := Evidence{Start: -1, End: -1, Description: f.Method + ":" + f.Source}
	selected := func(s span) (any, Evidence, bool, error) {
		e.Start = s.start
		e.End = s.end
		e.Source = s.end > s.start
		return r.raw[s.start:s.end], e, true, nil
	}
	switch f.Method {
	case "position", "line":
		items := r.tokens
		if f.Method == "line" {
			items = r.lines
		}
		start, _ := index(f.Source, 1)
		end, _ := index(f.SourceEnd, start)
		if start > len(items) || end > len(items) {
			return nil, e, false, nil
		}
		return selected(span{items[start-1].start, items[end-1].end})
	case "character", "byte":
		start, _ := index(f.Source, 1)
		end, _ := index(f.SourceEnd, start)
		s, ok, err := textRange(r.raw, start, end, f.Method == "byte")
		if err != nil {
			return nil, e, false, err
		}
		if !ok {
			return nil, e, false, nil
		}
		return selected(s)
	case "delimiter":
		text, base := r.raw, 0
		if f.Source != "raw" && f.Source != "" {
			i, _ := index(f.Source, 1)
			if i > len(r.tokens) {
				return nil, e, false, nil
			}
			s := r.tokens[i-1]
			text, base = r.raw[s.start:s.end], s.start
		}
		part, _ := index(f.Part, 1)
		parts, err := splitParts(text, f.Delimiter, f.QuoteAware == nil || *f.QuoteAware)
		if err != nil {
			return nil, e, false, err
		}
		if part > len(parts) {
			return nil, e, false, nil
		}
		item := parts[part-1]
		e.Start = base + item.start
		e.End = base + item.end
		e.Source = e.End > e.Start
		return item.value, e, true, nil
	case "key_value":
		separator := f.KVSeparator
		if separator == "" {
			separator = "="
		}
		delimiters := f.PairDelimiters
		if delimiters == "" {
			delimiters = ":;, \t\r\n"
		}
		cacheKey := separator + "\x00" + delimiters
		pairs, ok := r.kv[cacheKey]
		if !ok {
			var err error
			pairs, err = scanPairs(r.raw, separator, delimiters)
			if err != nil {
				return nil, e, false, err
			}
			r.kv[cacheKey] = pairs
		}
		values := pairs[f.Source]
		if len(values) == 0 {
			return nil, e, false, nil
		}
		if len(values) > 1 && f.DuplicatePolicy == "reject" {
			return nil, e, false, fmt.Errorf("duplicate key %q requires an explicit first, last, or array policy", f.Source)
		}
		item := values[0]
		if f.DuplicatePolicy == "last" {
			item = values[len(values)-1]
		}
		e.Start = item.start
		e.End = item.end
		e.Source = item.end > item.start
		if f.DuplicatePolicy == "array" {
			output := make([]any, len(values))
			for i, v := range values {
				output[i] = v.value
			}
			e.Source = false
			return output, e, true, nil
		}
		return item.value, e, true, nil
	case "regex":
		indexes := cf.pattern.FindStringSubmatchIndex(r.raw)
		if indexes == nil {
			return nil, e, false, nil
		}
		group := cf.pattern.SubexpIndex(f.Name)
		if f.Part != "" {
			if n, err := strconv.Atoi(f.Part); err == nil {
				group = n
			} else {
				group = cf.pattern.SubexpIndex(f.Part)
			}
		}
		if group < 0 {
			if cf.pattern.NumSubexp() > 0 {
				group = 1
			} else {
				group = 0
			}
		}
		if group*2+1 >= len(indexes) || indexes[group*2] < 0 {
			return nil, e, false, nil
		}
		return selected(span{indexes[group*2], indexes[group*2+1]})
	case "json_path":
		if !r.jsonParsed {
			r.object, r.jsonError = parseJSON(r.raw)
			r.jsonParsed = true
		}
		if r.jsonError != nil {
			return nil, e, false, r.jsonError
		}
		path, _ := parsePath(f.Source)
		value := r.object
		for _, key := range path {
			switch key := key.(type) {
			case string:
				object, ok := value.(map[string]any)
				if !ok {
					return nil, e, false, nil
				}
				var exists bool
				value, exists = object[key]
				if !exists {
					return nil, e, false, nil
				}
			case int:
				array, ok := value.([]any)
				if !ok || key >= len(array) {
					return nil, e, false, nil
				}
				value = array[key]
			}
		}
		e.Start = 0
		e.End = len(r.raw)
		e.Source = len(r.raw) > 0
		return value, e, true, nil
	case "constant":
		return f.DefaultValue, e, true, nil
	case "derived":
		parts := []string{}
		for _, name := range strings.Split(f.Source, ",") {
			value, ok := prior[strings.TrimSpace(name)]
			if !ok || value == nil {
				return nil, e, false, fmt.Errorf("referenced field %q is absent", name)
			}
			parts = append(parts, stringValue(value))
		}
		return strings.Join(parts, f.JoinSeparator), e, true, nil
	}
	return nil, e, false, fmt.Errorf("unsupported extraction method")
}

func textRange(text string, start, end int, bytes bool) (span, bool, error) {
	if end == 0 {
		end = start
	}
	if start < 1 || end < start {
		return span{}, false, fmt.Errorf("range must be 1-based inclusive")
	}
	if bytes {
		if end > len(text) {
			return span{}, false, nil
		}
		if !utf8.ValidString(text[start-1 : end]) {
			return span{}, false, fmt.Errorf("byte range splits a UTF-8 character")
		}
		return span{start - 1, end}, true, nil
	}
	first, last, count := -1, -1, 0
	for offset := range text {
		count++
		if count == start {
			first = offset
		}
		if count == end+1 {
			last = offset
			break
		}
	}
	if first < 0 || count < end {
		return span{}, false, nil
	}
	if last < 0 {
		last = len(text)
	}
	return span{first, last}, true, nil
}

func keyByte(c byte, first bool) bool {
	return c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c == '_' || !first && (c >= '0' && c <= '9' || c == '.' || c == '-')
}
func keyAt(raw string, pos int, separator string) (string, int, bool) {
	if pos >= len(raw) || !keyByte(raw[pos], true) {
		return "", 0, false
	}
	end := pos + 1
	for end < len(raw) && keyByte(raw[end], false) {
		end++
	}
	keyEnd := end
	for end < len(raw) && (raw[end] == ' ' || raw[end] == '\t') {
		end++
	}
	if !strings.HasPrefix(raw[end:], separator) {
		return "", 0, false
	}
	return raw[pos:keyEnd], end + len(separator), true
}
func boundary(c byte, delimiters string) bool {
	return c == ' ' || c == '\t' || c == '\r' || c == '\n' || c == '\v' || c == '\f' || strings.ContainsRune(delimiters, rune(c))
}

// scanPairs uses delimiters only when followed by a syntactically valid key and
// separator. Colons inside URLs, IPv6 addresses and ordinary values survive.
// Quoted/escaped boundaries are never treated as new pairs.
func scanPairs(raw, separator, delimiters string) (map[string][]pair, error) {
	output := map[string][]pair{}
	pos := 0
	pairCount := 0
	for pos < len(raw) {
		if pos > 0 && !boundary(raw[pos-1], delimiters) {
			pos++
			continue
		}
		key, valueStart, ok := keyAt(raw, pos, separator)
		if !ok {
			pos++
			continue
		}
		for valueStart < len(raw) && (raw[valueStart] == ' ' || raw[valueStart] == '\t') {
			valueStart++
		}
		quote := byte(0)
		structured := valueStart < len(raw) && (raw[valueStart] == '[' || raw[valueStart] == '{')
		var brackets []byte
		end, next := len(raw), len(raw)
		for i := valueStart; i < len(raw); i++ {
			c := raw[i]
			if c == '\\' {
				if i+1 >= len(raw) {
					return nil, fmt.Errorf("key %q ends with an incomplete escape", key)
				}
				i++
				continue
			}
			if quote != 0 {
				if c == quote {
					if i+1 < len(raw) && raw[i+1] == quote {
						i++
						continue
					}
					quote = 0
				}
				continue
			}
			if (c == '\'' || c == '"') && (i == valueStart || boundary(raw[i-1], delimiters) || raw[i-1] == '[' || raw[i-1] == '(') {
				quote = c
				continue
			}
			if structured {
				if c == '[' || c == '{' {
					brackets = append(brackets, c)
					if len(brackets) > MaxDepth {
						return nil, fmt.Errorf("nested key-value structure exceeds depth limit")
					}
					continue
				}
				if c == ']' || c == '}' {
					if len(brackets) == 0 {
						return nil, fmt.Errorf("unmatched key-value structure delimiter")
					}
					open := brackets[len(brackets)-1]
					if c == ']' && open != '[' || c == '}' && open != '{' {
						return nil, fmt.Errorf("mismatched key-value structure delimiter")
					}
					brackets = brackets[:len(brackets)-1]
					continue
				}
				if len(brackets) > 0 {
					continue
				}
			}
			if boundary(c, delimiters) {
				j := i + 1
				for j < len(raw) && boundary(raw[j], delimiters) {
					j++
				}
				if _, _, valid := keyAt(raw, j, separator); valid {
					end, next = i, j
					break
				}
			}
		}
		if quote != 0 {
			return nil, fmt.Errorf("key %q has an unterminated quoted value", key)
		}
		if len(brackets) > 0 {
			return nil, fmt.Errorf("key %q has an unterminated nested structure", key)
		}
		valueEnd := end
		for valueEnd > valueStart && (raw[valueEnd-1] == ' ' || raw[valueEnd-1] == '\t' || raw[valueEnd-1] == '\r' || raw[valueEnd-1] == '\n') {
			valueEnd--
		}
		text := raw[valueStart:valueEnd]
		if len(text) >= 2 && (text[0] == '\'' || text[0] == '"') && text[len(text)-1] == text[0] {
			q := text[0]
			text = text[1 : len(text)-1]
			text = strings.ReplaceAll(text, string([]byte{q, q}), string(q))
		}
		var b strings.Builder
		for i := 0; i < len(text); i++ {
			if text[i] == '\\' && i+1 < len(text) && (text[i+1] == '\\' || text[i+1] == '"' || text[i+1] == '\'' || boundary(text[i+1], delimiters)) {
				i++
			}
			b.WriteByte(text[i])
		}
		output[key] = append(output[key], pair{b.String(), valueStart, valueEnd})
		pairCount++
		if pairCount > 4096 {
			return nil, fmt.Errorf("record exceeds key-value pair limit")
		}
		pos = next
	}
	return output, nil
}

var pathSegment = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_-]*$`)

func parsePath(text string) ([]any, error) {
	if strings.HasPrefix(text, "$") && len(text) > 1 && text[1] != '.' && text[1] != '[' {
		return nil, fmt.Errorf("JSON root must be followed by a dot or bracket")
	}
	text = strings.TrimPrefix(text, "$")
	text = strings.TrimPrefix(text, ".")
	if text == "" {
		return nil, nil
	}
	var path []any
	for len(text) > 0 {
		if len(path) >= MaxDepth {
			return nil, fmt.Errorf("JSON path exceeds nesting limit")
		}
		if text[0] == '[' {
			if len(text) > 2 && (text[1] == '"' || text[1] == '\'') {
				quote := text[1]
				end := -1
				for i := 2; i < len(text); i++ {
					if text[i] == '\\' {
						i++
						continue
					}
					if text[i] == quote {
						end = i
						break
					}
				}
				if end < 0 || end+1 >= len(text) || text[end+1] != ']' {
					return nil, fmt.Errorf("invalid quoted JSON path key")
				}
				var key string
				if quote == '"' {
					var err error
					key, err = strconv.Unquote(text[1 : end+1])
					if err != nil {
						return nil, fmt.Errorf("invalid escaped JSON path key")
					}
				} else {
					key = strings.ReplaceAll(strings.ReplaceAll(text[2:end], `\'`, "'"), `\\`, `\`)
				}
				if key == "" || len(key) > 256 {
					return nil, fmt.Errorf("quoted JSON path key exceeds bounds")
				}
				path = append(path, key)
				text = text[end+2:]
				if len(text) > 0 && text[0] != '.' && text[0] != '[' {
					return nil, fmt.Errorf("JSON path segments need a dot or bracket separator")
				}
				if strings.HasPrefix(text, ".") {
					text = text[1:]
					if text == "" {
						return nil, fmt.Errorf("JSON path cannot end with a dot")
					}
				}
				continue
			}
			end := strings.IndexByte(text, ']')
			if end < 0 {
				return nil, fmt.Errorf("invalid JSON array path")
			}
			n, err := strconv.Atoi(text[1:end])
			if err != nil || n < 0 || n > 4095 {
				return nil, fmt.Errorf("JSON array index must be 0..4095")
			}
			path = append(path, n)
			text = text[end+1:]
			if len(text) > 0 && text[0] != '.' && text[0] != '[' {
				return nil, fmt.Errorf("JSON path segments need a dot or bracket separator")
			}
		} else {
			end := strings.IndexAny(text, ".[")
			if end < 0 {
				end = len(text)
			}
			key := text[:end]
			if !pathSegment.MatchString(key) {
				return nil, fmt.Errorf("JSON path supports bounded dot keys and array indices")
			}
			path = append(path, key)
			text = text[end:]
		}
		if strings.HasPrefix(text, ".") {
			text = text[1:]
			if text == "" {
				return nil, fmt.Errorf("JSON path cannot end with a dot")
			}
		}
	}
	return path, nil
}

func parseJSON(text string) (any, error) {
	decoder := json.NewDecoder(strings.NewReader(text))
	decoder.UseNumber()
	nodes := 0
	var read func(int) (any, error)
	read = func(depth int) (any, error) {
		if depth > MaxDepth {
			return nil, fmt.Errorf("JSON exceeds nesting limit")
		}
		nodes++
		if nodes > 8192 {
			return nil, fmt.Errorf("JSON exceeds node limit")
		}
		token, err := decoder.Token()
		if err != nil {
			return nil, err
		}
		switch token := token.(type) {
		case json.Delim:
			switch token {
			case '{':
				m := map[string]any{}
				for decoder.More() {
					keyToken, err := decoder.Token()
					if err != nil {
						return nil, err
					}
					key, ok := keyToken.(string)
					if !ok {
						return nil, fmt.Errorf("JSON object key is invalid")
					}
					if _, exists := m[key]; exists {
						return nil, fmt.Errorf("duplicate JSON key %q", key)
					}
					value, err := read(depth + 1)
					if err != nil {
						return nil, err
					}
					m[key] = value
				}
				if _, err := decoder.Token(); err != nil {
					return nil, err
				}
				return m, nil
			case '[':
				a := []any{}
				for decoder.More() {
					if len(a) >= 4096 {
						return nil, fmt.Errorf("JSON array exceeds element limit")
					}
					value, err := read(depth + 1)
					if err != nil {
						return nil, err
					}
					a = append(a, value)
				}
				if _, err := decoder.Token(); err != nil {
					return nil, err
				}
				return a, nil
			}
			return nil, fmt.Errorf("unexpected JSON delimiter")
		default:
			return token, nil
		}
	}
	value, err := read(0)
	if err != nil {
		return nil, err
	}
	if _, err = decoder.Token(); err != io.EOF {
		return nil, fmt.Errorf("JSON must contain exactly one value")
	}
	return value, nil
}

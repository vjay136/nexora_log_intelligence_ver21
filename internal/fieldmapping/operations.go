package fieldmapping

import (
	"encoding/json"
	"fmt"
	"math"
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

var integerLiteral = regexp.MustCompile(`^-?[0-9]+$`)
var compactMail = regexp.MustCompile(`^[0-9]{8} [0-9]{9}[+-][0-9]{4}$`)

func stringValue(value any) string {
	if text, ok := value.(string); ok {
		return text
	}
	data, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return string(data)
}
func valueSize(value any) int {
	if text, ok := value.(string); ok {
		return len(text)
	}
	data, _ := json.Marshal(value)
	return len(data)
}

func applyOperation(value any, op Operation, re *regexp.Regexp, prior map[string]any) (any, error) {
	text := stringValue(value)
	switch op.Op {
	case "trim":
		return strings.TrimSpace(text), nil
	case "lowercase":
		return strings.ToLower(text), nil
	case "uppercase":
		return strings.ToUpper(text), nil
	case "prefix":
		return op.Value + text, nil
	case "suffix", "add":
		return text + op.Value, nil
	case "strip_prefix":
		return strings.TrimPrefix(text, op.Value), nil
	case "strip_suffix":
		return strings.TrimSuffix(text, op.Value), nil
	case "replace", "remove":
		replacement := op.Replace
		if op.Op == "remove" {
			replacement = ""
		}
		count := strings.Count(text, op.Find)
		growth := len(replacement) - len(op.Find)
		if growth > 0 && count > (MaxInputBytes-len(text))/growth {
			return nil, fmt.Errorf("replacement output exceeds bounds")
		}
		return strings.ReplaceAll(text, op.Find, replacement), nil
	case "split":
		parts, err := splitParts(text, op.Delimiter, op.QuoteAware == nil || *op.QuoteAware)
		if err != nil {
			return nil, err
		}
		if op.Part > 0 {
			if op.Part > len(parts) {
				return nil, fmt.Errorf("split part %d is absent", op.Part)
			}
			return parts[op.Part-1].value, nil
		}
		values := make([]any, len(parts))
		for i, v := range parts {
			values[i] = v.value
		}
		return values, nil
	case "select":
		values, ok := value.([]any)
		if !ok {
			return nil, fmt.Errorf("select requires an array from split or JSON")
		}
		if op.Part > len(values) {
			return nil, fmt.Errorf("selected part is absent")
		}
		return values[op.Part-1], nil
	case "substring", "character", "byte":
		end := op.End
		if end == 0 {
			if op.Op == "byte" {
				end = len(text)
			} else {
				end = utf8.RuneCountInString(text)
			}
		}
		s, ok, err := textRange(text, op.Start, end, op.Op == "byte")
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, fmt.Errorf("selected range is absent")
		}
		return text[s.start:s.end], nil
	case "line":
		parts := strings.Split(text, "\n")
		end := op.End
		if end == 0 {
			end = op.Start
		}
		if op.Start > len(parts) || end > len(parts) {
			return nil, fmt.Errorf("selected line is absent")
		}
		return strings.Join(parts[op.Start-1:end], "\n"), nil
	case "key_value":
		sep := op.Delimiter
		if sep == "" {
			sep = "="
		}
		delimiters := op.Separator
		if delimiters == "" {
			delimiters = ":;, \t\r\n"
		}
		pairs, err := scanPairs(text, sep, delimiters)
		if err != nil {
			return nil, err
		}
		items := pairs[op.Value]
		if len(items) == 0 {
			return nil, fmt.Errorf("%w: nested key %q", ErrSourceAbsent, op.Value)
		}
		if len(items) > 1 {
			return nil, fmt.Errorf("nested key %q is duplicated", op.Value)
		}
		return items[0].value, nil
	case "regex_extract":
		match := re.FindStringSubmatch(text)
		if match == nil {
			return nil, fmt.Errorf("regex operation did not match")
		}
		group := 1
		if re.NumSubexp() == 0 {
			group = 0
		}
		if op.Group != "" {
			if n, err := strconv.Atoi(op.Group); err == nil {
				group = n
			} else {
				group = re.SubexpIndex(op.Group)
			}
		}
		if group < 0 || group >= len(match) {
			return nil, fmt.Errorf("regex group is absent")
		}
		return match[group], nil
	case "regex_replace":
		// Stream replacement through a bounded builder so a high-match-count
		// pattern cannot allocate unbounded expanded output before rejection.
		indexes := re.FindAllStringSubmatchIndex(text, MaxTokens+1)
		if len(indexes) > MaxTokens {
			return nil, fmt.Errorf("regex replacement exceeds match limit")
		}
		var b strings.Builder
		previous := 0
		for _, m := range indexes {
			expanded := re.ExpandString(nil, op.Replace, text, m)
			if b.Len()+m[0]-previous+len(expanded) > MaxInputBytes {
				return nil, fmt.Errorf("regex replacement output exceeds bounds")
			}
			b.WriteString(text[previous:m[0]])
			b.Write(expanded)
			previous = m[1]
		}
		if b.Len()+len(text)-previous > MaxInputBytes {
			return nil, fmt.Errorf("regex replacement output exceeds bounds")
		}
		b.WriteString(text[previous:])
		return b.String(), nil
	case "join", "merge":
		values := []string{}
		if len(op.Fields) > 0 {
			for _, name := range op.Fields {
				v, ok := prior[name]
				if !ok || v == nil {
					return nil, fmt.Errorf("referenced field %q is absent", name)
				}
				values = append(values, stringValue(v))
			}
		} else {
			array, ok := value.([]any)
			if !ok {
				return nil, fmt.Errorf("join requires split/JSON array or explicit prior fields")
			}
			for _, v := range array {
				values = append(values, stringValue(v))
			}
		}
		length := 0
		for _, v := range values {
			length += len(v)
		}
		length += len(op.Separator) * max(0, len(values)-1)
		if length > MaxInputBytes {
			return nil, fmt.Errorf("joined output exceeds bounds")
		}
		return strings.Join(values, op.Separator), nil
	}
	return nil, fmt.Errorf("unsupported operation %q", op.Op)
}

func convert(value any, f Field) (any, error) {
	text := stringValue(value)
	switch f.Type {
	case "string":
		return text, nil
	case "integer", "int", "port":
		if !integerLiteral.MatchString(text) {
			return nil, fmt.Errorf("value is not an integer")
		}
		n, err := strconv.ParseInt(text, 10, 64)
		if err != nil || n < -9007199254740991 || n > 9007199254740991 {
			return nil, fmt.Errorf("integer exceeds exact browser/JSON range; preserve identifiers as strings")
		}
		if f.Type == "port" && (n < 0 || n > 65535) {
			return nil, fmt.Errorf("port must be 0..65535")
		}
		return int(n), nil
	case "number", "float":
		n, err := strconv.ParseFloat(text, 64)
		if err != nil || math.IsNaN(n) || math.IsInf(n, 0) {
			return nil, fmt.Errorf("value is not a finite number")
		}
		return n, nil
	case "boolean", "bool":
		if text == "true" {
			return true, nil
		}
		if text == "false" {
			return false, nil
		}
		return nil, fmt.Errorf("boolean must be literal true or false")
	case "ip":
		if net.ParseIP(text) == nil {
			return nil, fmt.Errorf("value is not an IPv4 or IPv6 address")
		}
		return text, nil
	case "timestamp", "time":
		return convertTimestamp(text, f)
	case "object", "array":
		parsed := value
		if _, ok := value.(string); ok {
			var err error
			parsed, err = parseJSON(text)
			if err != nil {
				return nil, fmt.Errorf("invalid bounded JSON: %w", err)
			}
		}
		if f.Type == "object" {
			if _, ok := parsed.(map[string]any); !ok {
				return nil, fmt.Errorf("value is not a JSON object")
			}
		} else if _, ok := parsed.([]any); !ok {
			return nil, fmt.Errorf("value is not a JSON array")
		}
		return parsed, nil
	}
	return nil, fmt.Errorf("unsupported type")
}

func convertTimestamp(text string, f Field) (any, error) {
	layout := f.TimestampFormat
	parseText := text
	if layout == "compact_mail" || layout == "yyyyMMdd HHmmssSSSZZ" {
		if !compactMail.MatchString(text) {
			return nil, fmt.Errorf("compact_mail timestamp needs YYYYMMDD HHMMSSmmm+ZZZZ")
		}
		parseText = text[:15] + "." + text[15:]
		layout = "20060102 150405.000-0700"
	}
	if layout == "" {
		layout = time.RFC3339Nano
	}
	if strings.HasPrefix(layout, "epoch_") {
		number, err := strconv.ParseInt(text, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("epoch timestamp is not an integer")
		}
		var parsed time.Time
		switch layout {
		case "epoch_seconds":
			parsed = time.Unix(number, 0)
		case "epoch_milliseconds":
			parsed = time.UnixMilli(number)
		case "epoch_microseconds":
			parsed = time.UnixMicro(number)
		case "epoch_nanoseconds":
			parsed = time.Unix(0, number)
		default:
			return nil, fmt.Errorf("unsupported epoch timestamp format")
		}
		if parsed.Year() < 1 || parsed.Year() > 9999 {
			return nil, fmt.Errorf("timestamp year is outside JSON/RFC3339 bounds")
		}
		if f.Timezone == "configured" {
			location, err := time.LoadLocation(f.TimezoneName)
			if err != nil {
				return nil, err
			}
			return parsed.In(location).Format(time.RFC3339Nano), nil
		}
		return parsed.UTC().Format(time.RFC3339Nano), nil
	}
	var parsed time.Time
	var err error
	if f.Timezone == "configured" {
		location, loadErr := time.LoadLocation(f.TimezoneName)
		if loadErr != nil {
			return nil, loadErr
		}
		parsed, err = time.ParseInLocation(layout, parseText, location)
		if err == nil {
			return parsed.In(location).Format(time.RFC3339Nano), nil
		}
	} else {
		parsed, err = time.Parse(layout, parseText)
	}
	if err != nil {
		return nil, fmt.Errorf("timestamp does not match its explicit layout")
	}
	if f.Timezone == "utc" {
		return parsed.UTC().Format(time.RFC3339Nano), nil
	}
	return text, nil
}

func validateRules(rules string) error {
	for _, raw := range strings.Split(rules, ",") {
		rule := strings.TrimSpace(raw)
		if rule == "" {
			continue
		}
		parts := strings.SplitN(rule, "=", 2)
		if len(parts) != 2 || strings.TrimSpace(parts[1]) == "" {
			return fmt.Errorf("validation needs kind=value")
		}
		kind, value := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
		switch kind {
		case "min", "max", "min_length", "max_length":
			n, err := strconv.ParseFloat(value, 64)
			if err != nil || math.IsInf(n, 0) || math.IsNaN(n) {
				return fmt.Errorf("validation bound must be finite")
			}
			if (kind == "min_length" || kind == "max_length") && (n < 0 || n > MaxInputBytes || n != math.Trunc(n)) {
				return fmt.Errorf("length bound must be a nonnegative integer within input bounds")
			}
		case "one_of":
			items := strings.Split(value, "|")
			if len(items) > 100 {
				return fmt.Errorf("allowed set exceeds 100 values")
			}
			for _, item := range items {
				if len(item) == 0 || len(item) > 256 {
					return fmt.Errorf("allowed values require 1..256 bytes")
				}
			}
		default:
			return fmt.Errorf("unsupported validation rule %q", kind)
		}
	}
	return nil
}

func checkRules(value any, rules string) error {
	for _, raw := range strings.Split(rules, ",") {
		rule := strings.TrimSpace(raw)
		if rule == "" {
			continue
		}
		parts := strings.SplitN(rule, "=", 2)
		kind, expected := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
		switch kind {
		case "min", "max":
			number, err := strconv.ParseFloat(stringValue(value), 64)
			if err != nil {
				return fmt.Errorf("numeric validation requires a number")
			}
			switch value.(type) {
			case int, int64, float64, json.Number:
			default:
				return fmt.Errorf("numeric validation requires numeric output type")
			}
			bound, _ := strconv.ParseFloat(expected, 64)
			if kind == "min" && number < bound || kind == "max" && number > bound {
				return fmt.Errorf("value violates %s=%s", kind, expected)
			}
		case "min_length", "max_length":
			bound, _ := strconv.Atoi(expected)
			n := utf8.RuneCountInString(stringValue(value))
			if kind == "min_length" && n < bound || kind == "max_length" && n > bound {
				return fmt.Errorf("value violates %s=%s", kind, expected)
			}
		case "one_of":
			if !contains(strings.Split(expected, "|"), stringValue(value)) {
				return fmt.Errorf("value is outside its allowed set")
			}
		}
	}
	return nil
}

package fieldmapping

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var ErrIncompleteHeader = errors.New("incomplete_structured_header")

const MaxMailExtraTags = 16

const extendedMailLead = `^\s*(\d{8})\s+(\d{9}[+-]\d{4})\s+(\S+)\s+(\S+)\s+(\d+)\s+(\d+)\s+(\d+)\s+(\d+);`

var (
	extendedMailPrefix = regexp.MustCompile(extendedMailLead)
	extendedMailHeader = regexp.MustCompile(extendedMailLead + fmt.Sprintf(`((?:[^;\s]+;){1,%d})([^;()\s]+)(?:\(([^()\s]+)\))?(?:\s|$)`, MaxMailExtraTags+1))
)

var mailComponentNames = [...]string{"date", "timestamp", "hostname", "service", "pid", "minor_pid", "thread_id", "random_number", "", "event", "event_code"}

// MailLevelIndex identifies a unique conventional level. For older layouts
// with custom level names it retains the convention of a level next to the event.
func MailLevelIndex(parts []string) (int, bool) {
	level := -1
	for i, part := range parts {
		switch strings.ToLower(part) {
		case "trace", "debug", "info", "note", "notice", "warn", "warning", "err", "error", "fatal", "critical", "alert", "emerg", "panic":
			if level >= 0 {
				return -1, false
			}
			level = i
		}
	}
	if level < 0 {
		level = len(parts) - 1
	}
	return level, level >= 0
}

var extendedMailAliases = map[string]string{
	"date": "date", "timestamp": "timestamp", "time_timezone": "timestamp",
	"hostname": "hostname", "service": "service", "pid": "pid", "thread_id": "thread_id",
	"random_number": "random_number", "extra_tag": "extra_tag",
	"loglevel": "loglevel", "event_type": "loglevel", "level": "loglevel",
	"event": "event", "event_name": "event", "event_code": "event_code", "code": "event_code",
}

// HasMailHeaderExtension identifies the recognized numeric-prefix mail layout.
func HasMailHeaderExtension(raw string) bool {
	return extendedMailPrefix.MatchString(raw)
}

// ValidateMailHeader accounts for data components before KV extraction. Source
// ranges are zero-based, half-open UTF-8 byte offsets. A broader range covering
// multiple components is not proof that each was extracted.
func ValidateMailHeader(raw string, fields map[string]Evidence) error {
	matches := extendedMailHeader.FindStringSubmatchIndex(raw)
	if matches == nil {
		if HasMailHeaderExtension(raw) {
			return fmt.Errorf("%w: numeric-prefix mail header does not fit the recognized component grammar", ErrIncompleteHeader)
		}
		return nil
	}
	return validateMailHeader(raw, matches, fields)
}

func validateMailHeader(raw string, matches []int, fields map[string]Evidence) error {
	required := make(map[Span]string, len(mailComponentNames))
	for i, name := range mailComponentNames {
		start, end := matches[2*i+2], matches[2*i+3]
		if name == "" {
			parts := strings.Split(strings.TrimSuffix(raw[start:end], ";"), ";")
			level, ok := MailLevelIndex(parts)
			if !ok {
				return fmt.Errorf("%w: ambiguous mail level before the event at byte %d", ErrIncompleteHeader, start+1)
			}
			extra := 0
			for index, part := range parts {
				component := "loglevel"
				if index != level {
					extra++
					component = "extra_tag" + strconv.Itoa(extra)
				}
				required[Span{start, start + len(part)}] = component
				start += len(part) + 1
			}
			continue
		}
		if start >= 0 {
			required[Span{start, end}] = name
		}
	}

	covered := make(map[Span]bool, len(required))
	names := make([]string, 0, len(fields))
	for name := range fields {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		evidence := fields[name]
		spans := evidence.Spans
		if len(spans) == 0 && evidence.Start >= 0 && evidence.End > evidence.Start {
			spans = []Span{{evidence.Start, evidence.End}}
		}
		fieldComponents := 0
		for _, span := range spans {
			component, found := required[span]
			alias := extendedMailAliases[name]
			if alias == "extra_tag" {
				alias = "extra_tag1"
			}
			if alias == "" && strings.HasPrefix(name, "extra_tag") {
				if _, err := strconv.Atoi(strings.TrimPrefix(name, "extra_tag")); err == nil {
					alias = name
				}
			}
			if alias != "" && span.Start < matches[1] && component != alias {
				return fmt.Errorf("%w: field %q does not match its %s source component", ErrIncompleteHeader, name, alias)
			}
			if found {
				fieldComponents++
				if covered[span] || fieldComponents > 1 {
					return fmt.Errorf("%w: overlapping or combined source component %s at byte %d", ErrIncompleteHeader, component, span.Start+1)
				}
				covered[span] = true
			}
		}
	}
	firstMissing := Span{Start: len(raw) + 1}
	for span := range required {
		if !covered[span] && span.Start < firstMissing.Start {
			firstMissing = span
		}
	}
	if name, missing := required[firstMissing]; missing {
		return fmt.Errorf("%w: %s has no exact mapped or intentionally omitted source component at byte %d", ErrIncompleteHeader, name, firstMissing.Start+1)
	}
	return nil
}

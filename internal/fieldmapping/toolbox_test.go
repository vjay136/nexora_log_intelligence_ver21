package fieldmapping

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestFieldSenseToolboxSelectors(t *testing.T) {
	cases := []struct {
		name, raw string
		field     Field
		want      any
	}{
		{"between nth", "first[one] second[two]", Field{Method: "between", Selection: Selection{BoundaryStart: "[", BoundaryEnd: "]", Occurrence: "nth", OccurrenceIndex: 2}}, "two"},
		{"all preserved", "[one][two]", Field{Method: "between", Type: "array", Selection: Selection{BoundaryStart: "[", BoundaryEnd: "]", Occurrence: "all", PreserveBoundary: true}}, []any{"[one]", "[two]"}},
		{"before last", "a:b:c", Field{Method: "before", Selection: Selection{BoundaryStart: ":", Occurrence: "last"}}, "a:b"},
		{"after nth", "a:b:c", Field{Method: "after", Selection: Selection{BoundaryStart: ":", Occurrence: "nth", OccurrenceIndex: 2}}, "c"},
		{"mixed quoted token", `one,"two words";three`, Field{Method: "token", Source: "2", Selection: Selection{TokenMode: "mixed", TokenSeparators: []string{",", ";"}}}, "two words"},
		{"right token", "one two three", Field{Method: "word", Source: "1", Selection: Selection{Direction: "right"}}, "three"},
		{"arbitrary indices", "one two three", Field{Method: "word", Type: "array", Selection: Selection{Indices: []int{3, 1}}}, []any{"three", "one"}},
		{"count Unicode", "aé🙂", Field{Method: "count", Type: "integer", Selection: Selection{Unit: "character"}}, 3},
		{"all regex", "id=12 id=34", Field{Method: "regex_all", Source: `id=(\d+)`, Part: "1", Type: "array"}, []any{"12", "34"}},
		{"named regex", "a=7", Field{Method: "regex_groups", Source: `(?P<key>\w+)=(?P<value>\d+)`, Type: "object"}, map[string]any{"key": "a", "value": "7"}},
		{"multiple assignments", `user:=Alice status=failed:reason=notneeded`, Field{Method: "kv_all", Type: "object", KeyOptions: KeyOptions{AssignmentSeparators: []string{":=", "="}, KeyAliases: map[string]string{"user": "account"}, KeyExclude: []string{"reason"}, KeyPrefix: "x_"}}, map[string]any{"x_account": "Alice", "x_status": "failed"}},
		{"alias selected", `user=Alice`, Field{Method: "key_value", Source: "account", KeyOptions: KeyOptions{KeyAliases: map[string]string{"user": "account"}}}, "Alice"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := tc.field
			f.Name = "value"
			c, err := Compile(Plan{Version: 1, Fields: []Field{f}})
			if err != nil {
				t.Fatal(err)
			}
			got, err := c.Execute(context.Background(), tc.raw)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got.Values["value"], tc.want) {
				t.Fatalf("got %#v want %#v", got.Values["value"], tc.want)
			}
		})
	}
}

func TestFieldSenseOrderedToolbox(t *testing.T) {
	cases := []struct {
		name, raw, typ string
		op             Operation
		want           any
	}{
		{"Unicode last", "abé🙂", "string", Operation{Op: "last", Selection: Selection{Unit: "character", Count: 2}}, "é🙂"},
		{"word range", "a  b c", "string", Operation{Op: "range", Start: 2, End: 3, Selection: Selection{Unit: "word"}}, "b c"},
		{"right split preserve rest", "a:b:c:d", "array", Operation{Op: "split", Delimiter: ":", Selection: Selection{Direction: "right", Count: 2}}, []any{"a:b", "c", "d"}},
		{"split limit", "a:b:c:d", "array", Operation{Op: "split", Delimiter: ":", Limit: 2}, []any{"a", "b:c:d"}},
		{"split delimiter preservation", "a:b:c", "array", Operation{Op: "split", Delimiter: ":", PreserveDelimiter: true}, []any{"a:", "b:", "c"}},
		{"regex quoted split", `a,"b,c",d`, "array", Operation{Op: "split_regex", Pattern: ","}, []any{"a", "b,c", "d"}},
		{"character chunks", "abcé🙂", "array", Operation{Op: "split_characters", Selection: Selection{Count: 2}}, []any{"ab", "cé", "🙂"}},
		{"word chunks", "a b c d e", "array", Operation{Op: "split_words", Selection: Selection{Count: 2}}, []any{"a b", "c d", "e"}},
		{"regex token matches", "a=1 b=2", "array", Operation{Op: "token_match", Pattern: `\w=(\d)`, Group: "1"}, []any{"1", "2"}},
		{"mask Unicode", "é1234", "string", Operation{Op: "mask", Selection: Selection{Count: 2}}, "***34"},
		{"hash", "0042", "string", Operation{Op: "hash", Algorithm: "sha256"}, "1e20c878d20e7e3b4b0c25c9d7b4f653e8bf4c2cbbd32cc99a4dd4845d2fa887"},
		{"redact selected", "a=secret b=ok", "string", Operation{Op: "redact", Find: "secret"}, "a=[REDACTED] b=ok"},
		{"whitespace normalize", " a  b\t c ", "string", Operation{Op: "normalize", Mode: "whitespace"}, "a b c"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, err := Compile(Plan{Version: 1, Fields: []Field{{Name: "value", Method: "raw", Type: tc.typ, Operations: []Operation{tc.op}}}})
			if err != nil {
				t.Fatal(err)
			}
			got, err := c.Execute(context.Background(), tc.raw)
			if err != nil {
				t.Fatal(err)
			}
			if tc.name == "hash" {
				if got.Values["value"] != fmt.Sprintf("%x", sha256.Sum256([]byte("0042"))) {
					t.Fatal("SHA256 exact value")
				}
				return
			}
			if !reflect.DeepEqual(got.Values["value"], tc.want) {
				t.Fatalf("got %#v want %#v", got.Values["value"], tc.want)
			}
		})
	}
}

func TestFieldSenseOutputPoliciesTypesTraceAndImmutability(t *testing.T) {
	fields := []Field{
		{Name: "temporary", Method: "key_value", Source: "status", OutputPolicy: "intermediate"},
		{Name: "summary", Method: "field", Source: "temporary", Operations: []Operation{{Op: "conditional_merge", Fields: []string{"temporary"}, ConditionField: "temporary", ConditionOperator: "equals", ConditionValue: "failed", ElseValue: "ok", RemoveSources: true}}},
		{Name: "contact", Method: "key_value", Source: "email", Type: "email", Transform: "lowercase"},
		{Name: "nilvalue", Method: "constant", DefaultValue: "null", Type: "null"},
		{Name: "custom_value", Method: "constant", DefaultValue: "A12", Type: "custom", CustomPattern: `A\d{2}`},
	}
	c, err := Compile(Plan{Version: 1, Fields: fields})
	if err != nil {
		t.Fatal(err)
	}
	fields[1].Operations[0].Fields[0] = "later_mutation"
	got, err := c.ExecuteTrace(context.Background(), "status=failed email=Alice@example.test")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := got.Values["temporary"]; ok {
		t.Fatal("intermediate leaked")
	}
	if got.Values["summary"] != "failed" || got.Values["contact"] != "alice@example.test" || got.Values["nilvalue"] != nil {
		t.Fatalf("unexpected output %#v", got.Values)
	}
	if len(got.Trace["summary"]) != 2 || got.Trace["summary"][1].Operation != "conditional_merge" {
		t.Fatal("missing ordered trace")
	}
	plain, err := c.Execute(context.Background(), "status=passed email=Bob@example.test")
	if err != nil {
		t.Fatal(err)
	}
	if plain.Trace != nil || plain.Values["summary"] != "ok" {
		t.Fatal("trace in normal runtime or condition fallback wrong")
	}
	for _, typ := range []string{"email", "url", "uuid", "ipv6", "date", "datetime", "custom"} {
		f := Field{Name: "value", Method: "raw", Type: typ, CustomPattern: `A\d+`}
		c, err := Compile(Plan{Version: 1, Fields: []Field{f}})
		if err != nil {
			t.Fatal(err)
		}
		if _, err = c.Execute(context.Background(), "invalid"); err == nil {
			t.Fatalf("%s accepted invalid", typ)
		}
	}
	for _, tc := range []struct{ typ, value string }{{"url", "https://example.test/a?x=1"}, {"uuid", "123e4567-e89b-12d3-a456-426614174000"}, {"ipv6", "2001:db8::1"}, {"date", "2024-02-29"}, {"datetime", "2024-02-29T12:00:00Z"}} {
		c, err := Compile(Plan{Version: 1, Fields: []Field{{Name: "value", Method: "raw", Type: tc.typ}}})
		if err != nil {
			t.Fatal(err)
		}
		if _, err = c.Execute(context.Background(), tc.value); err != nil {
			t.Fatalf("valid %s: %v", tc.typ, err)
		}
	}
}

func TestFieldSenseEmptyAndBoundedSelections(t *testing.T) {
	for _, policy := range []string{"preserve", "missing", "null", "reject"} {
		c, err := Compile(Plan{Version: 1, Fields: []Field{{Name: "value", Method: "key_value", Source: "x", Policy: "default", DefaultValue: "fallback", EmptyPolicy: policy}}})
		if err != nil {
			t.Fatal(err)
		}
		got, err := c.Execute(context.Background(), "x=")
		if policy == "reject" {
			if err == nil {
				t.Fatal("empty rejected policy accepted")
			}
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		want := any("")
		if policy == "missing" {
			want = "fallback"
		}
		if policy == "null" {
			want = nil
		}
		if got.Values["value"] != want {
			t.Fatalf("empty %s %#v", policy, got.Values)
		}
	}
	c, err := Compile(Plan{Version: 1, Fields: []Field{{Name: "value", Method: "before", Type: "array", Selection: Selection{BoundaryStart: "a", Occurrence: "all"}}}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = c.Execute(context.Background(), strings.Repeat("a", 2000)); err == nil {
		t.Fatal("quadratic boundary output accepted")
	}
	if _, err := Compile(Plan{Version: 1, Fields: []Field{{Name: "value", Method: "raw", Description: strings.Repeat("x", MaxPlanBytes)}}}); err == nil {
		t.Fatal("new metadata bypasses plan bound")
	}
}

func FuzzFieldSenseToolboxNeverPanics(f *testing.F) {
	for _, s := range []string{`user:=Alice status=failed:reason=none`, `[one][two]`, `a,"b,c",d`, "é🙂 words", `a="unterminated`} {
		f.Add(s)
	}
	fields := []Field{
		{Name: "labels", Method: "kv_all", Type: "object", KeyOptions: KeyOptions{AssignmentSeparators: []string{":=", "="}}, DuplicatePolicy: "array"},
		{Name: "tokens", Method: "raw", Type: "array", Operations: []Operation{{Op: "split_regex", Pattern: ","}}},
		{Name: "slice", Method: "between", Policy: "optional", Selection: Selection{BoundaryStart: "[", BoundaryEnd: "]", Occurrence: "last"}},
		{Name: "last", Method: "raw", Policy: "optional", Operations: []Operation{{Op: "last", Selection: Selection{Unit: "character", Count: 1}}}},
		{Name: "mask", Method: "raw", Operations: []Operation{{Op: "mask"}}},
	}
	c, err := Compile(Plan{Version: 1, Fields: fields})
	if err != nil {
		f.Fatal(err)
	}
	f.Fuzz(func(t *testing.T, raw string) {
		if len(raw) > 16<<10 {
			return
		}
		_, _ = c.ExecuteTrace(context.Background(), raw)
	})
}

func TestFieldSenseHostileConfigurationAndExpansionBounds(t *testing.T) {
	base := Field{Name: "value", Method: "raw"}
	cases := []Field{base, base, base, base, base}
	cases[0].Operations = make([]Operation, MaxOperations+1)
	cases[1].Method = "regex_all"
	cases[1].Source = strings.Repeat("a", MaxRegexBytes+1)
	cases[2].TokenMode = "mixed"
	cases[2].TokenSeparators = make([]string, 33)
	cases[3].Operations = []Operation{{Op: "hash", Algorithm: "md5"}}
	cases[4].Operations = []Operation{{Op: "conditional_merge", ConditionField: "missing", ConditionOperator: "execute"}}
	for i, f := range cases {
		if _, err := Compile(Plan{Version: 1, Fields: []Field{f}}); err == nil {
			t.Fatalf("hostile configuration %d accepted", i)
		}
	}
	c, err := Compile(Plan{Version: 1, Fields: []Field{{Name: "value", Method: "raw", Operations: []Operation{{Op: "redact", Find: "a", Value: strings.Repeat("b", 4096)}}}}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = c.Execute(context.Background(), strings.Repeat("a", 300)); err == nil {
		t.Fatal("redaction expansion bypasses bound")
	}
	c, err = Compile(Plan{Version: 1, Fields: []Field{{Name: "value", Method: "raw", Operations: []Operation{{Op: "first", Selection: Selection{Unit: "byte"}}}}}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = c.Execute(context.Background(), "é"); err == nil {
		t.Fatal("byte selection silently corrupts UTF-8")
	}
}

func FuzzFieldSenseConfigurationNeverPanics(f *testing.F) {
	for _, s := range []string{`{"version":1,"fields":[{"name":"value","method":"raw","operations":[{"op":"mask","start":1,"count":2}]}]}`, `{"version":1,"fields":[{"name":"value","method":"regex_all","source":"()()()","type":"array"}]}`} {
		f.Add(s, "aé🙂 a=1")
	}
	f.Fuzz(func(t *testing.T, config, raw string) {
		if len(config) > 8192 || len(raw) > 8192 {
			return
		}
		var p Plan
		if json.Unmarshal([]byte(config), &p) != nil {
			return
		}
		c, err := Compile(p)
		if err == nil {
			_, _ = c.ExecuteTrace(context.Background(), raw)
		}
	})
}

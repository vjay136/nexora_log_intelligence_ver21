package fieldmapping

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
)

func TestNFIAllOrderedOperationContracts(t *testing.T) {
	for _, tc := range []struct {
		name, input string
		ops         []Operation
		want        any
		typ         string
	}{
		{"replace_remove", "a-b!", []Operation{{Op: "replace", Find: "-", Replace: "_"}, {Op: "remove", Find: "!"}}, "a_b", ""},
		{"case_trim", " A ", []Operation{{Op: "trim"}, {Op: "lowercase"}}, "a", ""},
		{"prefix_suffix_add", "x", []Operation{{Op: "prefix", Value: "["}, {Op: "suffix", Value: "]"}, {Op: "add", Value: "!"}}, "[x]!", ""},
		{"strip", "[x]", []Operation{{Op: "strip_prefix", Value: "["}, {Op: "strip_suffix", Value: "]"}}, "x", ""},
		{"substring", "Aé猫Z", []Operation{{Op: "substring", Start: 2, End: 3}}, "é猫", ""},
		{"characters", "Aé猫Z", []Operation{{Op: "character", Start: 3}}, "猫Z", ""},
		{"bytes", "Aé猫Z", []Operation{{Op: "byte", Start: 2, End: 6}}, "é猫", ""},
		{"line", "a\nb\nc", []Operation{{Op: "line", Start: 2, End: 3}}, "b\nc", ""},
		{"split_select", "x|y|z", []Operation{{Op: "split", Delimiter: "|"}, {Op: "select", Part: 3}}, "z", ""},
		{"split_quoted", `a,"b,c",d`, []Operation{{Op: "split", Delimiter: ",", Part: 2}}, "b,c", ""},
		{"regex_extract", "id=abc123", []Operation{{Op: "regex_extract", Pattern: `id=(?P<identifier>[a-z]+)[0-9]+`, Group: "identifier"}}, "abc", ""},
		{"regex_replace", "a12-b34", []Operation{{Op: "regex_replace", Pattern: `([0-9]+)`, Replace: "[$1]"}}, "a[12]-b[34]", ""},
		{"join", "a|b|c", []Operation{{Op: "split", Delimiter: "|"}, {Op: "join", Separator: " + "}}, "a + b + c", ""},
		{"nested_key", "action;success|recipient;user@example.test", []Operation{{Op: "key_value", Value: "action", Delimiter: ";", Separator: "|"}}, "success", ""},
		{"object", `{"a":[1,true]}`, nil, map[string]any{"a": []any{json.Number("1"), true}}, "object"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			typ := tc.typ
			if typ == "" {
				typ = "string"
			}
			r := execute(t, []Field{{Name: "output", Method: "constant", DefaultValue: tc.input, Operations: tc.ops, Type: typ}}, "")
			if !reflect.DeepEqual(r.Values["output"], tc.want) {
				t.Fatalf("got %#v want %#v", r.Values["output"], tc.want)
			}
		})
	}
	r := execute(t, []Field{{Name: "a", Method: "constant", DefaultValue: "left"}, {Name: "b", Method: "constant", DefaultValue: "right"}, {Name: "joined", Method: "constant", Operations: []Operation{{Op: "merge", Fields: []string{"a", "b"}, Separator: "/"}}}}, "")
	if r.Values["joined"] != "left/right" {
		t.Fatal(r.Values)
	}
}

func TestNFIRegexSelectorAndExplicitWholeMatch(t *testing.T) {
	r := execute(t, []Field{{Name: "full", Method: "regex", Source: `id=(?P<value>[0-9]+)`, Part: "0"}, {Name: "selected", Method: "regex", Source: `id=(?P<value>[0-9]+)`, Part: "value", Type: "integer"}}, "id=123")
	if r.Values["full"] != "id=123" || r.Values["selected"] != 123 {
		t.Fatal(r.Values)
	}
	for _, field := range []Field{{Name: "x", Method: "regex", Source: `(x)`, Part: "9"}, {Name: "x", Method: "constant", Operations: []Operation{{Op: "regex_extract", Pattern: `(x)`, Group: "missing"}}}} {
		if _, err := Compile(Plan{Version: 1, Fields: []Field{field}}); err == nil {
			t.Fatal("missing regex group was not rejected before execution")
		}
	}
}

func TestNFIConfiguredTimezoneAndMalformedNestedStructures(t *testing.T) {
	r := execute(t, []Field{{Name: "timestamp", Method: "constant", DefaultValue: "2026-11-06 12:00:00", Type: "timestamp", TimestampFormat: "2006-01-02 15:04:05", Timezone: "configured", TimezoneName: "Asia/Kolkata"}}, "")
	if r.Values["timestamp"] != "2026-11-06T12:00:00+05:30" {
		t.Fatal(r.Values)
	}
	r = execute(t, []Field{{Name: "payload", Method: "key_value", Source: "payload"}, {Name: "status", Method: "key_value", Source: "status"}}, `payload=[a=one:b=two]:status=ok`)
	if r.Values["payload"] != "[a=one:b=two]" || r.Values["status"] != "ok" {
		t.Fatal(r.Values)
	}
	c, _ := Compile(Plan{Version: 1, Fields: []Field{{Name: "x", Method: "key_value", Source: "payload"}}})
	if _, err := c.Execute(context.Background(), `payload=[a=one:b=two`); err == nil {
		t.Fatal("unterminated nested structure accepted")
	}
	for _, path := range []string{`$bad`, `a[0]bad`, `["a"]bad`} {
		if _, err := Compile(Plan{Version: 1, Fields: []Field{{Name: "x", Method: "json_path", Source: path}}}); err == nil {
			t.Fatal("malformed JSON path accepted", path)
		}
	}
}

package fieldmapping

import (
	"context"
	"strings"
	"testing"
)

func TestDynamicTailCompleteAndExplicitOmission(t *testing.T) {
	prefix := "20250811 080053105+0100 host smtp 3692 0 140026854221568 Note;SieveRuleMsg "
	tail := Field{Name: "tail_fields", Method: "kv_all", Type: "object", Source: "9", Selection: Selection{Unit: "token"}, OutputPolicy: "flatten", KeyOptions: KeyOptions{RequireComplete: true}}
	plan, err := Compile(Plan{Version: 1, Fields: []Field{tail}})
	if err != nil {
		t.Fatal(err)
	}
	raw := prefix + "status=failed:reason=not needed;server=cloud|future_key=0042\nother=value"
	result, err := plan.Execute(context.Background(), raw)
	if err != nil {
		t.Fatal(err)
	}
	for k, v := range map[string]string{"status": "failed", "reason": "not needed", "server": "cloud", "future_key": "0042", "other": "value"} {
		if result.Values[k] != v {
			t.Fatalf("%s: %+v", k, result)
		}
	}
	if err = plan.ValidateComplete(raw, result); err != nil {
		t.Fatal(err)
	}
	if _, ok := result.Values[tail.Name]; ok {
		t.Fatal("container escaped")
	}
	header, _ := Compile(Plan{Version: 1, Fields: []Field{{Name: "date", Method: "position", Source: "1"}}})
	result, _ = header.Execute(context.Background(), raw)
	if header.ValidateComplete(raw, result) == nil {
		t.Fatal("header-only accepted")
	}
	tail.OutputPolicy = "omit"
	omitter, _ := Compile(Plan{Version: 1, Fields: []Field{tail}})
	result, err = omitter.Execute(context.Background(), raw)
	if err != nil || omitter.ValidateComplete(raw, result) != nil {
		t.Fatalf("explicit omission lost: %v", err)
	}
}

func TestDynamicTailRejectsAmbiguityAndCollisions(t *testing.T) {
	field := Field{Name: "tail", Method: "kv_all", Type: "object", Source: "raw", OutputPolicy: "flatten", KeyOptions: KeyOptions{RequireComplete: true}}
	for _, raw := range []string{"status:failed reason;notneeded server;cloud", "status=failed;reason;notneeded", "a=1:a=2", "raw=secret", "_internal=bad", "a=\"unterminated"} {
		p, err := Compile(Plan{Version: 1, Fields: []Field{field}})
		if err != nil {
			t.Fatal(err)
		}
		if _, err = p.Execute(context.Background(), raw); err == nil {
			t.Fatalf("accepted %q", raw)
		}
	}
	field.AssignmentSeparators = []string{":"}
	p, _ := Compile(Plan{Version: 1, Fields: []Field{field}})
	r, err := p.Execute(context.Background(), "|status:failed|reason:notneeded|server:cloud")
	if err != nil || r.Values["server"] != "cloud" {
		t.Fatalf("explicit colon: %+v %v", r, err)
	}
	field.AssignmentSeparators = nil
	for _, fields := range [][]Field{{{Name: "status", Method: "constant", DefaultValue: "x"}, field}, {field, {Name: "status", Method: "constant", DefaultValue: "x"}}} {
		p, _ := Compile(Plan{Version: 1, Fields: fields})
		if _, err := p.Execute(context.Background(), "status=failed"); err == nil {
			t.Fatal("collision accepted")
		}
	}
	field.KeyExclude = []string{"status"}
	field.KeyPrefix = "custom_"
	p, _ = Compile(Plan{Version: 1, Fields: []Field{field}})
	r, err = p.Execute(context.Background(), "status=failed:server=cloud")
	if err != nil || len(r.Values) != 1 || r.Values["custom_server"] != "cloud" {
		t.Fatalf("filter %+v %v", r, err)
	}
	field.KeyExclude = nil
	field.KeyPrefix = ""
	field.Source = "2"
	field.Unit = "byte"
	p, _ = Compile(Plan{Version: 1, Fields: []Field{field}})
	if _, err = p.Execute(context.Background(), "ékey=value"); err == nil || !strings.Contains(err.Error(), "Unicode") {
		t.Fatalf("byte split accepted: %v", err)
	}
}

func TestExplicitColonNamedAndDynamicMappingCompleteness(t *testing.T) {
	raw := "20250811 080053105+0100 host smtp 3692 0 140026854221568 Note;SieveRuleMsg |status:failed|reason:notneeded|server:cloud"
	fields := []Field{}
	for _, name := range []string{"status", "reason", "server"} {
		fields = append(fields, Field{Name: name, Method: "key_value", Source: name, PairDelimiters: ":;| \t\r\n", KeyOptions: KeyOptions{AssignmentSeparators: []string{":"}}})
	}
	for _, plan := range []Plan{{Version: 1, Fields: fields}, {Version: 1, Fields: []Field{{Name: "tail", Method: "kv_all", Type: "object", Source: "9", OutputPolicy: "flatten", KeyOptions: KeyOptions{RequireComplete: true, AssignmentSeparators: []string{":"}}}}}} {
		compiled, err := Compile(plan)
		if err != nil {
			t.Fatal(err)
		}
		r, err := compiled.Execute(context.Background(), raw)
		if err != nil {
			t.Fatal(err)
		}
		if err = compiled.ValidateComplete(raw, r); err != nil {
			t.Fatal(err)
		}
	}
}

func TestKVTailEndOfLineIsExplicitAndBounded(t *testing.T) {
	for _, ending := range []string{"\n", "\r\n", "\r"} {
		field := Field{Name: "tail", Method: "kv_all", Type: "object", Source: "2", EndOfLine: true, OutputPolicy: "flatten", KeyOptions: KeyOptions{RequireComplete: true}}
		p, err := Compile(Plan{Version: 1, Fields: []Field{field}})
		if err != nil {
			t.Fatal(err)
		}
		raw := "prefix status=failed:server=cloud" + ending + "later=must_not_leak"
		result, err := p.Execute(context.Background(), raw)
		if err != nil {
			t.Fatal(err)
		}
		if len(result.Values) != 2 || result.Values["server"] != "cloud" {
			t.Fatalf("EOL crossed line: %+v", result)
		}
		if p.ValidateComplete(raw, result) == nil {
			t.Fatal("later structured evidence silently discarded")
		}
	}
}

func TestPartialSplitCannotClaimWholeStructuredToken(t *testing.T) {
	raw := "prefix status=failed:reason=notneeded:server=cloud"
	p, err := Compile(Plan{Version: 1, Fields: []Field{{Name: "first_pair", Method: "position", Source: "2", Operations: []Operation{{Op: "split", Delimiter: ":", Part: 1}}}}})
	if err != nil {
		t.Fatal(err)
	}
	result, err := p.Execute(context.Background(), raw)
	if err != nil {
		t.Fatal(err)
	}
	if p.ValidateComplete(raw, result) == nil {
		t.Fatal("partial split reused full-token evidence")
	}
}

func TestTailCustomSeparatorsPreserveLiteralPunctuation(t *testing.T) {
	p, err := Compile(Plan{Version: 1, Fields: []Field{{Name: "tail", Method: "kv_all", Type: "object", OutputPolicy: "flatten", PairDelimiters: "|", KeyOptions: KeyOptions{RequireComplete: true}}}})
	if err != nil {
		t.Fatal(err)
	}
	r, err := p.Execute(context.Background(), "reason=literal;punctuation:kept|status=ok")
	if err != nil || r.Values["reason"] != "literal;punctuation:kept" {
		t.Fatalf("custom separator ignored: %+v %v", r, err)
	}
}

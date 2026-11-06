package fieldmapping

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"sync"
	"testing"
)

const mailSample = "20250811 080053105+0100 aaa-gmailrgi009 smtp 3692 0 140026854221568 Note;SieveRuleMsg status=failed:reason=notneeded:server=cloud"

func userPlan() Plan {
	return Plan{Version: 1, Fields: []Field{
		{Name: "date", Method: "position", Source: "1"},
		{Name: "time", Method: "position", Source: "2"},
		{Name: "host", Method: "position", Source: "3"},
		{Name: "service", Method: "position", Source: "4"},
		{Name: "pid", Type: "integer", Method: "position", Source: "5"},
		{Name: "minor", Type: "integer", Method: "position", Source: "6"},
		{Name: "thread", Method: "position", Source: "7"},
		{Name: "loglevel", Method: "delimiter", Source: "8", Delimiter: ";", Part: "1", Confirmed: true},
		{Name: "event", Method: "delimiter", Source: "8", Delimiter: ";", Part: "2", Confirmed: true},
		{Name: "status", Method: "key_value", Source: "status"},
		{Name: "reason", Method: "key_value", Source: "reason"},
		{Name: "server", Method: "key_value", Source: "server"},
	}}
}

func execute(t *testing.T, fields []Field, raw string) Result {
	t.Helper()
	c, err := Compile(Plan{Version: 1, Fields: fields})
	if err != nil {
		t.Fatal(err)
	}
	r, err := c.Execute(context.Background(), raw)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestNFIUserMailGoldenAndReorderedKV(t *testing.T) {
	c, err := Compile(userPlan())
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]any{"date": "20250811", "time": "080053105+0100", "host": "aaa-gmailrgi009", "service": "smtp", "pid": 3692, "minor": 0, "thread": "140026854221568", "loglevel": "Note", "event": "SieveRuleMsg", "status": "failed", "reason": "notneeded", "server": "cloud"}
	for _, raw := range []string{mailSample, strings.Replace(mailSample, "status=failed:reason=notneeded:server=cloud", "server=cloud:reason=notneeded:status=failed", 1)} {
		result, err := c.Execute(context.Background(), raw)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(result.Values, want) {
			t.Fatalf("got %#v want %#v", result.Values, want)
		}
		for name, e := range result.Evidence {
			if !e.Source || e.End <= e.Start {
				t.Errorf("missing observed source evidence for %s: %+v", name, e)
			}
		}
	}
}

func TestNFIQuotedEscapedAndNestedKV(t *testing.T) {
	space := execute(t, []Field{{Name: "host", Method: "key_value", Source: "hostname", PairDelimiters: ":;,"}, {Name: "agent", Method: "key_value", Source: "useragent", PairDelimiters: ":;,"}}, `prefix hostname="edge" useragent="Mozilla/5.0 (test; exact)"`)
	if space.Values["host"] != "edge" || space.Values["agent"] != "Mozilla/5.0 (test; exact)" {
		t.Fatal("whitespace pair boundaries were lost with explicit punctuation config", space.Values)
	}
	raw := `prefix user="sample@example.test:1234567890123456789":reason="not needed:still=one":ip=2001:db8::1:url=https://example.test/path?q=a=b:empty="":id=================:note=a\:hidden=value:server=cloud`
	fields := []Field{}
	for _, name := range []string{"user", "reason", "ip", "url", "empty", "id", "note", "server"} {
		fields = append(fields, Field{Name: name, Method: "key_value", Source: name})
	}
	r := execute(t, fields, raw)
	for key, want := range map[string]string{"user": "sample@example.test:1234567890123456789", "reason": "not needed:still=one", "ip": "2001:db8::1", "url": "https://example.test/path?q=a=b", "empty": "", "id": "================", "note": "a:hidden=value", "server": "cloud"} {
		if r.Values[key] != want {
			t.Errorf("%s=%q want %q", key, r.Values[key], want)
		}
	}
	r = execute(t, []Field{{Name: "action", Method: "key_value", Source: "msg", Operations: []Operation{{Op: "strip_prefix", Value: "MISREPORT="}, {Op: "key_value", Value: "action", Delimiter: ";", Separator: "|"}}}}, `msg=MISREPORT=action;success|sender;example@test|result;accepted:node=edge`)
	if r.Values["action"] != "success" {
		t.Fatal(r.Values)
	}
	for _, raw := range []string{`status="unterminated`, `status='unterminated`, `status=trailing\`} {
		c, _ := Compile(Plan{Version: 1, Fields: []Field{{Name: "status", Method: "key_value", Source: "status"}}})
		if _, err := c.Execute(context.Background(), raw); err == nil {
			t.Errorf("malformed quoting accepted: %q", raw)
		}
	}
}

func TestNFIDuplicatePoliciesAndNullSemantics(t *testing.T) {
	for _, policy := range []string{"reject", "first", "last", "array"} {
		f := Field{Name: "x", Method: "key_value", Source: "x", DuplicatePolicy: policy}
		if policy == "array" {
			f.Type = "array"
		}
		c, err := Compile(Plan{Version: 1, Fields: []Field{f}})
		if err != nil {
			t.Fatal(err)
		}
		r, err := c.Execute(context.Background(), "x=one:x=two")
		if policy == "reject" {
			if err == nil {
				t.Fatal("ambiguous duplicate accepted")
			}
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		want := any("one")
		if policy == "last" {
			want = "two"
		}
		if policy == "array" {
			want = []any{"one", "two"}
		}
		if !reflect.DeepEqual(r.Values["x"], want) {
			t.Fatalf("%s got %#v", policy, r.Values)
		}
	}
	r := execute(t, []Field{{Name: "empty", Method: "key_value", Source: "empty"}, {Name: "absent", Method: "key_value", Source: "not_here", Policy: "optional"}, {Name: "nullable", Method: "key_value", Source: "not_here", Policy: "nullable"}, {Name: "fallback", Method: "key_value", Source: "not_here", Policy: "default", DefaultValue: "explicit"}}, `empty=""`)
	if r.Values["empty"] != "" || r.Values["fallback"] != "explicit" {
		t.Fatal(r.Values)
	}
	if _, ok := r.Values["absent"]; ok {
		t.Fatal("optional absence was fabricated")
	}
	if value, ok := r.Values["nullable"]; !ok || value != nil {
		t.Fatal("null was not preserved")
	}
	c, _ := Compile(Plan{Version: 1, Fields: []Field{{Name: "reject", Method: "key_value", Source: "absent", Policy: "nullable", NullPolicy: "reject"}}})
	if _, err := c.Execute(context.Background(), "other=one"); err == nil {
		t.Fatal("reject null policy ignored")
	}
}

func TestNFIOperationsAndPrecisePositions(t *testing.T) {
	r := execute(t, []Field{
		{Name: "unicode", Method: "character", Source: "2", SourceEnd: "3"},
		{Name: "bytes", Method: "byte", Source: "2", SourceEnd: "6"},
		{Name: "word", Method: "line", Source: "2", Operations: []Operation{{Op: "split", Delimiter: "|", Part: 2}, {Op: "trim"}, {Op: "replace", Find: "-", Replace: "_"}, {Op: "remove", Find: "!"}, {Op: "uppercase"}, {Op: "prefix", Value: "["}, {Op: "suffix", Value: "]"}}},
		{Name: "joined", Method: "derived", Source: "unicode,word", JoinSeparator: " / "},
	}, "Aé猫Z\nalpha| beta-word!|tail")
	if r.Values["unicode"] != "é猫" || r.Values["bytes"] != "é猫" || r.Values["word"] != "[BETA_WORD]" || r.Values["joined"] != "é猫 / [BETA_WORD]" {
		t.Fatal(r.Values)
	}
	if r.Evidence["word"].Source || r.Evidence["joined"].Source {
		t.Fatal("derived values falsely counted as untouched source")
	}
	c, _ := Compile(Plan{Version: 1, Fields: []Field{{Name: "bad", Method: "byte", Source: "2", SourceEnd: "2"}}})
	if _, err := c.Execute(context.Background(), "Aé"); err == nil {
		t.Fatal("UTF8 split accepted")
	}
	r = execute(t, []Field{{Name: "list", Method: "constant", DefaultValue: "a|b|c", Type: "array", Operations: []Operation{{Op: "split", Delimiter: "|"}}}, {Name: "selected", Method: "constant", DefaultValue: "a|b|c", Operations: []Operation{{Op: "split", Delimiter: "|"}, {Op: "select", Part: 2}}}, {Name: "join", Method: "constant", DefaultValue: "a|b|c", Operations: []Operation{{Op: "split", Delimiter: "|"}, {Op: "join", Separator: " / "}}}}, "")
	if r.Values["selected"] != "b" || r.Values["join"] != "a / b / c" {
		t.Fatal(r.Values)
	}
}

func TestNFIQuotedCSVAndJSONPaths(t *testing.T) {
	r := execute(t, []Field{{Name: "csv", Method: "delimiter", Source: "raw", Delimiter: ",", Part: "2", Confirmed: true}, {Name: "csvquote", Method: "delimiter", Source: "raw", Delimiter: ",", Part: "3", Confirmed: true}}, `alpha,"beta,gamma","a""b"`)
	if r.Values["csv"] != "beta,gamma" || r.Values["csvquote"] != `a"b` {
		t.Fatal(r.Values)
	}
	r = execute(t, []Field{{Name: "quoted", Method: "json_path", Source: `$["a.b"][0].value`}, {Name: "obj", Method: "json_path", Source: "$.nested", Type: "object"}, {Name: "array", Method: "json_path", Source: "$.items", Type: "array"}}, `{"a.b":[{"value":"exact"}],"nested":{"ready":true},"items":[1,"x"]}`)
	if r.Values["quoted"] != "exact" {
		t.Fatal(r.Values)
	}
	c, _ := Compile(Plan{Version: 1, Fields: []Field{{Name: "value", Method: "json_path", Source: "$.value"}}})
	for _, raw := range []string{`{"value":1,"value":2}`, `{"value":1} trailing`, strings.Repeat("[", MaxDepth+1) + strings.Repeat("]", MaxDepth+1)} {
		if _, err := c.Execute(context.Background(), raw); err == nil {
			t.Errorf("invalid JSON accepted %q", raw)
		}
	}
}

func TestNFITypedValidationAndTimestamps(t *testing.T) {
	r := execute(t, []Field{{Name: "instant", Method: "position", Source: "1", SourceEnd: "2", Type: "timestamp", TimestampFormat: "compact_mail", Timezone: "utc"}, {Name: "epoch", Method: "constant", DefaultValue: "0", Type: "timestamp", TimestampFormat: "epoch_seconds"}, {Name: "port", Method: "constant", DefaultValue: "65535", Type: "port", Validation: "min=1,max=65535"}, {Name: "bool", Method: "constant", DefaultValue: "true", Type: "boolean"}}, "20250811 080053105+0100")
	if r.Values["instant"] != "2025-08-11T07:00:53.105Z" || r.Values["epoch"] != "1970-01-01T00:00:00Z" || r.Values["port"] != 65535 || r.Values["bool"] != true {
		t.Fatal(r.Values)
	}
	for _, f := range []Field{{Name: "x", Method: "constant", DefaultValue: "9007199254740993", Type: "integer"}, {Name: "x", Method: "constant", DefaultValue: "NaN", Type: "number"}, {Name: "x", Method: "constant", DefaultValue: "1", Type: "boolean"}, {Name: "x", Method: "constant", DefaultValue: "999.1.2.3", Type: "ip"}, {Name: "x", Method: "constant", DefaultValue: "20250230 080053105+0100", Type: "timestamp", TimestampFormat: "compact_mail"}, {Name: "x", Method: "constant", DefaultValue: "ab", Validation: "min_length=3"}} {
		c, err := Compile(Plan{Version: 1, Fields: []Field{f}})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := c.Execute(context.Background(), ""); err == nil {
			t.Errorf("invalid typed value accepted: %+v", f)
		}
	}
}

func TestNFIHostileBoundsAndConcurrentCompiledPlan(t *testing.T) {
	base := Field{Name: "x", Method: "position", Source: "1"}
	for _, f := range []Field{{Name: "x", Method: "position", Source: "0"}, {Name: "x", Method: "derived", Source: "later"}, {Name: "x", Method: "regex", Source: `(?<=a)b`}, {Name: "x", Method: "regex", Source: strings.Repeat("x", MaxRegexBytes+1)}, {Name: "x", Method: "constant", Operations: make([]Operation, MaxOperations+1)}, {Name: "_secret", Method: "position", Source: "1"}} {
		if _, err := Compile(Plan{Version: 1, Fields: []Field{f}}); err == nil {
			t.Errorf("invalid plan accepted %+v", f)
		}
	}
	c, err := Compile(Plan{Version: 1, Fields: []Field{base}})
	if err != nil {
		t.Fatal(err)
	}
	for _, raw := range []string{strings.Repeat("x", MaxInputBytes+1), "x\x00y", string([]byte{0xff})} {
		if _, err := c.Execute(context.Background(), raw); err == nil {
			t.Fatal("hostile input accepted")
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := c.Execute(ctx, "x"); err == nil {
		t.Fatal("cancellation ignored")
	}
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r, err := c.Execute(context.Background(), "safe")
			if err != nil || r.Values["x"] != "safe" {
				t.Errorf("concurrent execution failed: %v", err)
			}
		}()
	}
	wg.Wait()
	plan := Plan{Version: 1, Fields: []Field{{Name: "x", Method: "constant", DefaultValue: "a", Operations: []Operation{{Op: "suffix", Value: "b"}}}}}
	c, _ = Compile(plan)
	plan.Fields[0].Operations[0].Value = "mutated"
	r, err := c.Execute(context.Background(), "")
	if err != nil || r.Values["x"] != "ab" {
		t.Fatal("compiled plan was mutable")
	}
}

func FuzzNFIPlanNeverPanics(f *testing.F) {
	for _, s := range []string{mailSample, `a="quoted:key=one":b=two`, `{"nested":[1,true]}`, "é猫\nαβ"} {
		f.Add(s)
	}
	c, err := Compile(Plan{Version: 1, Fields: []Field{{Name: "first", Method: "position", Source: "1", Policy: "optional"}, {Name: "value", Method: "key_value", Source: "a", Policy: "optional"}, {Name: "tail", Method: "character", Source: "2", SourceEnd: "3", Policy: "optional"}}})
	if err != nil {
		f.Fatal(err)
	}
	f.Fuzz(func(t *testing.T, raw string) {
		if len(raw) > 64<<10 {
			return
		}
		r, err := c.Execute(context.Background(), raw)
		if err == nil {
			if _, err := json.Marshal(r.Values); err != nil {
				t.Fatal(err)
			}
		}
	})
}

func BenchmarkNFIMailMapping(b *testing.B) {
	c, _ := Compile(userPlan())
	b.ReportAllocs()
	b.SetBytes(int64(len(mailSample)))
	for i := 0; i < b.N; i++ {
		if _, err := c.Execute(context.Background(), mailSample); err != nil {
			b.Fatal(err)
		}
	}
}

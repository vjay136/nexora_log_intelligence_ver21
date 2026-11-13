package fieldmapping

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestStructuralMailHeaderIntegrity(t *testing.T) {
	const raw = "20250811 080053105+0100 aaa-gmailrgi009 smtp 3692 0 140026854221568 123456;Note;SieveRuleMsg status=failed;reason=not needed"
	p := Plan{Version: 1}
	for index, name := range []string{"date", "timestamp", "hostname", "service", "pid", "minor_pid", "thread_id"} {
		p.Fields = append(p.Fields, Field{Name: name, Method: "position", Source: string(rune('1' + index)), Type: "string", Policy: "required"})
	}
	for index, name := range []string{"random_number", "loglevel", "event"} {
		p.Fields = append(p.Fields, Field{Name: name, Method: "delimiter", Source: "8", Delimiter: ";", Part: string(rune('1' + index)), Type: "string", Policy: "required", Confirmed: true})
	}
	p.Fields = append(p.Fields, Field{Name: "tail", Method: "kv_all", Source: "9", Type: "object", Policy: "required", OutputPolicy: "flatten", KeyOptions: KeyOptions{RequireComplete: true}})
	complete, err := Compile(p)
	if err != nil {
		t.Fatal(err)
	}
	result, err := complete.Execute(context.Background(), raw)
	if err != nil {
		t.Fatal(err)
	}
	if err := complete.ValidateComplete(raw, result); err != nil {
		t.Fatal(err)
	}
	for key, want := range map[string]string{"status": "failed", "reason": "not needed"} {
		proof := result.Evidence[key]
		if proof.Start < 0 || proof.End > len(raw) || raw[proof.Start:proof.End] != want {
			t.Fatalf("%s: source proof %+v, expected %q", key, proof, want)
		}
	}
	malformed := strings.Replace(raw, "Note;SieveRuleMsg", "extra;another;Note;SieveRuleMsg", 1)
	if !HasMailHeaderExtension(malformed) || !errors.Is(ValidateMailHeader(malformed, result.Evidence), ErrIncompleteHeader) {
		t.Fatal("malformed recognized prefix bypassed header integrity")
	}
	omittedPlan := Plan{Version: 1, Fields: append([]Field(nil), p.Fields...)}
	omittedPlan.Fields[9].OutputPolicy = "omit"
	omitted, err := Compile(omittedPlan)
	if err != nil {
		t.Fatal(err)
	}
	omittedResult, err := omitted.Execute(context.Background(), raw)
	if err != nil {
		t.Fatal(err)
	}
	if err := omitted.ValidateComplete(raw, omittedResult); err != nil {
		t.Fatalf("explicit omission is accounted for: %v", err)
	}
	omissions := omitted.HeaderOmissions(omittedResult)
	if proof, ok := omissions["event"]; !ok || raw[proof.Start:proof.End] != "SieveRuleMsg" {
		t.Fatalf("omission has no exact source proof: %+v", omissions)
	}
	if _, ok := omittedResult.Values["event"]; ok {
		t.Fatal("omitted event leaked to output")
	}
	for _, test := range []struct {
		name  string
		setup func(fields []Field)
	}{
		{"omitted event", func(fields []Field) { fields[9].Part = "2"; fields[9].Name = "extra" }},
		{"shifted semantics", func(fields []Field) { fields[7].Name = "event"; fields[9].Name = "random_number" }},
		{"unproven broad capture", func(fields []Field) { fields[9].Method = "position"; fields[9].Name = "extra"; fields[9].Part = "" }},
	} {
		t.Run(test.name, func(t *testing.T) {
			draft := Plan{Version: 1, Fields: append([]Field(nil), p.Fields...)}
			test.setup(draft.Fields)
			compiled, err := Compile(draft)
			if err != nil {
				t.Fatal(err)
			}
			parsed, err := compiled.Execute(context.Background(), raw)
			if err != nil {
				t.Fatal(err)
			}
			if err := compiled.ValidateComplete(raw, parsed); !errors.Is(err, ErrIncompleteHeader) {
				t.Fatalf("expected structural header rejection, got %v", err)
			}
		})
	}
}

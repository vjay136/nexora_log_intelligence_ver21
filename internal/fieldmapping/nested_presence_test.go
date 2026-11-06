package fieldmapping

import (
	"context"
	"strings"
	"testing"
)

func nestedReasonField(policy string) Field {
	return Field{Name: "msg_reason", Method: "key_value", Source: "msg", Policy: policy, Operations: []Operation{{Op: "strip_prefix", Value: "MISREPORT="}, {Op: "key_value", Value: "reason", Delimiter: ";", Separator: "|"}}}
}

func TestNFIOptionalNestedKVPresenceAcrossMergedProfileRecords(t *testing.T) {
	plan := Plan{Version: 1, Fields: []Field{{Name: "node", Method: "key_value", Source: "node"}, nestedReasonField("optional")}}
	compiled, err := Compile(plan)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		raw, want string
		present   bool
	}{
		{`msg=MISREPORT=action;success|reason;delivered:node=edge`, "delivered", true},
		{`msg=MISREPORT=action;success|sender;sample@example.test:node=edge`, "", false},
		{`msg="MISREPORT=action;success|reason;":node=edge`, "", true},
	} {
		result, err := compiled.Execute(context.Background(), tc.raw)
		if err != nil {
			t.Fatalf("optional nested absence rejected a matching profile record: %v", err)
		}
		value, present := result.Values["msg_reason"]
		if present != tc.present || present && value != tc.want {
			t.Fatalf("nested reason=%#v present=%v; want=%q present=%v", value, present, tc.want, tc.present)
		}
		if result.Values["node"] != "edge" {
			t.Fatal("optional absence changed the independent required field")
		}
		if !tc.present {
			if _, exists := result.Evidence["msg_reason"]; exists {
				t.Fatal("omitted optional value acquired fabricated evidence")
			}
		}
	}
}

func TestNFINestedMissingDefaultsAndNullsBeforeFinalValidation(t *testing.T) {
	raw := `msg=MISREPORT=action;success|sender;sample@example.test`
	for _, tc := range []struct {
		policy, nullPolicy, typ, fallback string
		want                              any
		wantError                         bool
	}{
		{policy: "required", wantError: true},
		{policy: "nullable", want: nil},
		{policy: "optional", nullPolicy: "preserve_null", want: nil},
		{policy: "optional", nullPolicy: "reject", wantError: true},
		{policy: "default", typ: "integer", fallback: "7", want: 7},
		{policy: "default", typ: "integer", fallback: "not-an-integer", wantError: true},
	} {
		t.Run(tc.policy+"_"+tc.nullPolicy+"_"+tc.fallback, func(t *testing.T) {
			field := nestedReasonField(tc.policy)
			field.NullPolicy = tc.nullPolicy
			field.Type = tc.typ
			field.DefaultValue = tc.fallback
			// A missing nested result must not flow through later string operations.
			field.Operations = append(field.Operations, Operation{Op: "suffix", Value: " must-not-run"})
			compiled, err := Compile(Plan{Version: 1, Fields: []Field{field}})
			if err != nil {
				t.Fatal(err)
			}
			result, err := compiled.Execute(context.Background(), raw)
			if tc.wantError {
				if err == nil {
					t.Fatal("required absence or invalid fallback was accepted")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			value, present := result.Values[field.Name]
			if !present || value != tc.want {
				t.Fatalf("got %#v present=%v, want %#v", value, present, tc.want)
			}
			if result.Evidence[field.Name].Source || result.Evidence[field.Name].Operations != 2 {
				t.Fatalf("fallback evidence misrepresented executed pipeline: %+v", result.Evidence[field.Name])
			}
		})
	}
}

func TestNFIOptionalNestedKVDoesNotSwallowMalformedOrInvalidValues(t *testing.T) {
	for _, policy := range []string{"optional", "default", "nullable"} {
		field := nestedReasonField(policy)
		field.DefaultValue = "fallback"
		compiled, err := Compile(Plan{Version: 1, Fields: []Field{field}})
		if err != nil {
			t.Fatal(err)
		}
		for _, raw := range []string{
			`msg="MISREPORT=action;success|reason;\"unterminated"`,
			`msg=MISREPORT=action;success|reason;one|reason;two`,
		} {
			if _, err := compiled.Execute(context.Background(), raw); err == nil {
				t.Fatalf("%s swallowed malformed/duplicate nested input", policy)
			} else if !strings.Contains(err.Error(), "quoted") && !strings.Contains(err.Error(), "duplicated") {
				t.Fatalf("unexpected rejection reason: %v", err)
			}
		}
		field.Type = "integer"
		field.DefaultValue = "7"
		compiled, err = Compile(Plan{Version: 1, Fields: []Field{field}})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := compiled.Execute(context.Background(), `msg=MISREPORT=action;success|reason;not-a-number`); err == nil {
			t.Fatalf("%s swallowed a present value's type failure", policy)
		}
	}
}

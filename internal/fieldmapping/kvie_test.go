package fieldmapping

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"nexora.local/nli-v20/internal/mixedkv"
)

const kvieMailHeader = "20250811 080053105+0100 aaa-gmailrgi009 smtp 3692 0 140026854221568 Note;SieveRuleMsg "

func TestKVIEOnePlanForReportedMailTails(t *testing.T) {
	field := Field{
		Name: "tail", Method: "kv_all", Type: "object", Source: "9",
		EndOfLine: true, OutputPolicy: "flatten", KVProfile: "auto",
	}
	plan, err := Compile(Plan{Version: 1, Fields: []Field{field}})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]any{"status": "failed", "reason": "notneeded", "server": "cloud"}
	for _, tail := range []string{
		"|status:failed|reason:notneeded|server:cloud",
		"|status:failed|reason:notneeded|server:cloud|",
		"status:failed reason;notneeded server;cloud",
		"status=failed:reason=notneeded:server=cloud",
		"{status=failed:reason=notneeded:server=cloud}",
		"status=failed reason=notneeded server=cloud",
		"status:failed,reason:notneeded,server:cloud",
	} {
		t.Run(tail, func(t *testing.T) {
			raw := kvieMailHeader + tail
			profiles, err := InspectKeyValues(context.Background(), raw, field)
			if err != nil || len(profiles) != 1 {
				t.Fatalf("inspect: %+v, %v", profiles, err)
			}
			for _, observation := range profiles[0].Observations {
				if observation.Start < len(kvieMailHeader) || raw[observation.Start:observation.End] != want[observation.Key] {
					t.Fatalf("wrong source evidence: %+v", observation)
				}
			}
			result, err := plan.Execute(context.Background(), raw)
			if err != nil || !reflect.DeepEqual(result.Values, want) {
				t.Fatalf("execute: %+v, %v", result.Values, err)
			}
			if err := plan.ValidateComplete(raw, result); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestKVIEOptInNestedObjects(t *testing.T) {
	raw := `status:ok,details:{phase:ready,env:{stage=prod;inner=[zone=west id=0042]}},note:(plain text)`
	field := Field{Name: "tail", Method: "kv_all", Type: "object", OutputPolicy: "flatten", KVProfile: "auto"}
	for _, nested := range []bool{false, true} {
		field.KVNestedObjects = nested
		plan, err := Compile(Plan{Version: 1, Fields: []Field{field}})
		if err != nil {
			t.Fatal(err)
		}
		result, err := plan.Execute(context.Background(), raw)
		if err != nil {
			t.Fatal(err)
		}
		if err := plan.ValidateComplete(raw, result); err != nil {
			t.Fatal(err)
		}
		if result.Values["note"] != "(plain text)" {
			t.Fatalf("plain bracketed value changed: %#v", result.Values)
		}
		var want any = "{phase:ready,env:{stage=prod;inner=[zone=west id=0042]}}"
		if nested {
			want = map[string]any{"phase": "ready", "env": map[string]any{
				"stage": "prod", "inner": map[string]any{"zone": "west", "id": "0042"},
			}}
		}
		if !reflect.DeepEqual(result.Values["details"], want) {
			t.Fatalf("nested=%v, got %#v, want %#v", nested, result.Values["details"], want)
		}
	}
	plan, err := Compile(Plan{Version: 1, Fields: []Field{field}})
	if err != nil {
		t.Fatal(err)
	}
	tooDeep := "status:ok,details:" + strings.Repeat("{a=", MaxDepth+1) + "value" + strings.Repeat("}", MaxDepth+1)
	if _, err := plan.Execute(context.Background(), tooDeep); !errors.Is(err, ErrIncompleteKV) || !strings.Contains(err.Error(), "depth limit") {
		t.Fatalf("excessive nesting should fail with a bounded diagnostic: %v", err)
	}
}

func TestKVIEConfiguredNamedFieldsShareScopeAndRejectPartialMapping(t *testing.T) {
	raw := kvieMailHeader + "|status:failed|ip:2001:db8::1|url:https://example.org:8443/a?x=y"
	fields := []Field{}
	for _, key := range []string{"status", "ip", "url"} {
		fields = append(fields, Field{
			Name: key, Method: "key_value", Source: key, KVStart: "9", EndOfLine: true, KVProfile: "configured",
			PairDelimiters: "|", KeyOptions: KeyOptions{AssignmentSeparators: []string{":"}},
		})
	}
	for _, n := range []int{len(fields), 2} {
		plan, err := Compile(Plan{Version: 1, Fields: fields[:n]})
		if err != nil {
			t.Fatal(err)
		}
		result, err := plan.Execute(context.Background(), raw)
		if err != nil || result.Values["ip"] != "2001:db8::1" || len(result.kvieScans) != 1 {
			t.Fatalf("execute: %+v, %v", result, err)
		}
		err = plan.ValidateComplete(raw, result)
		if n == 2 {
			if err == nil || !strings.Contains(err.Error(), "url") {
				t.Fatalf("incomplete extraction passed: %v", err)
			}
		} else if err != nil || result.Values["url"] != "https://example.org:8443/a?x=y" {
			t.Fatalf("complete extraction: %+v, %v", result.Values, err)
		}
	}
}

func TestKVIEQuotedEscapedAndBracketedValues(t *testing.T) {
	raw := `message="é value; left=right" env=(dev stage=local) link=<a b=c> token=a\|b empty= id=001 id=002`
	field := Field{
		Name: "tail", Method: "kv_all", Type: "object", KVProfile: "auto",
		DuplicatePolicy: "last", OutputPolicy: "flatten",
	}
	plan, err := Compile(Plan{Version: 1, Fields: []Field{field}})
	if err != nil {
		t.Fatal(err)
	}
	got, err := plan.Execute(context.Background(), raw)
	want := map[string]any{
		"message": "é value; left=right", "env": "(dev stage=local)", "link": "<a b=c>",
		"token": "a|b", "empty": "", "id": "002",
	}
	if err != nil || !reflect.DeepEqual(got.Values, want) {
		t.Fatalf("got %+v, %v", got.Values, err)
	}
	if _, err := plan.Execute(context.Background(), `message="unterminated reason=bad`); err == nil {
		t.Fatal("unterminated value was accepted")
	}
}

func TestKVIEConfiguredLexicalOptions(t *testing.T) {
	field := Field{
		Name: "tail", Method: "kv_all", Type: "object", KVProfile: "configured",
		KVPairSeparators: []string{"||"}, KVQuotes: "'", KVEscape: "~",
		KVBrackets: &map[string]string{"<": ">"}, KVKeyPattern: `[\pL_][\pL\pN_.-]*`,
		KVRejectEmpty: true, KeyOptions: KeyOptions{AssignmentSeparators: []string{"=>"}},
	}
	plan, err := Compile(Plan{Version: 1, Fields: []Field{field}})
	if err != nil {
		t.Fatal(err)
	}
	raw := "||port=>042||café=>'left||right'||detail=><a||b>||token=>a~||b||"
	want := map[string]any{"port": "042", "café": "left||right", "detail": "<a||b>", "token": "a||b"}
	result, err := plan.Execute(context.Background(), raw)
	if err != nil || !reflect.DeepEqual(result.Values["tail"], want) {
		t.Fatalf("configured output: %+v, %v", result.Values, err)
	}
	_, err = plan.Execute(context.Background(), "port=>")
	var lexical *mixedkv.Error
	if !errors.Is(err, ErrIncompleteKV) || !errors.As(err, &lexical) || lexical.Code != mixedkv.CodeEmptyValue {
		t.Fatalf("configured empty error: %v", err)
	}
}

func TestKVIEEmptyBracketConfigurationSurvivesPackRoundTrip(t *testing.T) {
	field := Field{
		Name: "tail", Method: "kv_all", Type: "object", KVProfile: "configured",
		KVBrackets: &map[string]string{},
	}
	serialized, err := json.Marshal(Plan{Version: 1, Fields: []Field{field}})
	if err != nil {
		t.Fatal(err)
	}
	var loaded Plan
	if err := json.Unmarshal(serialized, &loaded); err != nil {
		t.Fatal(err)
	}
	plan, err := Compile(loaded)
	if err != nil {
		t.Fatal(err)
	}
	got, err := plan.Execute(context.Background(), "literal=unmatched> reason=ok")
	want := map[string]any{"literal": "unmatched>", "reason": "ok"}
	if err != nil || !reflect.DeepEqual(got.Values["tail"], want) {
		t.Fatalf("reloaded output: %+v, %v", got.Values, err)
	}
}

func TestKVIEDiagnosticPointsToOriginalMalformedSource(t *testing.T) {
	field := Field{Name: "tail", Method: "kv_all", Type: "object", Source: "9", KVProfile: "auto"}
	raw := kvieMailHeader + `|status:"unterminated|reason:failed`
	_, err := InspectKeyValues(context.Background(), raw, field)
	var lexical *mixedkv.Error
	if !errors.Is(err, ErrIncompleteKV) || !errors.As(err, &lexical) || lexical.Code != mixedkv.CodeMalformedQuote || lexical.Offset != len(raw) {
		t.Fatalf("diagnostic: %v", err)
	}
	_, err = InspectKeyValues(context.Background(), kvieMailHeader+"ordinary text", field)
	if err == nil || errors.Is(err, ErrIncompleteKV) {
		t.Fatalf("unrelated source classified as structured: %v", err)
	}
}

func TestKVIEKeyValueEditsPreserveCoverage(t *testing.T) {
	field := Field{
		Name: "tail", Method: "kv_all", Type: "object", Source: "9",
		EndOfLine: true, OutputPolicy: "flatten", KVProfile: "auto",
		KeyOptions: KeyOptions{RequireComplete: true, KeyAliases: map[string]string{"server": "backend"}, KeyPrefix: "kv_"},
		KeyValueEdits: []KeyValueEdit{
			{Key: "kv_status", Action: "omit"},
			{Key: "kv_reason", Action: "blank"},
			{Key: "kv_backend", Action: "replace", Value: "redacted"},
			{Key: "kv_absent", Action: "replace", Value: "must not be emitted"},
		},
	}
	raw := kvieMailHeader + "status:failed,reason:notneeded,server:cloud,result:ok"
	plan, err := Compile(Plan{Version: 1, Fields: []Field{field}})
	if err != nil {
		t.Fatal(err)
	}
	result, err := plan.Execute(context.Background(), raw)
	want := map[string]any{"kv_reason": "", "kv_backend": "redacted", "kv_result": "ok"}
	if err != nil || !reflect.DeepEqual(result.Values, want) {
		t.Fatalf("execute: %+v, %v", result.Values, err)
	}
	if err := plan.ValidateComplete(raw, result); err != nil {
		t.Fatal(err)
	}
	field.KeyValueEdits = append(field.KeyValueEdits, KeyValueEdit{Key: "kv_status", Action: "blank"})
	if _, err := Compile(Plan{Version: 1, Fields: []Field{field}}); err == nil {
		t.Fatal("duplicate edit should be rejected")
	}
}

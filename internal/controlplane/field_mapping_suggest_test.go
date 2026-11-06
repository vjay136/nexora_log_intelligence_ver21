package controlplane

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"nexora.local/nli-v20/internal/fieldmapping"
)

func executeNFISuggestedFields(t *testing.T, fields []fieldmapping.Field, raw string) map[string]any {
	t.Helper()
	copyFields := append([]fieldmapping.Field(nil), fields...)
	for i := range copyFields {
		if copyFields[i].Method == "delimiter" {
			copyFields[i].Confirmed = true
		}
	}
	compiled, err := fieldmapping.Compile(fieldmapping.Plan{Version: 1, Fields: copyFields})
	if err != nil {
		t.Fatal(err)
	}
	result, err := compiled.Execute(context.Background(), raw)
	if err != nil {
		t.Fatal(err)
	}
	return result.Values
}

func TestNFISuggestionsRequestedColonSampleAndVariants(t *testing.T) {
	raw := "20250811 080053105+0100 aaa-gmailrgi009 smtp 3692 0 140026854221568 Note;SieveRuleMsg status=failed:reason=notneeded:server=cloud"
	family, fields, err := nfiSuggestRecord(context.Background(), raw, "")
	if err != nil || family != "mail-header/smtp" {
		t.Fatalf("family=%q err=%v", family, err)
	}
	got := executeNFISuggestedFields(t, fields, raw)
	for name, want := range map[string]string{"loglevel": "Note", "event": "SieveRuleMsg", "status": "failed", "reason": "notneeded", "server": "cloud", "thread_id": "140026854221568", "pid": "3692"} {
		if fmt.Sprint(got[name]) != want {
			t.Fatalf("%s=%v want %s", name, got[name], want)
		}
	}
	if _, ok := got["thread_id"].(string); !ok {
		t.Fatal("thread identifier lost string precision")
	}
	variant := "20310102 010203004-0400 synthetic-host smtp 88 9 000001 Info;OtherEvent status=ok;reason=not needed"
	_, fields, err = nfiSuggestRecord(context.Background(), variant, "")
	if err != nil {
		t.Fatal(err)
	}
	got = executeNFISuggestedFields(t, fields, variant)
	if got["event"] != "OtherEvent" || got["reason"] != "not needed" || got["thread_id"] != "000001" {
		t.Fatalf("non-hardcoded variant failed: %v", got)
	}
}

func TestNFISuggestionsCorpusProfilesOptionalAndQuotedValues(t *testing.T) {
	samples := []string{
		`2031-01-02 01:02:03+0000 agent="Example Client status=fake" session="" duration="3"`,
		`2031-01-02 01:02:04+0000 agent="Another Client" duration="4"`,
		`20310102 010203004+0000 synthetic-host imap 9 0 0007 Note;Connection(10/2) user=person@example.invalid:00042:mss=host-a:duration=2`,
	}
	profiles, diagnostics := nfiSuggestCorpus(context.Background(), samples, "")
	if len(profiles) != 2 || len(diagnostics) == 0 {
		t.Fatalf("profiles=%d diagnostics=%v", len(profiles), diagnostics)
	}
	quoted := profiles[0]
	if quoted.SampleCount != 2 {
		t.Fatal("quoted records did not group structurally")
	}
	for _, field := range quoted.Fields {
		if field.Name == "status" {
			t.Fatal("quoted inner text became a fake outer key")
		}
		if field.Name == "session" && field.Policy != "optional" {
			t.Fatal("absent session was not inferred optional")
		}
	}
	got := executeNFISuggestedFields(t, quoted.Fields, samples[0])
	if got["session"] != "" || got["agent"] != "Example Client status=fake" {
		t.Fatalf("quoted/empty values changed: %v", got)
	}
	got = executeNFISuggestedFields(t, profiles[1].Fields, samples[2])
	if got["user"] != "person@example.invalid:00042" || got["event"] != "Connection" || got["event_code"] != "10/2" {
		t.Fatalf("opaque user or compound event changed: %v", got)
	}
}

func TestNFISuggestionsNestedAndBoundaryPreservation(t *testing.T) {
	raw := `url=https://example.invalid/a:b?x=y:ip=2001:db8::1:msg=REPORT=eventType;summary|action;keep|geo;ZZ:status=ok`
	_, fields, err := nfiSuggestRecord(context.Background(), raw, "")
	if err != nil {
		t.Fatal(err)
	}
	got := executeNFISuggestedFields(t, fields, raw)
	for key, want := range map[string]string{"url": "https://example.invalid/a:b?x=y", "ip": "2001:db8::1", "msg_action": "keep", "msg_geo": "ZZ", "msg_eventType": "summary", "status": "ok"} {
		if got[key] != want {
			t.Fatalf("%s=%v want %s", key, got[key], want)
		}
	}
	if got["msg"] != "REPORT=eventType;summary|action;keep|geo;ZZ" {
		t.Fatal("discovery discarded the original nested payload")
	}
}

func TestNFISuggestionsJSONAndQuotedDelimitedPlansMatchRuntime(t *testing.T) {
	jsonRaw := `{"user":{"name":"Synthetic","id":9007199254740993},"enabled":true,"tags":["a","b"],"empty":null}`
	_, fields, err := nfiSuggestRecord(context.Background(), jsonRaw, "")
	if err != nil {
		t.Fatal(err)
	}
	got := executeNFISuggestedFields(t, fields, jsonRaw)
	if got["user_name"] != "Synthetic" || got["user_id"] != "9007199254740993" || got["enabled"] != true || got["empty"] != nil {
		t.Fatalf("JSON proposal diverged: %v", got)
	}
	csvRaw := `"one,two",003,"say ""hello""",`
	_, fields, err = nfiSuggestRecord(context.Background(), csvRaw, ",")
	if err != nil {
		t.Fatal(err)
	}
	got = executeNFISuggestedFields(t, fields, csvRaw)
	for key, want := range map[string]string{"column_1": "one,two", "column_2": "003", "column_3": `say "hello"`, "column_4": ""} {
		if got[key] != want {
			t.Fatalf("CSV %s=%v want %s", key, got[key], want)
		}
	}
	_, fields, err = nfiSuggestRecord(context.Background(), `{"a.b":"opaque","name":"synthetic"}`, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(fields) != 1 || fields[0].Source != "$" {
		t.Fatal("unsupported structured keys were silently omitted")
	}
	_, fields, err = nfiSuggestRecord(context.Background(), `alpha=one,beta=two`, ",")
	if err != nil {
		t.Fatal(err)
	}
	got = executeNFISuggestedFields(t, fields, `alpha=one,beta=two`)
	if got["column_1"] != "alpha=one" || got["column_2"] != "beta=two" {
		t.Fatal("explicit CSV separator lost precedence")
	}
	if _, _, err = nfiSuggestRecord(context.Background(), `"unterminated,value`, ","); err == nil {
		t.Fatal("malformed explicitly delimited input was treated as a valid proposal")
	}
}

func TestNFISuggestionsMalformedAndTruncatedDoNotFabricate(t *testing.T) {
	profiles, diagnostics := nfiSuggestCorpus(context.Background(), []string{`agent="unfinished`, `{"x":1,"x":2}`, "a\x00b"}, "")
	if len(profiles) != 0 || len(diagnostics) < 3 {
		t.Fatalf("invalid evidence produced mappings: %v %v", profiles, diagnostics)
	}
	full := "20310102 010203004+0000 synthetic-host smtp 9 0 7 Note;Action key=value:extra=present"
	truncated := "20310102 010203004+0000 synthetic-host smtp 9 0 7 Note;Action"
	profiles, _ = nfiSuggestCorpus(context.Background(), []string{full, truncated}, "")
	if len(profiles) != 1 {
		t.Fatalf("compatible header structures split unexpectedly: %v", profiles)
	}
	got := executeNFISuggestedFields(t, profiles[0].Fields, truncated)
	if _, ok := got["key"]; ok {
		t.Fatal("missing truncated value was fabricated")
	}
}

func TestNFISuggestionsHTTPBoundsAndNoPersistence(t *testing.T) {
	s := &Server{}
	for _, test := range []struct {
		name, body string
		status     int
	}{
		{"empty", `{"samples":[]}`, http.StatusBadRequest},
		{"separator", `{"samples":["x"],"separator":"||"}`, http.StatusBadRequest},
		{"valid", `{"samples":["status=ok:empty=\"\""],"separator":""}`, http.StatusOK},
	} {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/field-mapping/suggest", strings.NewReader(test.body))
			response := httptest.NewRecorder()
			s.handleFieldMappingSuggest(response, req)
			if response.Code != test.status {
				t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
			}
			if response.Code == http.StatusOK {
				var body map[string]any
				if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
					t.Fatal(err)
				}
				if body["persisted"] != false || body["activated"] != false {
					t.Fatal("discovery claimed mutation")
				}
			}
		})
	}
	samples := make([]string, 101)
	for i := range samples {
		samples[i] = "x"
	}
	body, _ := json.Marshal(map[string]any{"samples": samples})
	response := httptest.NewRecorder()
	s.handleFieldMappingSuggest(response, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(string(body))))
	if response.Code != http.StatusBadRequest {
		t.Fatal("sample cap was not enforced")
	}
}

func TestNFISuggestionsReservedNamesRemainMappable(t *testing.T) {
	used := map[string]bool{}
	for _, reserved := range fieldmapping.ReservedNames {
		name := nfiSuggestionUniqueName(reserved, used)
		if name == reserved {
			t.Fatalf("reserved name proposed: %s", name)
		}
		plan := fieldmapping.Plan{Version: 1, Fields: []fieldmapping.Field{{Name: name, Method: "constant", Source: "observed", Policy: "default", DefaultValue: "observed"}}}
		if _, err := fieldmapping.Compile(plan); err != nil {
			t.Fatalf("alias rejected: %s: %v", name, err)
		}
	}
}

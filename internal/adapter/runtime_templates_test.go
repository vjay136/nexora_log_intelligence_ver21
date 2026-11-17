package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"nexora.local/nli-v20/internal/engine"
	"nexora.local/nli-v20/internal/fieldmapping"
)

const templateHeaderPrefix = "20250811 080053105+0100 aaa-gmailrgi009 smtp 3692 0 140026854221568 123456;Note;efgh;ijkl;mnop;SieveRuleMsg"

func TestPublishedHeaderTemplateUsesCompleteGlobalKV(t *testing.T) {
	ctx := context.Background()
	baseline, err := NewParserPackAdapter()
	if err != nil {
		t.Fatal(err)
	}
	captures := map[string]string{}
	for _, name := range []string{"date", "timestamp", "hostname", "service", "pid", "minor_pid", "thread_id", "random_number", "event_type", "extra_tag1", "extra_tag2", "extra_tag3", "event"} {
		captures[name] = name
	}
	header := fieldmapping.HeaderTemplate{
		Engine:  "re2",
		Pattern: `^(?P<date>[0-9]{8})[ \t]+(?P<timestamp>\S+)[ \t]+(?P<hostname>\S+)[ \t]+(?P<service>\S+)[ \t]+(?P<pid>[0-9]+)[ \t]+(?P<minor_pid>[0-9]+)[ \t]+(?P<thread_id>[0-9]+)(?:[ \t]*\r?\n[ \t]*|[ \t]+)(?P<random_number>[0-9]+);(?P<event_type>[^;\s]+);(?P<extra_tag1>[^;\s]+);(?P<extra_tag2>[^;\s]+);(?P<extra_tag3>[^;\s]+);(?P<event>[^;\s]+)(?:[ \t]+|$)`,
		Fields:  captures, ResourceLimits: fieldmapping.HeaderTemplateLimits{MaxInputBytes: 1048576, TimeoutMS: 50},
	}
	payload, err := json.Marshal(map[string]any{"config": header})
	if err != nil {
		t.Fatal(err)
	}
	runtime, diagnostics, err := baseline.WithRuntimeTemplates(ctx, []RuntimeTemplateAsset{{AssetClass: "templates", AssetID: "mail-header-v2", Revision: 1, Payload: payload}})
	if err != nil || len(diagnostics) > 0 {
		t.Fatalf("compile templates: %v %v", diagnostics, err)
	}
	record := templateHeaderPrefix + " status=failed;reason=not needed\n"
	result := runtime.TryParseV5Production(record, engine.Source{Service: "auto"})
	if !result.Success || result.Quarantined || result.ParseTelemetry.ParserID != "template/mail-header-v2" {
		t.Fatalf("template did not win: %+v", result.ParseTelemetry)
	}
	if result.EventPayload.Fields["extra_tag1"] != "efgh" || result.EventPayload.Fields["extra_tag2"] != "ijkl" || result.EventPayload.Fields["extra_tag3"] != "mnop" {
		t.Fatalf("header fields wrong: %+v", result.EventPayload.Fields)
	}
	if result.EventPayload.Fields["kv_pair"] != nil || result.EventPayload.Fields["status"] != "failed" || result.EventPayload.Fields["reason"] != "not needed" {
		t.Fatalf("KV tail wrong: %#v", result.EventPayload.Fields)
	}
	for _, lineBreak := range []string{"\n", "\r\n"} {
		wrapped := strings.Replace(record, "221568 123456", "221568"+lineBreak+"123456", 1)
		parsed := runtime.TryParseV5Production(wrapped, engine.Source{Service: "auto"})
		if !parsed.Success || parsed.ParseTelemetry.ParserID != "template/mail-header-v2" ||
			!reflect.DeepEqual(parsed.EventPayload.Fields, result.EventPayload.Fields) {
			t.Fatalf("wrapped header did not preserve all fields: %+v", parsed.ParseTelemetry)
		}
	}
	found := false
	for _, proof := range result.ParseTelemetry.FieldEvidence {
		if proof.Field == "reason" && record[proof.Start:proof.End] == "not needed" {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing exact KV evidence: %#v", result.ParseTelemetry.FieldEvidence)
	}
	for name, input := range map[string]string{
		"adjacent bracketed mixed assignments": "[status:failed][reason;not needed][server=cloud]\r\n",
		"quoted pair boundary":                 "status=failed;reason=\"not needed\"",
		"pipe framing":                         "|status=failed|reason=not needed|",
	} {
		t.Run(name, func(t *testing.T) {
			parsed := runtime.TryParseV5Production(templateHeaderPrefix+" "+input, engine.Source{Service: "auto"})
			if !parsed.Success || parsed.ParseTelemetry.ParserID != "template/mail-header-v2" || parsed.EventPayload.Fields["reason"] != "not needed" {
				t.Fatalf("unexpected parse: %+v", parsed)
			}
		})
	}
	pairs := make([]string, 500)
	for i := range pairs {
		pairs[i] = fmt.Sprintf("key%d=value%d", i, i)
	}
	large := runtime.TryParseV5Production(templateHeaderPrefix+" "+strings.Join(pairs, ";"), engine.Source{Service: "auto"})
	if !large.Success || len(large.EventPayload.Fields) != 500+len(captures) {
		t.Fatalf("a 500-pair global tail did not retain all fields: %+v", large)
	}
	headerOnly := runtime.TryParseV5Production(templateHeaderPrefix, engine.Source{Service: "auto"})
	if !headerOnly.Success || len(headerOnly.EventPayload.Fields) != len(captures) {
		t.Fatalf("header only: %+v", headerOnly)
	}
	malformed := runtime.TryParseV5Production(templateHeaderPrefix+" status=failed;reason", engine.Source{Service: "auto"})
	if malformed.Success || !malformed.Quarantined {
		t.Fatalf("partial suffix escaped quarantine: %+v fields=%+v", malformed.ParseTelemetry, malformed.EventPayload)
	}
	collision := runtime.TryParseV5Production(templateHeaderPrefix+" event=other;status=queued", engine.Source{Service: "auto"})
	if collision.Success || !collision.Quarantined || !strings.Contains(strings.Join(collision.ParseTelemetry.ErrorCodes, ","), "template_output_field_collision") {
		t.Fatalf("a KV field silently overwrote its header counterpart: %+v", collision)
	}
}

func TestExportedTemplatesPreserveRawKeysWithPublishedCatalog(t *testing.T) {
	baseline, err := NewParserPackAdapter()
	if err != nil {
		t.Fatal(err)
	}
	runtime, diagnostics, err := baseline.WithRuntimeTemplates(context.Background(), []RuntimeTemplateAsset{
		{AssetClass: "templates", AssetID: "extra_yag_3_ver1_id", Revision: 4, Payload: json.RawMessage(`{"config":{"engine":"re2","fields":{"pid":"pid","date":"date","event":"event","service":"service","hostname":"hostname","minor_pid":"minor_pid","thread_id":"thread_id","timestamp":"timestamp","event_type":"event_type","extra_tag1":"extra_tag1","extra_tag2":"extra_tag2","extra_tag3":"extra_tag3","random_number":"random_number"},"pattern":"^(?P<date>[0-9]{8})[ \\t]+(?P<timestamp>[0-9]{9}[+-][0-9]{4})[ \\t]+(?P<hostname>[^ \\t\\r\\n]+)[ \\t]+(?P<service>[^ \\t\\r\\n]+)[ \\t]+(?P<pid>[0-9]+)[ \\t]+(?P<minor_pid>[0-9]+)[ \\t]+(?P<thread_id>[0-9]+)(?:[ \\t]*\\r?\\n[ \\t]*|[ \\t]+)(?P<random_number>[0-9]+);(?P<event_type>(?i:trace|debug|info|note|notice|warn|warning|err|error|fatal|critical|alert|emerg|panic));(?P<extra_tag1>[^; \\t\\r\\n]+);(?P<extra_tag2>[^; \\t\\r\\n]+);(?P<extra_tag3>[^; \\t\\r\\n]+);(?P<event>[^;() \\t\\r\\n]+)(?:[ \\t]+|$)","optional_fields":[],"resource_limits":{"timeout_ms":50,"max_input_bytes":1048576}}}`)},
		{AssetClass: "kv-config", AssetID: "extra_tag3_kv_pair_id", Revision: 4, Payload: json.RawMessage(`{"config":{"max_fields":512,"pair_separators":[";"],"duplicate_policy":"error","literal_boundaries":true,"assignment_separators":["="]}}`)},
		{AssetClass: "kv-config", AssetID: "conflicting-profile", Revision: 1, Payload: json.RawMessage(`{"config":{"pair_separators":[";"],"assignment_separators":["="],"key_aliases":{"status":"state"},"literal_boundaries":true}}`)},
		{AssetClass: "known-keys", AssetID: "mail-keys", Revision: 4, Payload: json.RawMessage(`{"config":{"keys":["status","reason"]}}`)},
	})
	if err != nil || len(diagnostics) != 0 {
		t.Fatalf("compile exports: %v %v", diagnostics, err)
	}
	parsed := runtime.TryParseV5Production(templateHeaderPrefix+" status=failed;reason=not needed;vjay=vjay;service.name=edge;tenant-name=mx", engine.Source{Service: "auto"})
	if !parsed.Success || parsed.EventPayload == nil {
		t.Fatalf("combined parse: %+v", parsed.ParseTelemetry)
	}
	want := map[string]any{"status": "failed", "reason": "not needed", "vjay": "vjay", "service.name": "edge", "tenant-name": "mx"}
	for key, value := range want {
		if actual := parsed.EventPayload.Fields[key]; actual != value {
			t.Errorf("%s = %v, want %v", key, actual, value)
		}
	}
	if len(parsed.EventPayload.Fields) != 13+len(want) || parsed.EventPayload.Fields["extra_tag3"] != "mnop" {
		t.Fatalf("unexpected output: %#v", parsed.EventPayload.Fields)
	}
	encoded, err := json.Marshal(parsed.EventPayload.WithSourceOrder(parsed.ParseTelemetry.FieldEvidence))
	if err != nil || !strings.HasSuffix(string(encoded), `"event":"SieveRuleMsg","status":"failed","reason":"not needed","vjay":"vjay","service.name":"edge","tenant-name":"mx"}`) {
		t.Fatalf("flat output was not serialized in source order: %s, %v", encoded, err)
	}
}

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
		Pattern: `^(?P<date>[0-9]{8})[ \t]+(?P<timestamp>\S+)[ \t]+(?P<hostname>\S+)[ \t]+(?P<service>\S+)[ \t]+(?P<pid>[0-9]+)[ \t]+(?P<minor_pid>[0-9]+)[ \t]+(?P<thread_id>[0-9]+)[ \t]+(?P<random_number>[0-9]+);(?P<event_type>[^;\s]+);(?P<extra_tag1>[^;\s]+);(?P<extra_tag2>[^;\s]+);(?P<extra_tag3>[^;\s]+);(?P<event>[^;\s]+)(?:[ \t]+|$)`,
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
	if !reflect.DeepEqual(result.EventPayload.Fields["kv_pair"], map[string]any{"status": "failed", "reason": "not needed"}) {
		t.Fatalf("KV tail wrong: %#v", result.EventPayload.Fields["kv_pair"])
	}
	found := false
	for _, proof := range result.ParseTelemetry.FieldEvidence {
		if proof.Field == "kv_pair.reason" && record[proof.Start:proof.End] == "not needed" {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing exact nested evidence: %#v", result.ParseTelemetry.FieldEvidence)
	}
	for name, input := range map[string]string{
		"adjacent bracketed mixed assignments": "[status:failed][reason;not needed][server=cloud]\r\n",
		"quoted pair boundary":                 "status=failed;reason=\"not needed\"",
		"pipe framing":                         "|status=failed|reason=not needed|",
	} {
		t.Run(name, func(t *testing.T) {
			parsed := runtime.TryParseV5Production(templateHeaderPrefix+" "+input, engine.Source{Service: "auto"})
			if !parsed.Success || parsed.ParseTelemetry.ParserID != "template/mail-header-v2" || parsed.EventPayload.Fields["kv_pair"].(map[string]any)["reason"] != "not needed" {
				t.Fatalf("unexpected parse: %+v", parsed)
			}
		})
	}
	pairs := make([]string, 500)
	for i := range pairs {
		pairs[i] = fmt.Sprintf("key%d=value%d", i, i)
	}
	large := runtime.TryParseV5Production(templateHeaderPrefix+" "+strings.Join(pairs, ";"), engine.Source{Service: "auto"})
	if !large.Success || len(large.EventPayload.Fields["kv_pair"].(map[string]any)) != 500 {
		t.Fatalf("a 500-pair global tail did not retain all fields: %+v", large)
	}
	headerOnly := runtime.TryParseV5Production(templateHeaderPrefix, engine.Source{Service: "auto"})
	if !headerOnly.Success || headerOnly.EventPayload.Fields["kv_pair"] != nil {
		t.Fatalf("header only: %+v", headerOnly)
	}
	malformed := runtime.TryParseV5Production(templateHeaderPrefix+" status=failed;reason", engine.Source{Service: "auto"})
	if malformed.Success || !malformed.Quarantined {
		t.Fatalf("partial suffix escaped quarantine: %+v fields=%+v", malformed.ParseTelemetry, malformed.EventPayload)
	}
}

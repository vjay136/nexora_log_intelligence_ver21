package adapter

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	"nexora.local/nli-v20/internal/engine"
	"nexora.local/nli-v20/internal/fieldmapping"
	parserpolicy "nexora.local/nli-v20/parser-policy"
)

const additiveSMTP = "20250812 130034198+0100 mail001 smtp 3609 0 140290550523648 Note;SmtpServerConnect(2001/3) id=6864BD5E047384B0:host=11.22.33.44:connection_count=1:local_port=2525"

func TestSMTPWrappedTailCannotNormalizePartially(t *testing.T) {
	const header = "20250812 130035121+0100 aaa-gmailrgo009 smtp 3609 0 140293780948736 Note;SmtpServerEHLO(2001/4) "
	const tail = "{id=6864BD5E047384C4:domain=server035.mx.internal:host=11.22.33.44}"
	adapter, err := NewParserPackAdapter()
	if err != nil {
		t.Fatal(err)
	}
	source := engine.Source{Service: "auto"}
	result := adapter.TryParseV5Production(header+tail, source)
	if !result.Success || result.Quarantined {
		t.Fatalf("complete sample was not parsed: %+v", result.ParseTelemetry)
	}
	t.Logf("sample selected parser %s", result.ParseTelemetry.ParserID)
	for key, want := range map[string]string{"id": "6864BD5E047384C4", "domain": "server035.mx.internal", "host": "11.22.33.44"} {
		if got, ok := result.EventPayload.Get(key); !ok || got != want {
			t.Errorf("%s = %v, want %q", key, got, want)
		}
	}
	for _, input := range []string{header + strings.TrimSuffix(tail, "}"), header + "{id=6864BD5E047384C4:unmapped_field=lost}"} {
		rejected := adapter.TryParseV5Production(input, source)
		if rejected.Success || !rejected.Quarantined || !slices.Contains(rejected.ParseTelemetry.ErrorCodes, "incomplete_structured_tail") {
			t.Fatalf("partial result escaped quarantine: %+v", rejected.ParseTelemetry)
		}
	}
	misrouted := adapter.TryParseV5Production(header+tail, engine.Source{Service: "unrecognized-mail-service"})
	if misrouted.Success || !misrouted.Quarantined || !slices.Contains(misrouted.ParseTelemetry.ErrorCodes, "incomplete_structured_tail") {
		t.Fatalf("unsupported configured service allowed generic fallback: %+v", misrouted.ParseTelemetry)
	}
}

func TestSMTPPipeTailCannotNormalizePartially(t *testing.T) {
	const header = "20250812 130035121+0100 aaa-gmailrgo009 smtp 3609 0 140293780948736 Note;SmtpServerEHLO(2001/4) "
	adapter, err := NewParserPackAdapter()
	if err != nil {
		t.Fatal(err)
	}
	deps := engine.Default()
	deps.SetV5Parser(adapter)
	for _, tail := range []string{
		`|sid=5f9e|tenant=acme corp|from=sales@example.com|to=buyer@example.org|subj==?UTF-8?B?w4RuZGVydW5nIGZ1ciBBbmdlYm90?=|sz=5120|enc=quoted-printable|body=Hello team please reviewbudget`,
		`|RCPT|sid=5f9e\|2c1a|tenant=acme corp|from=sales@example.com|to=buyer@example.org|subj==?UTF-8?B?w4RuZGVydW5nIGZ1ciBBbmdlYm90?=|sz=5120|enc=quoted-printable|body=Hello=20team,=\nplease review =E2=82=AC500 budget=\r\n`,
		`|DATA|sid=5f9e\|2c1a|dkim=fail(sig-expired)|spam=0.12|hdr=X-Mailer:Outlook 16.0\, Build 1234|hdr=X-Forwarded-For:198.51.100.91, 203.0.113.9`,
		`|GW-EU-07|DEFER|sid=5f9e\|2c1a|reason=4.7.1 greylisted; retry after 300s|`,
	} {
		for _, service := range []string{"auto", "unrecognized-mail-service"} {
			result := adapter.TryParseV5Production(header+tail, engine.Source{Service: service})
			if result.Success || !result.Quarantined || !slices.Contains(result.ParseTelemetry.ErrorCodes, "incomplete_structured_tail") {
				t.Fatalf("partial pipe payload escaped quarantine for %s: %+v", service, result.ParseTelemetry)
			}
			routed := engine.Parse(header+tail, engine.Source{Service: service}, deps)
			if string(routed.Meta.Route) != "quarantine" {
				t.Fatalf("partial pipe payload entered normalized output: %+v", routed)
			}
		}
	}
}

func TestSMTPReviewedPipeMappingRoutesCompletePayload(t *testing.T) {
	const raw = "20250812 130035121+0100 aaa-gmailrgo009 smtp 3609 0 140293780948736 Note;SmtpServerEHLO(2001/4) " +
		`|DATA|sid=5f9e\|2c1a|hdr=X-Mailer:Outlook 16.0\, Build 1234|hdr=X-Forwarded-For:198.51.100.91, 203.0.113.9`
	fields := []fieldmapping.Field{
		{Name: "smtp_command", Type: "string", Source: "field1"},
		{Name: "session_id", Type: "string", Source: "sid"},
		{Name: "headers", Type: "array", Source: "hdr", DuplicatePolicy: "array"},
	}
	for i := range fields {
		fields[i].Method, fields[i].KVProfile, fields[i].KVStart = "key_value", "auto", "9"
	}
	adapter, want := sourceParityAdapter(t, raw, fieldmapping.Plan{Version: 1, Fields: fields})
	deps := engine.Default()
	deps.SetV5Parser(adapter)
	result := engine.Parse(raw, engine.Source{Service: "nfi-fixture"}, deps)
	if string(result.Meta.Route) != "normal" || !reflect.DeepEqual(result.Data, want) {
		t.Fatalf("complete reviewed pipe mapping did not normalize: %+v", result)
	}
	if result.Data["smtp_command"] != "DATA" || result.Data["session_id"] != "5f9e|2c1a" || !reflect.DeepEqual(result.Data["headers"], []any{
		`X-Mailer:Outlook 16.0\, Build 1234`,
		"X-Forwarded-For:198.51.100.91, 203.0.113.9",
	}) {
		t.Fatalf("pipe payload lost evidence: %#v", result.Data)
	}
}

func TestAutomaticReviewedMappingDoesNotShortCircuitExistingParsers(t *testing.T) {
	adapter, _ := sourceParityAdapter(t, additiveSMTP, fieldmapping.Plan{Version: 1, Fields: []fieldmapping.Field{{Name: "custom_message", Method: "raw"}}})
	for index := range adapter.profiles {
		profile := &adapter.profiles[index]
		if !profile.Managed {
			continue
		}
		profile.Pack.Rules[0].ServiceSelectors = []string{"smtp-in"}
		profile.Pack.RegressionCases[0].Service = "smtp-in"
		if err := parserpolicy.SealParserPack(profile.Pack); err != nil {
			t.Fatal(err)
		}
		executor, err := parserpolicy.NewParserPackExecutor(profile.Pack)
		if err != nil {
			t.Fatal(err)
		}
		profile.Executor = executor
	}
	result := adapter.TryParseV5Production(additiveSMTP, engine.Source{Service: "smtp-in"})
	if !result.Success || !result.ReviewedMapping {
		t.Fatalf("reviewed mapping lost its approved output choice: %+v", result)
	}
	attempted := map[string]bool{}
	for _, attempt := range result.ParseTelemetry.OrderedAttempts {
		attempted[attempt.ParserID] = true
	}
	for _, id := range []string{"smtp_native_v5", "nli_reviewed_smtp_in/reviewed_smtp_in", "nli_reviewed_smtp_out", "nfi-source-parity/mapping"} {
		found := attempted[id]
		for parser := range attempted {
			if strings.HasPrefix(parser, id+"/") {
				found = true
			}
		}
		if !found {
			t.Fatalf("existing/add-on parser omitted from automatic pool: %s: %+v", id, attempted)
		}
	}
	if result.EventPayload.FieldCount() != 1 {
		t.Fatal("candidate evaluation emitted multiple customer results")
	}
}

func TestAutomaticWildcardAddOnRunsAlongsideSuccessfulExistingParser(t *testing.T) {
	directory := t.TempDir()
	writeWildcardRuntimePack(t, directory, "add-on-wildcard")
	adapter, err := NewParserPackAdapterFromDirectory(directory)
	if err != nil {
		t.Fatal(err)
	}
	result := adapter.TryParseV5Production(additiveSMTP, engine.Source{Service: "smtp-in"})
	if !result.Success || result.ParseTelemetry.ParserID != "nli_reviewed_smtp_in/reviewed_smtp_in" {
		t.Fatalf("add-on changed existing successful format: %+v", result)
	}
	found := false
	for _, attempt := range result.ParseTelemetry.OrderedAttempts {
		if strings.HasPrefix(attempt.ParserID, "add-on-wildcard/") {
			found = true
		}
	}
	if !found {
		t.Fatal("wildcard add-on was skipped after embedded success")
	}
	if len(result.ParseTelemetry.ErrorCodes) != 0 {
		t.Fatal("failed add-on contaminated successful embedded parser health")
	}
}

func TestAutomaticAddOnCannotSuppressExistingJSONFallback(t *testing.T) {
	directory := t.TempDir()
	writeWildcardRuntimePack(t, directory, "add-on-wildcard")
	adapter, err := NewParserPackAdapterFromDirectory(directory)
	if err != nil {
		t.Fatal(err)
	}
	deps := engine.Default()
	deps.SetV5Parser(adapter)
	for _, message := range []string{"unchanged", "status=ok"} {
		raw := `{"message":"` + message + `","original":true}`
		result := engine.Parse(raw, engine.Source{Service: "auto"}, deps)
		if result.Data["message"] != message || result.Data["original"] != true || string(result.Meta.Route) != "normal" {
			t.Fatalf("add-on suppressed structural JSON: %+v", result)
		}
		if result.Meta.ParserAttempts < 1 {
			t.Fatal("add-on not considered before preserved JSON fallback")
		}
	}
}

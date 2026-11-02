package controlplane

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	metastore "nexora.local/nli-v20/internal/store"
	parserpolicy "nexora.local/nli-v20/parser-policy"
)

func TestParserCatalogRejectsNonRuntimeDraftsBeforeReview(t *testing.T) {
	fixture := newAPIFixture(t)
	definition := map[string]any{
		"pattern_type": "regex", "pattern": `^status=(?P<status>\w+)$`,
		"fields": []any{map[string]any{"name": "status", "source": "group:status", "type": "string"}},
	}
	checksum := parserArtifactChecksum(definition)

	migrationPack, err := assuranceRuntimePack("migration-without-evidence", "1.0.0", "migration", definition, []assuranceRequest{{
		Raw: "status=OK", Desired: map[string]any{"status": "OK"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	migrationPack.RegressionCases = nil
	migrationPack.ReviewState = parserpolicy.ReviewStateDraft
	migrationPack.ActivatedAt = nil
	migrationPack.SHA256 = ""
	if err := parserpolicy.SealParserPack(migrationPack); err != nil {
		t.Fatal(err)
	}
	migrationPayload, err := json.Marshal(migrationPack)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		id      string
		payload any
		want    string
	}{
		{
			id: "napi-without-retained-evidence",
			payload: map[string]any{
				"id": "napi-without-retained-evidence", "parser": definition,
				"validation_passed": true, "artifact_checksum": checksum,
				"generated_by": "napi-bounded-v1",
			},
			want: "no retained reviewed regression cases",
		},
		{
			id: "promoted-provenance-only",
			payload: map[string]any{
				"id": "promoted-provenance-only", "parser": definition,
				"validation_passed": true, "artifact_checksum": checksum,
				"regression_cases": []any{map[string]any{"raw_sha256": "evidence-only", "desired": map[string]any{"status": "OK"}}},
			},
			want: "no retained reviewed regression cases",
		},
		{id: "migration-without-evidence", payload: json.RawMessage(migrationPayload), want: "reviewed regression"},
	}

	for _, test := range tests {
		t.Run(test.id, func(t *testing.T) {
			payload, marshalErr := json.Marshal(test.payload)
			if raw, ok := test.payload.(json.RawMessage); ok {
				payload = append([]byte(nil), raw...)
				marshalErr = nil
			}
			if marshalErr != nil {
				t.Fatal(marshalErr)
			}
			if _, putErr := fixture.server.metadata.Put(context.Background(), metastore.Document{
				Collection: "parser-packs", ID: test.id, State: "draft", Payload: payload, CreatedBy: "admin",
			}); putErr != nil {
				t.Fatal(putErr)
			}

			response, body := fixture.request(http.MethodPost, "/api/v1/parser-packs/"+test.id+"/submit", map[string]any{})
			if response.StatusCode != http.StatusConflict || !strings.Contains(string(body), test.want) {
				t.Fatalf("unsafe draft advanced to review: status=%d body=%s", response.StatusCode, body)
			}
			stored, getErr := fixture.server.metadata.Get(context.Background(), "parser-packs", test.id, 0)
			if getErr != nil {
				t.Fatal(getErr)
			}
			if stored.State != "draft" {
				t.Fatalf("rejected parser state=%q, want draft", stored.State)
			}

			// Historical databases can already contain an in-review record from an
			// older release. Approval must apply the same executable-evidence gate
			// instead of allowing that record to bypass the new submit check.
			if _, putErr := fixture.server.metadata.Put(context.Background(), metastore.Document{
				Collection: "parser-packs", ID: test.id, State: "in_review", Payload: payload, CreatedBy: "legacy-author",
			}); putErr != nil {
				t.Fatal(putErr)
			}
			response, body = fixture.request(http.MethodPost, "/api/v1/parser-packs/"+test.id+"/approve", map[string]any{})
			if response.StatusCode != http.StatusConflict || !strings.Contains(string(body), test.want) {
				t.Fatalf("unsafe historical review was approved: status=%d body=%s", response.StatusCode, body)
			}
			stored, getErr = fixture.server.metadata.Get(context.Background(), "parser-packs", test.id, 0)
			if getErr != nil {
				t.Fatal(getErr)
			}
			if stored.State != "in_review" {
				t.Fatalf("rejected historical review state=%q, want in_review", stored.State)
			}
		})
	}
}

func TestRawCustomParserCompletesCatalogReviewAndPublication(t *testing.T) {
	fixture := newAPIFixture(t)
	const id = "raw-custom-lifecycle"
	if _, err := fixture.server.metadata.Put(context.Background(), metastore.Document{
		Collection: "parser-packs", ID: id, State: "draft", Payload: legacyParserPayload(t, id), CreatedBy: "admin",
	}); err != nil {
		t.Fatal(err)
	}

	response, body := fixture.request(http.MethodPost, "/api/v1/parser-packs/"+id+"/submit", map[string]any{})
	if response.StatusCode != http.StatusOK {
		t.Fatalf("submit custom parser: %d %s", response.StatusCode, body)
	}
	awaiting, err := fixture.server.metadata.Get(context.Background(), "parser-packs", id, 0)
	if err != nil {
		t.Fatal(err)
	}
	response, body = fixture.request(http.MethodPost, "/api/v1/parser-packs/"+id+"/approve", map[string]any{
		"emergency_self_approval": true,
		"reason":                  "Administrator is approving this exact reviewed parser for quarantine recovery.",
		"expected_revision":       awaiting.Revision,
		"expected_checksum":       awaiting.Hash,
	})
	if response.StatusCode != http.StatusOK {
		t.Fatalf("approve exact custom parser: %d %s", response.StatusCode, body)
	}
	response, body = fixture.request(http.MethodPost, "/api/v1/parser-packs/"+id+"/activate", map[string]any{})
	if response.StatusCode != http.StatusOK {
		t.Fatalf("publish custom parser: %d %s", response.StatusCode, body)
	}
	published, err := fixture.server.metadata.Get(context.Background(), "parser-packs", id, 0)
	if err != nil {
		t.Fatal(err)
	}
	if published.State != "published" {
		t.Fatalf("published state=%q", published.State)
	}
	if _, _, err := runtimeParserPackForDocument(published); err != nil {
		t.Fatalf("published custom parser is not executable: %v", err)
	}
}

func TestQuarantineCustomParserCompletesAssurancePublication(t *testing.T) {
	fixture := newAPIFixture(t)
	if err := fixture.server.storeQuarantineRecord(context.Background(), quarantineRecordInput{
		ID: "q-lifecycle", NodeID: "node-lifecycle", Status: "waiting",
	}, false); err != nil {
		t.Fatal(err)
	}
	definition := map[string]any{
		"pattern_type": "regex", "pattern": `^status=(?P<status>\w+)$`,
		"fields": []any{map[string]any{"name": "status", "source": "group:status", "type": "string", "required": true}},
	}
	response, body := fixture.request(http.MethodPost, "/api/v1/assurance/parsers", map[string]any{
		"id": "quarantine-custom-lifecycle", "name": "Quarantine custom lifecycle", "version": "1.0.0",
		"parser":                      definition,
		"cases":                       []any{map[string]any{"raw": "status=OK", "desired": map[string]any{"status": "OK"}}},
		"source_quarantine_record_id": "q-lifecycle",
	})
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("create quarantine parser: %d %s", response.StatusCode, body)
	}
	response, body = fixture.request(http.MethodPost, "/api/v1/assurance/parsers/quarantine-custom-lifecycle/submit", map[string]any{})
	if response.StatusCode != http.StatusOK {
		t.Fatalf("submit quarantine parser: %d %s", response.StatusCode, body)
	}
	doc, err := fixture.server.metadata.Get(context.Background(), "parser-packs", "quarantine-custom-lifecycle", 0)
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]any
	if err := json.Unmarshal(doc.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	response, body = fixture.request(http.MethodPost, "/api/v1/assurance/parsers/quarantine-custom-lifecycle/approve", map[string]any{
		"emergency_self_approval": true,
		"reason":                  "Administrator is approving this exact validated quarantine parser.",
		"expected_revision":       doc.Revision,
		"expected_checksum":       payload["artifact_checksum"],
	})
	if response.StatusCode != http.StatusOK {
		t.Fatalf("approve quarantine parser: %d %s", response.StatusCode, body)
	}
	response, body = fixture.request(http.MethodPost, "/api/v1/assurance/parsers/quarantine-custom-lifecycle/publish", map[string]any{})
	if response.StatusCode != http.StatusOK {
		t.Fatalf("publish quarantine parser: %d %s", response.StatusCode, body)
	}
	published, err := fixture.server.metadata.Get(context.Background(), "parser-packs", "quarantine-custom-lifecycle", 0)
	if err != nil {
		t.Fatal(err)
	}
	if published.State != "published" {
		t.Fatalf("published state=%q", published.State)
	}
	if _, _, err := runtimeParserPackForDocument(published); err != nil {
		t.Fatalf("published quarantine parser is not executable: %v", err)
	}
}

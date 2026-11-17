package controlplane

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	metastore "nexora.local/nli-v20/internal/store"
)

func TestCatalogRevisionKeepsPublishedBundleUntilPublication(t *testing.T) {
	f := newAPIFixture(t)
	ctx := context.Background()
	const id = "catalog-revision-header"
	const route = "/api/v1/config/templates/" + id
	published, err := f.server.metadata.Put(ctx, metastore.Document{
		Collection: "templates", ID: id, State: "active", CreatedBy: "admin",
		Payload: json.RawMessage(`{"name":"Published header","config":{"pattern":"^(?P<extra_tag1>[a-z]+)(?: |$)","fields":{"extra_tag1":"extra_tag1"}}}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	current := func() metastore.Document {
		t.Helper()
		doc, err := f.server.metadata.Get(ctx, "templates", id, 0)
		if err != nil {
			t.Fatal(err)
		}
		return doc
	}
	bundleHash := func() string {
		t.Helper()
		bundle, err := f.server.buildActiveParserBundle(ctx)
		if err != nil {
			t.Fatal(err)
		}
		return bundle.BundleHash
	}
	firstBundle := bundleHash()
	edit := func(revision, wantStatus int) {
		t.Helper()
		response, data := f.request(http.MethodPut, route, map[string]any{
			"name": "Revised header", "expected_revision": revision,
			"config": map[string]any{"pattern": `^(?P<extra_tag1>[a-z]+)(?: |$)`, "fields": map[string]string{"extra_tag1": "tenant_code"}},
		})
		if response.StatusCode != wantStatus {
			t.Fatalf("revision edit: %d %s", response.StatusCode, data)
		}
	}
	edit(published.Revision, http.StatusOK)
	edit(published.Revision, http.StatusConflict)
	if got := bundleHash(); got != firstBundle {
		t.Fatal("draft replaced the published bundle")
	}
	response, data := f.request(http.MethodDelete, route, map[string]any{})
	if response.StatusCode != http.StatusOK || bundleHash() != firstBundle || current().State != "discarded" {
		t.Fatalf("discard did not retain the published bundle: %d %s", response.StatusCode, data)
	}
	edit(current().Revision, http.StatusOK)
	for _, action := range []string{"submit", "approve", "publish"} {
		response, data := f.request(http.MethodPost, route+"/"+action, map[string]any{})
		if response.StatusCode != http.StatusOK {
			t.Fatalf("%s: %d %s", action, response.StatusCode, data)
		}
		if action != "publish" && bundleHash() != firstBundle {
			t.Fatalf("%s replaced the published bundle before publication", action)
		}
	}
	if bundleHash() == firstBundle || current().State != "active" {
		t.Fatal("approved field rename did not become a live publication")
	}
}

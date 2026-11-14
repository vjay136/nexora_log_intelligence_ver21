package controlplane

import (
	"net/http"
	"path/filepath"
	"testing"
	"time"
)

func TestRolloutsAndParserSyncRecoverFromMetadataFailure(t *testing.T) {
	for _, tc := range []struct {
		name, path, failingCollection string
	}{
		{"rollout catalog", "/api/v1/rollouts", "rollout-proposals"},
		{"fleet reconciliation", "/api/v1/fleet/sync", "fleet-sync"},
		{"managed parser catalog", "/api/v1/parser-packs", "parser-packs"},
		{"parser catalog", "/api/v1/parser-sync", "parser-packs"},
		{"client acknowledgements", "/api/v1/parser-sync", "fleet-sync"},
		{"parser rollout approvals", "/api/v1/parser-sync", "rollout-proposals"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newAPIFixture(t)
			store := f.server.metadata
			f.server.metadata = navigationFailingStore{Store: store, failCollection: tc.failingCollection}
			response, _ := f.request(http.MethodGet, tc.path, nil)
			if response.StatusCode != http.StatusServiceUnavailable {
				t.Fatalf("cold failure status = %d", response.StatusCode)
			}

			f.server.metadata = store
			response, body := f.request(http.MethodGet, tc.path, nil)
			fresh := decodeMap(t, body)
			if response.StatusCode != http.StatusOK || fresh["snapshot_stale"] != false || fresh["snapshot_at"] == nil {
				t.Fatalf("fresh status %d: %s", response.StatusCode, body)
			}

			f.server.metadata = navigationFailingStore{Store: store, failCollection: tc.failingCollection}
			response, body = f.request(http.MethodGet, tc.path, nil)
			stale := decodeMap(t, body)
			if response.StatusCode != http.StatusOK || stale["snapshot_stale"] != true || stale["snapshot_at"] != fresh["snapshot_at"] || stale["snapshot_error"] == nil {
				t.Fatalf("fallback status %d: %s", response.StatusCode, body)
			}
			if response.Header.Get("Cache-Control") != "no-store" {
				t.Fatal("stale catalog must never be cached by intermediaries")
			}
			view := &f.server.parserSyncView
			if tc.path == "/api/v1/rollouts" {
				view = &f.server.rolloutCatalogView
				response, _ = f.request(http.MethodGet, tc.path+"?state=active", nil)
				if response.StatusCode != http.StatusServiceUnavailable {
					t.Fatalf("filtered query used the unfiltered snapshot: %d", response.StatusCode)
				}
			}
			if tc.path == "/api/v1/parser-packs" {
				view = &f.server.parserCatalogView
			}
			if tc.path == "/api/v1/fleet/sync" {
				view = &f.server.fleetSyncView
			}
			view.mu.Lock()
			view.at = time.Now().Add(-metadataViewMaxAge - time.Second)
			view.mu.Unlock()
			response, _ = f.request(http.MethodGet, tc.path, nil)
			if response.StatusCode != http.StatusServiceUnavailable {
				t.Fatalf("expired fallback status = %d", response.StatusCode)
			}

			f.server.metadata = store
			response, body = f.request(http.MethodGet, tc.path, nil)
			if response.StatusCode != http.StatusOK || decodeMap(t, body)["snapshot_stale"] != false {
				t.Fatalf("recovered status %d: %s", response.StatusCode, body)
			}
		})
	}
}

func TestRolloutReadViewsAreDetachedFromNodeResults(t *testing.T) {
	manager := newRolloutStateManager(filepath.Join(t.TempDir(), "rollouts.json"))
	created, err := manager.Create(rolloutTypeRestart, "restart client", "admin", []string{"client"}, map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.Start(created.ID, []string{"client"}); err != nil {
		t.Fatal(err)
	}
	getView, found := manager.Get(created.ID)
	listView := manager.List()
	if !found || len(listView) != 1 {
		t.Fatal("new rollout missing")
	}
	if err := manager.RecordNodeResult(created.ID, "client", rolloutStatusCompleted, "restarted"); err != nil {
		t.Fatal(err)
	}
	for _, previous := range []*rolloutRecord{getView, listView[0]} {
		if previous.NodeResults["client"].Status != rolloutStatusPending {
			t.Fatal("prior read retained the live mutable result")
		}
	}
	current, _ := manager.Get(created.ID)
	if current.NodeResults["client"].Status != rolloutStatusCompleted {
		t.Fatal("current read did not observe completion")
	}
}

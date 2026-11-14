package controlplane

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"nexora.local/nli-v20/internal/familyprofile"
	"nexora.local/nli-v20/internal/middleware"
	"nexora.local/nli-v20/internal/protocol"
	metastore "nexora.local/nli-v20/internal/store"
)

// deleteParserDocument is the single deletion boundary for both catalog APIs.
// The retained metadata archive is supported by local and remote stores alike.
// Caller holds parserMutationMu, so no stale publish can race the removal.
func (s *Server) deleteParserDocument(w http.ResponseWriter, r *http.Request, p *middleware.SecurityContext, doc metastore.Document) {
	if familyprofile.IsBuiltinPackID(doc.ID) {
		jsonError(w, "embedded baseline parsers cannot be deleted", 409)
		return
	}
	active := doc.State == "published" || doc.State == "active"
	removed := []metastore.Document{doc}
	if active {
		if !principalHasPermission(p, middleware.PermissionTriggerRollout) {
			jsonError(w, "active parser deletion requires rollout permission", 403)
			return
		}
		if pack, _, err := runtimeParserPackForDocumentContext(r.Context(), doc); err == nil {
			if familyprofile.IsBuiltinPackID(pack.ID) {
				jsonError(w, "embedded baseline parsers cannot be deleted", 409)
				return
			}
			// Legacy import aliases refer to a single runtime pack. Remove the entire
			// active identity rather than letting an alias silently resurrect it.
			peers, err := s.listParserSyncDocuments(r.Context(), "parser-packs", "", parserSyncTargetLimit)
			if err != nil {
				jsonError(w, "parser identity inventory unavailable", 503)
				return
			}
			for _, peer := range peers {
				if peer.ID == doc.ID || (peer.State != "published" && peer.State != "active") {
					continue
				}
				candidate, _, err := runtimeParserPackForDocumentContext(r.Context(), peer)
				if err == nil && candidate.ID == pack.ID {
					removed = append(removed, peer)
				}
			}
		}
	}
	deleted := []metastore.Document{}
	compensate := func() error {
		if len(deleted) == 0 {
			return nil
		}
		// Request cancellation must never cancel restoration of a mutation
		// already committed by a remote metadata RPC.
		rollbackCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		var failures []error
		for _, old := range deleted {
			if _, err := s.metadata.Put(rollbackCtx, metastore.Document{Collection: "parser-packs", ID: old.ID, State: old.State, Payload: old.Payload, CreatedBy: old.CreatedBy}); err != nil {
				failures = append(failures, fmt.Errorf("restore %s: %w", old.ID, err))
			}
		}
		if previous, err := s.buildActiveParserBundle(rollbackCtx); err != nil {
			failures = append(failures, err)
		} else if _, err := s.materializeParserBundle(previous); err != nil {
			failures = append(failures, err)
		}
		return errors.Join(failures...)
	}
	fail := func(message string) {
		if err := compensate(); err != nil {
			s.auditAction(p, "parser.delete.rollback", "parser-packs/"+doc.ID, "failure", map[string]string{"rollback_error": err.Error(), "archive_id": doc.ID})
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(map[string]any{"error": message + "; automatic restoration did not complete", "code": "parser_delete_rollback_failed", "archive_id": doc.ID, "rollback_failed": true, "next_action": "Inspect the retained parser archive and controller/client bundle status before retrying deletion or restoring a reviewed draft."})
			return
		}
		jsonError(w, message, http.StatusServiceUnavailable)
	}
	ids := []string{}
	for _, old := range removed {
		archive, _ := json.Marshal(map[string]any{"original_id": old.ID, "original_revision": old.Revision, "original_state": old.State, "original_created_by": old.CreatedBy, "name": parserArchiveName(old), "definition": json.RawMessage(old.Payload), "deleted_at": time.Now().UTC(), "deleted_by": p.Username, "restorable": true})
		if _, err := s.metadata.Put(r.Context(), metastore.Document{Collection: "parser-trash", ID: old.ID, State: "retained", Payload: archive, CreatedBy: p.Username}); err != nil {
			fail("parser archive failed; deletion cancelled")
			return
		}
		// Include an attempted deletion: a remote RPC can commit its tombstone
		// and then lose the response when the caller disconnects.
		deleted = append(deleted, old)
		if err := s.metadata.Delete(r.Context(), "parser-packs", old.ID, false, p.Username); err != nil {
			fail("parser deletion failed")
			return
		}
		ids = append(ids, old.ID)
	}
	result := map[string]any{"status": "deleted", "id": doc.ID, "archive_id": doc.ID, "removed_catalog_ids": ids, "restorable": true, "client_sync_status": "not_required"}
	if active {
		bundle, err := s.buildActiveParserBundle(r.Context())
		if err == nil {
			_, err = s.materializeParserBundle(bundle)
		}
		var proposal metastore.Document
		if err == nil {
			proposal, err = s.proposeParserBundleRollout(r.Context(), p.Username, "Remove parser "+doc.ID, "remove deleted parser runtime identity", "all_eligible", bundle, nil)
		}
		if err != nil {
			fail("parser removal rollout failed; verify controller and client status before retrying")
			return
		}
		result["bundle_hash"] = bundle.BundleHash
		result["rollout_proposal"] = documentView(proposal)
		result["approval_gate"] = "pending_approval"
		result["client_sync_status"] = "pending_approval"
	}
	s.auditAction(p, "parser.delete", "parser-packs/"+doc.ID, "success", map[string]string{"removed_catalog_ids": strings.Join(ids, ","), "retained": "true"})
	jsonOK(w, result)
}

func (s *Server) handleDeletedParsers(w http.ResponseWriter, r *http.Request) {
	permission := middleware.PermissionReadParsers
	if r.Method != http.MethodGet {
		permission = middleware.PermissionManageParsers
	}
	p := s.authorize(w, r, permission)
	if p == nil {
		return
	}
	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/parser-packs/deleted"), "/")
	if path == "" && r.Method == http.MethodGet {
		docs, err := s.listParserSyncDocuments(r.Context(), "parser-trash", "retained", parserSyncTargetLimit)
		if err != nil {
			jsonError(w, "deleted parser history unavailable", 503)
			return
		}
		views := []map[string]any{}
		for _, doc := range docs {
			if _, err := s.metadata.Get(r.Context(), "parser-packs", doc.ID, 0); err == nil {
				continue
			}
			var body map[string]any
			if json.Unmarshal(doc.Payload, &body) != nil {
				continue
			}
			delete(body, "definition")
			body["id"] = doc.ID
			body["revision"] = doc.Revision
			views = append(views, body)
		}
		jsonOK(w, map[string]any{"deleted_parsers": views, "count": len(views)})
		return
	}
	parts := strings.Split(path, "/")
	if r.Method != http.MethodPost || len(parts) != 2 || parts[1] != "restore" {
		jsonError(w, "use GET deleted or POST deleted/:id/restore", 405)
		return
	}
	s.parserMutationMu.Lock()
	defer s.parserMutationMu.Unlock()
	archive, err := s.metadata.Get(r.Context(), "parser-trash", parts[0], 0)
	if err != nil {
		jsonError(w, "retained parser not found", 404)
		return
	}
	var old struct {
		Definition       map[string]any `json:"definition"`
		OriginalRevision int            `json:"original_revision"`
	}
	if json.Unmarshal(archive.Payload, &old) != nil || old.Definition == nil {
		jsonError(w, "invalid retained parser", 409)
		return
	}
	var input struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		Version string `json:"version"`
		Reason  string `json:"reason"`
	}
	if err := decodeJSON(r, &input, 16<<10); err != nil || len(input.Name) > 256 || len(input.Reason) > 512 {
		jsonError(w, "restore request required", 400)
		return
	}
	if input.ID == "" {
		input.ID = fmt.Sprintf("%.40s-restored-%d", parts[0], time.Now().UnixNano())
	}
	if input.ID != safeID(input.ID) || input.ID == parts[0] || familyprofile.IsBuiltinPackID(input.ID) {
		jsonError(w, "restore requires a new safe parser ID", 400)
		return
	}
	if _, err := s.metadata.Get(r.Context(), "parser-packs", input.ID, 0); err == nil {
		jsonError(w, "parser ID already exists", 409)
		return
	} else if !errors.Is(err, os.ErrNotExist) {
		jsonError(w, "parser uniqueness cannot be verified", 503)
		return
	}
	body := old.Definition
	if _, canonical := body["parser"].(map[string]any); !canonical {
		if runtime, ok := body["runtime_pack"].(map[string]any); ok {
			for _, key := range []string{"rules", "regression_cases", "version", "resource_limits"} {
				if value, ok := runtime[key]; ok {
					body[key] = value
				}
			}
		}
	}
	for _, key := range []string{"runtime_pack", "approved_by", "approved_at", "approved_revision", "approved_checksum", "published_by", "published_at", "approval_mode", "approval_reason", "emergency_approval_reason", "emergency_self_approval", "emergency_approved_at", "emergency_approved_by", "emergency_approved_revision", "validation", "sha256", "catalog_id", "runtime_id", "submitted_checksum", "approved_bundle_hash", "reviewed_by"} {
		delete(body, key)
	}
	body["id"] = input.ID
	body["review_state"] = "draft"
	body["activated"] = false
	body["validation"] = map[string]any{"passed": false}
	body["validation_passed"] = false
	body["restored_from"] = map[string]any{"id": parts[0], "revision": old.OriginalRevision, "restored_by": p.Username, "restored_at": time.Now().UTC(), "reason": strings.TrimSpace(input.Reason)}
	if input.Name != "" {
		body["name"] = input.Name
	}
	if input.Version != "" && !semverPattern.MatchString(input.Version) {
		jsonError(w, "version must be semantic version", 400)
		return
	}
	if parser, ok := body["parser"].(map[string]any); ok {
		parser["id"] = input.ID
		body["artifact_checksum"] = parserArtifactChecksum(parser)
	}
	if input.Version != "" {
		body["version"] = input.Version
	}
	payload, _ := json.Marshal(body)
	saved, err := s.metadata.Put(r.Context(), metastore.Document{Collection: "parser-packs", ID: input.ID, State: "draft", Payload: payload, CreatedBy: p.Username})
	if err != nil {
		jsonError(w, "parser restore failed", 503)
		return
	}
	s.auditAction(p, "parser.restore", "parser-packs/"+input.ID, "success", map[string]string{"source": parts[0]})
	jsonCreated(w, map[string]any{"id": input.ID, "revision": saved.Revision, "state": "draft", "restored_from": parts[0], "requires_validation": true, "activated": false})
}

// runtime paths are read-only projections of the authoritative published bundle.
func (s *Server) controllerParserInventory(bundle parserAssetBundle) map[string]any {
	dir := filepath.Join(s.cfg.StateDir, "parser-assets")
	packs := []map[string]any{}
	for _, asset := range bundle.Assets {
		if asset.AssetClass == "parser-packs" {
			packs = append(packs, map[string]any{"id": asset.AssetID, "runtime_id": asset.AssetID, "revision": asset.Revision, "sha256": asset.SHA256, "path": filepath.Join(dir, "active-parsers", asset.AssetID+".json")})
		}
	}
	actual, err := s.readMaterializedParserBundleHash()
	status := "applied"
	if err != nil {
		status = "not_materialized"
	} else if actual != bundle.BundleHash {
		status = "drifted"
	}
	return map[string]any{"bundle_path": filepath.Join(dir, "bundle.json"), "parser_directory": filepath.Join(dir, "active-parsers"), "inventory_path": filepath.Join(dir, "runtime-inventory.json"), "bundle_hash": bundle.BundleHash, "materialized_hash": actual, "status": status, "parsers": packs, "authority": "published_controller_metadata", "baseline_policy": "embedded baseline parsers remain available"}
}

// materializeParserInventory replaces only controller-owned projection files.
// bundle.json remains the authoritative atomic synchronization artifact.
func (s *Server) materializeParserInventory(bundle parserAssetBundle) error {
	root := filepath.Join(s.cfg.StateDir, "parser-assets")
	dir := filepath.Join(root, "active-parsers")
	if err := os.MkdirAll(dir, 0750); err != nil {
		return err
	}
	wanted := map[string]bool{}
	for _, asset := range bundle.Assets {
		if asset.AssetClass != "parser-packs" {
			continue
		}
		if !validRuntimeParserID(asset.AssetID) {
			return fmt.Errorf("unsafe runtime parser identity")
		}
		name := asset.AssetID + ".json"
		wanted[name] = true
		temp, err := os.CreateTemp(dir, ".parser-*")
		if err != nil {
			return err
		}
		if err = temp.Chmod(0640); err == nil {
			_, err = temp.Write(asset.Payload)
		}
		if err == nil {
			err = temp.Sync()
		}
		closeErr := temp.Close()
		if err == nil {
			err = closeErr
		}
		if err == nil {
			err = os.Rename(temp.Name(), filepath.Join(dir, name))
		}
		if err != nil {
			_ = os.Remove(temp.Name())
			return err
		}
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") && !wanted[entry.Name()] {
			if err := os.Remove(filepath.Join(dir, entry.Name())); err != nil {
				return err
			}
		}
	}
	inventory, _ := json.Marshal(s.controllerParserInventory(bundle))
	return os.WriteFile(filepath.Join(root, "runtime-inventory.json"), inventory, 0640)
}

func parserSyncBlocker(ack bool, offline bool, state map[string]any) (string, string) {
	if offline {
		return "client_offline", "Reconnect the selected client before delivering and verifying its parser bundle."
	}
	if ack {
		return "", "Exact published bundle is acknowledged; replay is available."
	}
	if fmt.Sprint(state["asset_sync_status"]) == "failed" {
		return "client_apply_failed", "Inspect the client error, correct it, then retry the current parser bundle rollout and approve it here."
	}
	return "parser_bundle_not_acknowledged", "Prepare the current bundle rollout for this client, approve the rollout here, then refresh verification after the client applies it. Replay unlocks on the exact acknowledgement."
}

func validRuntimeInventory(inventory *protocol.ParserRuntimeInventory) bool {
	if inventory == nil || len(inventory.Files) > 128 {
		return false
	}
	for _, path := range []string{inventory.ParserDirectory, inventory.ManagedManifestPath, inventory.BundlePath, inventory.InventoryPath} {
		if len(path) > 4096 {
			return false
		}
	}
	if len(inventory.ManagedFilesDigest) != 64 {
		return false
	}
	if len(inventory.BundleHash) > 64 {
		return false
	}
	for _, file := range inventory.Files {
		if len(file.ID) > 256 || len(file.Path) > 4096 || len(file.SHA256) != 64 {
			return false
		}
	}
	return true
}

func parserArchiveName(doc metastore.Document) string {
	var body map[string]any
	_ = json.Unmarshal(doc.Payload, &body)
	name, _ := body["name"].(string)
	if name == "" {
		name = doc.ID
	}
	return name
}

func validRuntimeParserID(value string) bool {
	if len(value) == 0 || len(value) > 128 {
		return false
	}
	for i, char := range value {
		if char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' || char >= '0' && char <= '9' || i > 0 && (char == '-' || char == '_' || char == '.') {
			continue
		}
		return false
	}
	return true
}

func runtimeInventoryMatchesBundle(inventory *protocol.ParserRuntimeInventory, bundle parserAssetBundle) bool {
	// Earlier clients do not report file inventories; their authenticated exact
	// bundle acknowledgement remains supported during a rolling upgrade.
	if inventory == nil {
		return true
	}
	if !validRuntimeInventory(inventory) || inventory.BundleHash != bundle.BundleHash {
		return false
	}
	files := []protocol.ParserFileDigest{}
	for _, asset := range bundle.Assets {
		if asset.AssetClass == "parser-packs" {
			files = append(files, protocol.ParserFileDigest{ID: asset.AssetID, SHA256: asset.SHA256})
		}
	}
	return inventory.ManagedFilesDigest == protocol.ManagedParserFilesDigest(files)
}

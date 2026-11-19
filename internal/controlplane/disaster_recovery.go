package controlplane

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"nexora.local/nli-v20/internal/auth"
	"nexora.local/nli-v20/internal/canonicaljson"
	"nexora.local/nli-v20/internal/middleware"
	metastore "nexora.local/nli-v20/internal/store"
)

var recoveryCollections = append(append([]string{}, parserAssetCollections...), "input-paths", "output-destinations")

const recoveryDocumentLimit = 2000
const recoveryPayloadLimit = 32 << 20

type recoveryReference struct {
	Collection string `json:"collection"`
	ID         string `json:"id"`
	Revision   int    `json:"revision"`
	Hash       string `json:"sha256"`
	Archive    string `json:"archive"`
}
type recoveryManifest struct {
	Schema    int                 `json:"schema_version"`
	CreatedAt time.Time           `json:"created_at"`
	Digest    string              `json:"content_sha256"`
	Documents []recoveryReference `json:"documents"`
}

func recoveryDigest(value any) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	data, err = canonicaljson.Normalize(data)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:]), nil
}

func (s *Server) recoveryDocuments(ctx context.Context) ([]metastore.Document, error) {
	documents := []metastore.Document{}
	total := 0
	for _, collection := range recoveryCollections {
		docs, err := s.listParserSyncDocuments(ctx, collection, "", parserSyncTargetLimit)
		if err != nil {
			return nil, err
		}
		for _, doc := range docs {
			if doc.Deleted {
				continue
			}
			if isParserAssetCollection(collection) {
				doc, err = s.effectiveParserAsset(ctx, doc)
				if err != nil {
					return nil, err
				}
				if doc.Revision == 0 {
					continue
				}
			} else if doc.State != "active" {
				continue
			}
			total += len(doc.Payload)
			if total > recoveryPayloadLimit || len(documents) >= recoveryDocumentLimit {
				return nil, errors.New("recovery checkpoint exceeds 2000 documents or 32 MiB")
			}
			documents = append(documents, doc)
		}
	}
	sort.Slice(documents, func(i, j int) bool {
		return documents[i].Collection+"/"+documents[i].ID < documents[j].Collection+"/"+documents[j].ID
	})
	return documents, nil
}

func (s *Server) createRecoveryCheckpoint(ctx context.Context, actor string) (metastore.Document, error) {
	saved, err := s.saveRecoveryCheckpoint(ctx, actor)
	s.recoveryStatusMu.Lock()
	s.recoveryLastAttempt = time.Now().UTC()
	s.recoveryLastError = ""
	if err != nil {
		s.recoveryLastError = err.Error()
	} else {
		s.recoveryLastSuccess = s.recoveryLastAttempt
	}
	s.recoveryStatusMu.Unlock()
	return saved, err
}

func (s *Server) saveRecoveryCheckpoint(ctx context.Context, actor string) (metastore.Document, error) {
	s.parserMutationMu.Lock()
	docs, err := s.recoveryDocuments(ctx)
	s.parserMutationMu.Unlock()
	if err != nil {
		return metastore.Document{}, err
	}
	refs := make([]recoveryReference, 0, len(docs))
	for _, doc := range docs {
		hash, err := recoveryDigest(doc)
		if err != nil {
			return metastore.Document{}, err
		}
		refs = append(refs, recoveryReference{Collection: doc.Collection, ID: doc.ID, Revision: doc.Revision, Hash: hash, Archive: hash})
	}
	digest, err := recoveryDigest(refs)
	if err != nil {
		return metastore.Document{}, err
	}
	id := "catalog-" + digest
	existing, err := s.metadata.Get(ctx, "recovery-checkpoints", id, 0)
	if err == nil && existing.State == "complete" && !existing.Deleted {
		var manifest recoveryManifest
		if json.Unmarshal(existing.Payload, &manifest) != nil {
			return metastore.Document{}, errors.New("stored recovery manifest is invalid")
		}
		return existing, s.exportRecoveryCatalog(docs, manifest)
	}
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return metastore.Document{}, err
	}
	for i, doc := range docs {
		_, err := s.metadata.Get(ctx, "recovery-documents", refs[i].Archive, 0)
		if errors.Is(err, os.ErrNotExist) {
			encoded, _ := json.Marshal(doc)
			_, err = s.metadata.Put(ctx, metastore.Document{Collection: "recovery-documents", ID: refs[i].Archive, State: "archived", Payload: encoded, CreatedBy: actor})
		}
		if err != nil {
			return metastore.Document{}, err
		}
	}
	manifest := recoveryManifest{Schema: 1, CreatedAt: time.Now().UTC(), Digest: digest, Documents: refs}
	payload, _ := json.Marshal(manifest)
	saved, err := s.metadata.Put(ctx, metastore.Document{Collection: "recovery-checkpoints", ID: id, State: "complete", Payload: payload, CreatedBy: actor})
	if err != nil {
		return metastore.Document{}, err
	}
	if err = s.exportRecoveryCatalog(docs, manifest); err != nil {
		return saved, fmt.Errorf("checkpoint stored but local export failed: %w", err)
	}
	return saved, nil
}

func (s *Server) exportRecoveryCatalog(docs []metastore.Document, manifest recoveryManifest) error {
	root := filepath.Join(s.cfg.StateDir, "recovery")
	if err := os.MkdirAll(root, 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(map[string]any{"manifest": manifest, "documents": docs}, "", "  ")
	if err != nil {
		return err
	}
	file, err := os.CreateTemp(root, ".catalog-*")
	if err != nil {
		return err
	}
	name := file.Name()
	defer os.Remove(name)
	if err = file.Chmod(0600); err == nil {
		_, err = file.Write(data)
	}
	if err == nil {
		err = file.Sync()
	}
	closeErr := file.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(name, filepath.Join(root, "catalog.json"))
}

func (s *Server) readRecoveryCheckpoint(ctx context.Context, id string) (recoveryManifest, []metastore.Document, error) {
	var manifest recoveryManifest
	stored, err := s.metadata.Get(ctx, "recovery-checkpoints", id, 0)
	if err != nil || stored.Deleted || stored.State != "complete" {
		return manifest, nil, errors.New("completed recovery checkpoint not found")
	}
	if json.Unmarshal(stored.Payload, &manifest) != nil || manifest.Schema != 1 || len(manifest.Documents) > recoveryDocumentLimit {
		return manifest, nil, errors.New("invalid recovery manifest")
	}
	digest, err := recoveryDigest(manifest.Documents)
	if err != nil || digest != manifest.Digest || id != "catalog-"+digest {
		return manifest, nil, errors.New("recovery manifest digest mismatch")
	}
	allowed := map[string]bool{}
	for _, collection := range recoveryCollections {
		allowed[collection] = true
	}
	docs := make([]metastore.Document, 0, len(manifest.Documents))
	total := 0
	for _, ref := range manifest.Documents {
		archived, err := s.metadata.Get(ctx, "recovery-documents", ref.Archive, 0)
		if err != nil {
			return manifest, nil, err
		}
		var doc metastore.Document
		if json.Unmarshal(archived.Payload, &doc) != nil {
			return manifest, nil, errors.New("invalid archived document")
		}
		hash, err := recoveryDigest(doc)
		if err != nil || hash != ref.Hash || hash != ref.Archive || doc.Collection != ref.Collection || doc.ID != ref.ID || doc.Revision != ref.Revision || !allowed[doc.Collection] || doc.Deleted {
			return manifest, nil, errors.New("archived catalog document failed verification")
		}
		total += len(doc.Payload)
		if total > recoveryPayloadLimit {
			return manifest, nil, errors.New("checkpoint content exceeds recovery limit")
		}
		docs = append(docs, doc)
	}
	return manifest, docs, nil
}

func (s *Server) stageRecoveryDrafts(ctx context.Context, id, actor string) (map[string]any, error) {
	s.parserMutationMu.Lock()
	defer s.parserMutationMu.Unlock()
	_, docs, err := s.readRecoveryCheckpoint(ctx, id)
	if err != nil {
		return nil, err
	}
	staged, unchanged, conflicts := []string{}, []string{}, []string{}
	for _, doc := range docs {
		key := doc.Collection + "/" + doc.ID
		var payload map[string]any
		if json.Unmarshal(doc.Payload, &payload) != nil {
			return nil, errors.New("invalid catalog payload")
		}
		current, err := s.metadata.Get(ctx, doc.Collection, doc.ID, 0)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
		if err == nil && !current.Deleted {
			var now map[string]any
			_ = json.Unmarshal(current.Payload, &now)
			if current.State == "draft" && now["recovery_checkpoint"] == id {
				staged = append(staged, key)
				continue
			}
			same, _ := recoveryDigest(current.Payload)
			prior, _ := recoveryDigest(doc.Payload)
			if same == prior {
				unchanged = append(unchanged, key)
				continue
			}
			if doc.Collection == "parser-packs" || current.State != "active" {
				conflicts = append(conflicts, key)
				continue
			}
			payload["previous_active_revision"] = current.Revision
		}
		if doc.Collection == "parser-packs" {
			if _, _, err := runtimeParserPackForDocumentContext(ctx, doc); err != nil {
				return nil, fmt.Errorf("%s: %w", key, err)
			}
		} else {
			config, _ := payload["config"].(map[string]any)
			if err := validateCatalogConfig(doc.Collection, config); err != nil {
				return nil, fmt.Errorf("%s: %w", key, err)
			}
		}
		for _, field := range []string{"approved_by", "approved_at", "published_by", "published_at", "submitted_by", "submitted_at", "reviewer", "approved_checksum"} {
			delete(payload, field)
		}
		payload["recovery_checkpoint"] = id
		payload["recovery_source_revision"] = doc.Revision
		encoded, _ := json.Marshal(payload)
		_, err = s.metadata.Put(ctx, metastore.Document{Collection: doc.Collection, ID: doc.ID, State: "draft", Payload: encoded, CreatedBy: actor})
		if err != nil {
			return map[string]any{"staged": staged, "conflicts": conflicts, "retry_checkpoint": id}, err
		}
		staged = append(staged, key)
	}
	return map[string]any{"staged": staged, "unchanged": unchanged, "conflicts": conflicts, "review_required": true}, nil
}

func (s *Server) handleDisasterRecovery(w http.ResponseWriter, r *http.Request) {
	principal := s.authorize(w, r, middleware.PermissionSystemHealth)
	if principal == nil {
		return
	}
	if principal.Role != auth.RoleAdmin {
		jsonError(w, "administrator access required", http.StatusForbidden)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	action := strings.TrimPrefix(r.URL.Path, "/api/v1/recovery")
	if r.Method == http.MethodGet && action == "" {
		docs, err := s.metadata.List(r.Context(), "recovery-checkpoints", metastore.Query{State: "complete", Limit: 100})
		if err != nil {
			jsonError(w, "recovery checkpoint catalog unavailable", http.StatusServiceUnavailable)
			return
		}
		items := []map[string]any{}
		for _, doc := range docs {
			var value recoveryManifest
			if json.Unmarshal(doc.Payload, &value) == nil {
				items = append(items, map[string]any{"id": doc.ID, "created_at": value.CreatedAt, "sha256": value.Digest, "documents": len(value.Documents)})
			}
		}
		s.recoveryStatusMu.Lock()
		status := map[string]any{"last_attempt": s.recoveryLastAttempt, "last_success": s.recoveryLastSuccess, "last_error": s.recoveryLastError, "automatic_enabled": s.metadata.Mode() == "supabase"}
		s.recoveryStatusMu.Unlock()
		capsules, err := s.metadata.List(r.Context(), "recovery-capsules", metastore.Query{State: "complete", Limit: 100})
		if err != nil {
			jsonError(w, "host recovery capsule catalog unavailable", http.StatusServiceUnavailable)
			return
		}
		hostItems := []map[string]any{}
		for _, doc := range capsules {
			var manifest recoveryCapsuleManifest
			if json.Unmarshal(doc.Payload, &manifest) == nil {
				hostItems = append(hostItems, map[string]any{"id": doc.ID, "created_at": manifest.CreatedAt, "sha256": manifest.SHA256, "bytes": manifest.Bytes})
			}
		}
		jsonOK(w, map[string]any{"status": status, "host_capsules": hostItems, "storage_mode": s.metadata.Mode(), "checkpoints": items, "automatic_interval_seconds": 300, "catalog_export": filepath.Join(s.cfg.StateDir, "recovery", "catalog.json"), "controller_state": s.cfg.StateDir, "controller_config": s.cfg.RecoveryConfigDir, "controller_parser_assets": filepath.Join(s.cfg.StateDir, "parser-assets"), "client_parser_assets": "/var/lib/nli/parser-assets", "client_parsers": "/etc/nli/parsers", "identity_dependencies": []string{"/etc/nli controller configuration, environment credentials and TLS trust", "controller state users.json, signing/encryption keys, fleet.json and rollout state", "client node-id, node-token and parser/control configuration"}})
		return
	}
	if r.Method == http.MethodGet && strings.HasPrefix(action, "/capsule/") {
		id := strings.TrimPrefix(action, "/capsule/")
		if len(id) != 69 || !strings.HasPrefix(id, "host-") {
			jsonError(w, "invalid capsule id", http.StatusBadRequest)
			return
		}
		s.handleRecoveryCapsuleDownload(w, r, id)
		return
	}
	if r.Method == http.MethodPost && action == "/capsule" {
		var input struct {
			Capsule json.RawMessage `json:"capsule"`
		}
		if decodeJSON(r, &input, 7<<20) != nil || len(input.Capsule) > 6<<20 {
			jsonError(w, "encrypted capsule JSON required (up to 6 MiB)", http.StatusBadRequest)
			return
		}
		id, err := s.importRecoveryCapsule(r.Context(), input.Capsule, principal.Username)
		if err != nil {
			jsonError(w, "host recovery capsule failed: "+err.Error(), http.StatusServiceUnavailable)
			return
		}
		s.auditAction(principal, "recovery.host-capsule", id, "success", nil)
		jsonOK(w, map[string]any{"id": id, "encrypted": true})
		return
	}
	if r.Method == http.MethodPost && action == "/checkpoint" {
		doc, err := s.createRecoveryCheckpoint(r.Context(), principal.Username)
		if err != nil {
			jsonError(w, "recovery checkpoint failed: "+err.Error(), http.StatusServiceUnavailable)
			return
		}
		s.auditAction(principal, "recovery.checkpoint", doc.ID, "success", nil)
		jsonOK(w, map[string]any{"id": doc.ID, "state": doc.State})
		return
	}
	if r.Method == http.MethodPost && action == "/resync" {
		s.parserMutationMu.Lock()
		defer s.parserMutationMu.Unlock()
		bundle, err := s.buildActiveParserBundle(r.Context())
		if err == nil {
			_, err = s.materializeParserBundle(bundle)
		}
		if err != nil {
			jsonError(w, "authoritative bundle recovery failed: "+err.Error(), http.StatusServiceUnavailable)
			return
		}
		proposal, err := s.proposeParserBundleRollout(r.Context(), principal.Username, "Recover published parser assets", "Rebuild controller assets and request verified fleet resynchronization", "all_eligible", bundle, nil)
		if err != nil {
			jsonError(w, "controller assets recovered; fleet proposal failed: "+err.Error(), http.StatusServiceUnavailable)
			return
		}
		s.auditAction(principal, "recovery.resync", bundle.BundleHash, "success", nil)
		jsonOK(w, map[string]any{"bundle_hash": bundle.BundleHash, "rollout_proposal": documentView(proposal)})
		return
	}
	if r.Method == http.MethodPost && action == "/restore" {
		var input struct {
			Checkpoint string `json:"checkpoint"`
		}
		if decodeJSON(r, &input, 1024) != nil || !strings.HasPrefix(input.Checkpoint, "catalog-") {
			jsonError(w, "checkpoint id required", http.StatusBadRequest)
			return
		}
		report, err := s.stageRecoveryDrafts(r.Context(), input.Checkpoint, principal.Username)
		if err != nil {
			jsonError(w, "recovery draft staging failed; retry the same checkpoint: "+err.Error(), http.StatusServiceUnavailable)
			return
		}
		s.auditAction(principal, "recovery.restore-drafts", input.Checkpoint, "success", nil)
		jsonOK(w, report)
		return
	}
	jsonError(w, "unsupported recovery action", http.StatusMethodNotAllowed)
}

// Published configuration is already committed synchronously to the protected
// metadata store. Checkpoints retain an independent recovery view every five
// minutes while the authoritative store is reachable.
func (s *Server) startRecoveryCheckpoints() {
	if s.metadata.Mode() != "supabase" {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.recoveryCancel = cancel
	s.recoveryDone = make(chan struct{})
	go func() {
		defer close(s.recoveryDone)
		timer := time.NewTimer(10 * time.Second)
		defer timer.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-timer.C:
				work, done := context.WithTimeout(ctx, 2*time.Minute)
				_, _ = s.createRecoveryCheckpoint(work, "recovery-scheduler")
				done()
				timer.Reset(5 * time.Minute)
			}
		}
	}()
}

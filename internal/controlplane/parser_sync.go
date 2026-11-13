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

	"nexora.local/nli-v20/internal/canonicaljson"
	"nexora.local/nli-v20/internal/fieldmapping"
	"nexora.local/nli-v20/internal/fleet"
	"nexora.local/nli-v20/internal/middleware"
	"nexora.local/nli-v20/internal/protocol"
	"nexora.local/nli-v20/internal/rollout"
	metastore "nexora.local/nli-v20/internal/store"
)

var parserAssetCollections = []string{"kv-config", "static-patterns", "projections", "templates", "regex-patterns", "known-keys", "semantic-extraction", "parser-packs"}

const (
	parserSyncPageSize    = 1000
	parserSyncTargetLimit = 10000
)

type parserBundleAsset struct {
	AssetClass string          `json:"asset_class"`
	AssetID    string          `json:"asset_id"`
	Name       string          `json:"name"`
	Revision   int             `json:"revision"`
	SHA256     string          `json:"sha256"`
	Payload    json.RawMessage `json:"payload"`
}
type parserAssetBundle struct {
	SchemaVersion int                 `json:"schema_version"`
	Assets        []parserBundleAsset `json:"assets"`
	BundleHash    string              `json:"bundle_hash"`
}

type parserSyncProposalSnapshot struct {
	BundleHash string `json:"bundle_hash"`
	Payload    struct {
		Bundle     parserAssetBundle `json:"asset_bundle"`
		BundleHash string            `json:"bundle_hash"`
	} `json:"payload"`
}

type verifiedParserSyncProposal struct {
	Bundle         parserAssetBundle
	CanonicalHash  string
	StoredHash     string
	ValidationMode string
}

// legacyBundleDigest is the pre-R5 digest. It is retained only to verify and
// safely migrate pending proposals created by older controllers. New bundles
// must always use bundleDigest, whose input is independent of JSON storage
// formatting.
func legacyBundleDigest(bundle parserAssetBundle) (string, []byte, error) {
	bundle.BundleHash = ""
	unsigned, err := json.Marshal(struct {
		SchemaVersion int                 `json:"schema_version"`
		Assets        []parserBundleAsset `json:"assets"`
	}{bundle.SchemaVersion, bundle.Assets})
	if err != nil {
		return "", nil, err
	}
	digest := sha256.Sum256(unsigned)
	return hex.EncodeToString(digest[:]), unsigned, nil
}

// bundleDigest hashes a semantic, canonical JSON representation. Metadata may
// pass through PostgreSQL jsonb (or another JSON-aware store), which is allowed
// to reorder object keys and rewrite equivalent numbers. Hashing raw
// json.RawMessage bytes made an otherwise immutable proposal unverifiable after
// a store round trip. Canonicalization keeps arrays ordered, sorts object keys,
// and gives every equivalent JSON number one representation.
func bundleDigest(bundle parserAssetBundle) (string, []byte, error) {
	bundle.BundleHash = ""
	assets := append([]parserBundleAsset(nil), bundle.Assets...)
	for index := range assets {
		canonical, err := canonicaljson.Normalize(assets[index].Payload)
		if err != nil {
			return "", nil, fmt.Errorf("canonical parser asset %s/%s: %w", assets[index].AssetClass, assets[index].AssetID, err)
		}
		assets[index].Payload = canonical
		assetDigest := sha256.Sum256(canonical)
		assets[index].SHA256 = hex.EncodeToString(assetDigest[:])
	}
	unsigned, err := json.Marshal(struct {
		SchemaVersion int                 `json:"schema_version"`
		Assets        []parserBundleAsset `json:"assets"`
	}{bundle.SchemaVersion, assets})
	if err != nil {
		return "", nil, err
	}
	digest := sha256.Sum256(unsigned)
	return hex.EncodeToString(digest[:]), unsigned, nil
}

func validLegacyBundleHash(hash string) bool {
	if len(hash) < 16 || len(hash) > sha256.Size*2 {
		return false
	}
	for _, char := range hash {
		if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f') || (char >= 'A' && char <= 'F')) {
			return false
		}
	}
	return true
}

func storedBundleHashMatches(stored string, candidates ...string) bool {
	stored = strings.ToLower(strings.TrimSpace(stored))
	for _, candidate := range candidates {
		candidate = strings.ToLower(strings.TrimSpace(candidate))
		if len(stored) == sha256.Size*2 && stored == candidate {
			return true
		}
		if len(stored) < sha256.Size*2 && strings.HasPrefix(candidate, stored) {
			return true
		}
	}
	return false
}

// verifyParserSyncProposal validates the immutable stored snapshot before any
// approval side effect. Legacy hashes are accepted only when they identify the
// same semantic bundle as the controller's current published bundle. This
// permits a safe one-way upgrade to the canonical SHA-256 without allowing a
// stale or arbitrary legacy proposal to bypass the approval gate.
func (s *Server) verifyParserSyncProposal(ctx context.Context, raw json.RawMessage) (verifiedParserSyncProposal, error) {
	var snapshot parserSyncProposalSnapshot
	if err := json.Unmarshal(raw, &snapshot); err != nil {
		return verifiedParserSyncProposal{}, errors.New("parser sync proposal cannot be approved: bundle snapshot JSON is invalid")
	}
	hashFields := []string{snapshot.BundleHash, snapshot.Payload.BundleHash, snapshot.Payload.Bundle.BundleHash}
	storedHash := ""
	missingFields := false
	for _, hash := range hashFields {
		hash = strings.TrimSpace(hash)
		if hash == "" {
			missingFields = true
			continue
		}
		if storedHash == "" {
			storedHash = hash
			continue
		}
		if !strings.EqualFold(storedHash, hash) {
			return verifiedParserSyncProposal{}, errors.New("parser sync proposal cannot be approved: stored bundle hash fields disagree; create a new proposal")
		}
	}
	if storedHash == "" {
		return verifiedParserSyncProposal{}, errors.New("parser sync proposal cannot be approved: no bundle hash is stored; create a new proposal")
	}
	if !validLegacyBundleHash(storedHash) {
		return verifiedParserSyncProposal{}, errors.New("parser sync proposal cannot be approved: stored bundle hash is malformed; create a new proposal")
	}
	canonicalSnapshotHash, _, err := bundleDigest(snapshot.Payload.Bundle)
	if err != nil {
		return verifiedParserSyncProposal{}, errors.New("parser sync proposal cannot be approved: bundle snapshot cannot be canonicalized")
	}
	legacySnapshotHash, _, legacyErr := legacyBundleDigest(snapshot.Payload.Bundle)
	if legacyErr != nil {
		return verifiedParserSyncProposal{}, errors.New("parser sync proposal cannot be approved: legacy bundle snapshot cannot be verified")
	}
	current, err := s.buildActiveParserBundle(ctx)
	if err != nil {
		return verifiedParserSyncProposal{}, errors.New("parser sync proposal cannot be approved while the published parser bundle is unavailable")
	}
	if canonicalSnapshotHash != current.BundleHash {
		return verifiedParserSyncProposal{}, fmt.Errorf("parser sync proposal is stale: snapshot %s does not match current published bundle %s; create a new proposal", canonicalSnapshotHash, current.BundleHash)
	}
	legacyCurrentHash, _, legacyCurrentErr := legacyBundleDigest(current)
	if legacyCurrentErr != nil {
		return verifiedParserSyncProposal{}, errors.New("parser sync proposal cannot be approved: current legacy bundle cannot be verified")
	}
	hashMatched := storedBundleHashMatches(storedHash, canonicalSnapshotHash, legacySnapshotHash, legacyCurrentHash)
	mode := "canonical_sha256"
	if !hashMatched {
		// R4 hashed the raw bytes nested inside the proposal before PostgreSQL
		// jsonb storage. jsonb may lawfully reorder object keys, so that original
		// full digest can be impossible to reconstruct. At this point all three
		// stored hash fields agree, the value is a full SHA-256, and the stored
		// semantic snapshot has independently been proven identical to the freshly
		// built published bundle. Migrating that exact current content is safe;
		// short or malformed unbound values remain rejected.
		if len(strings.TrimSpace(storedHash)) != sha256.Size*2 {
			return verifiedParserSyncProposal{}, errors.New("parser sync proposal cannot be safely upgraded: its legacy short hash does not identify the current bundle; create a new proposal")
		}
		mode = "legacy_jsonb_hash_migrated"
	} else if missingFields || !strings.EqualFold(storedHash, canonicalSnapshotHash) || storedHash != strings.ToLower(storedHash) || len(storedHash) != sha256.Size*2 {
		mode = "legacy_hash_upgraded"
	}
	return verifiedParserSyncProposal{
		// Use the freshly rebuilt, canonically encoded wire bundle after proving
		// that it is semantically identical to the immutable proposal snapshot.
		// This also repairs JSONB formatting changes before client-side asset SHA
		// verification without changing the approved content.
		Bundle: current, CanonicalHash: canonicalSnapshotHash,
		StoredHash: storedHash, ValidationMode: mode,
	}, nil
}

func setCanonicalProposalBundleHash(proposal map[string]any, hash string) {
	proposal["bundle_hash"] = hash
	payload, ok := proposal["payload"].(map[string]any)
	if !ok {
		return
	}
	payload["bundle_hash"] = hash
	bundle, ok := payload["asset_bundle"].(map[string]any)
	if ok {
		bundle["bundle_hash"] = hash
	}
}

// listParserSyncDocuments reads bounded metadata in stable pages. Parser sync
// used to stop after the first 1,000 rows, which made acknowledgements and
// proposals beyond that point invisible in large fleets. The 10,000 row guard
// is an explicit operational bound rather than a silent truncation.
func (s *Server) listParserSyncDocuments(ctx context.Context, collection, state string, maximum int) ([]metastore.Document, error) {
	if maximum <= 0 || maximum > parserSyncTargetLimit {
		maximum = parserSyncTargetLimit
	}
	documents := make([]metastore.Document, 0, min(parserSyncPageSize, maximum))
	for offset := 0; offset < maximum; offset += parserSyncPageSize {
		limit := parserSyncPageSize
		if remaining := maximum - offset; remaining < limit {
			limit = remaining
		}
		page, err := s.metadata.List(ctx, collection, metastore.Query{State: state, Limit: limit, Offset: offset})
		if err != nil {
			return nil, err
		}
		documents = append(documents, page...)
		if len(page) < limit {
			return documents, nil
		}
	}
	extra, err := s.metadata.List(ctx, collection, metastore.Query{State: state, Limit: 1, Offset: maximum})
	if err != nil {
		return nil, err
	}
	if len(extra) > 0 {
		return nil, fmt.Errorf("%s metadata exceeds the %d-row parser sync safety limit", collection, maximum)
	}
	return documents, nil
}

// Keep Fleet Sync and replay on the same acknowledgement contract.
func parserBundleAcknowledged(state map[string]any, hash string) bool {
	return hash != "" &&
		state["desired_asset_bundle_hash"] == hash &&
		state["active_asset_bundle_hash"] == hash &&
		state["asset_sync_status"] == "applied" &&
		state["asset_drifted"] != true
}

func (s *Server) buildActiveParserBundle(ctx context.Context) (parserAssetBundle, error) {
	bundle := parserAssetBundle{SchemaVersion: 1, Assets: []parserBundleAsset{}}
	seenRuntimeParserID := map[string]string{}
	seenRuntimePayload := map[string]string{}
	for _, collection := range parserAssetCollections {
		page, err := s.listParserSyncDocuments(ctx, collection, "", parserSyncTargetLimit)
		if err != nil {
			return bundle, err
		}
		// Storage backends are not required to return the same order. Sort before
		// compatibility reconciliation so collision diagnostics are deterministic.
		sort.Slice(page, func(i, j int) bool { return page[i].ID < page[j].ID })
		for _, doc := range page {
			if doc.State != "active" || doc.Deleted {
				if collection != "parser-packs" || doc.State != "published" || doc.Deleted {
					continue
				}
			}
			var payload map[string]any
			if err := json.Unmarshal(doc.Payload, &payload); err != nil {
				// A malformed parser record from an older release must not take the
				// entire fleet bundle offline. It remains visible as blocked in the
				// catalog and cannot be republished until corrected.
				if collection == "parser-packs" {
					continue
				}
				return bundle, fmt.Errorf("invalid active parser asset %s/%s", collection, doc.ID)
			}
			if collection == "parser-packs" {
				// Reconcile both current Assurance envelopes and valid parser packs
				// imported from older NLI releases. Invalid stale records are isolated
				// and exposed as blocked by the catalog API instead of breaking sync.
				pack, encoded, compatibilityErr := runtimeParserPackForDocumentContext(ctx, doc)
				if ctx.Err() != nil {
					return bundle, ctx.Err()
				}
				if compatibilityErr != nil {
					continue
				}
				canonicalPayload, canonicalErr := canonicaljson.Normalize(encoded)
				if canonicalErr != nil {
					return bundle, fmt.Errorf("canonical parser pack %s: %w", doc.ID, canonicalErr)
				}
				if existingCatalogID, duplicate := seenRuntimeParserID[pack.ID]; duplicate && existingCatalogID != doc.ID {
					if seenRuntimePayload[pack.ID] == string(canonicalPayload) {
						continue
					}
					return bundle, fmt.Errorf("active parser runtime identity %q conflicts between catalog documents %q and %q", pack.ID, existingCatalogID, doc.ID)
				}
				seenRuntimeParserID[pack.ID] = doc.ID
				seenRuntimePayload[pack.ID] = string(canonicalPayload)
				digest := sha256.Sum256(canonicalPayload)
				name, _ := payload["name"].(string)
				if name == "" {
					name = pack.ID
				}
				// Runtime synchronization keys parser packs by ParserPack.ID. A legacy
				// catalog record can have a separate catalog ID, so the bundle must use
				// the reconciled runtime identity while the controller retains doc.ID.
				bundle.Assets = append(bundle.Assets, parserBundleAsset{AssetClass: collection, AssetID: pack.ID, Name: name, Revision: doc.Revision, SHA256: hex.EncodeToString(digest[:]), Payload: canonicalPayload})
				continue
			}
			name, _ := payload["name"].(string)
			canonicalPayload, canonicalErr := canonicaljson.Normalize(doc.Payload)
			if canonicalErr != nil {
				return bundle, fmt.Errorf("canonical parser asset %s/%s: %w", collection, doc.ID, canonicalErr)
			}
			digest := sha256.Sum256(canonicalPayload)
			bundle.Assets = append(bundle.Assets, parserBundleAsset{AssetClass: collection, AssetID: doc.ID, Name: name, Revision: doc.Revision, SHA256: hex.EncodeToString(digest[:]), Payload: canonicalPayload})
		}
	}
	// Stable order makes the same approved asset set produce the same bundle hash.
	sort.Slice(bundle.Assets, func(i, j int) bool {
		if bundle.Assets[i].AssetClass == bundle.Assets[j].AssetClass {
			return bundle.Assets[i].AssetID < bundle.Assets[j].AssetID
		}
		return bundle.Assets[i].AssetClass < bundle.Assets[j].AssetClass
	})
	hash, _, err := bundleDigest(bundle)
	bundle.BundleHash = hash
	return bundle, err
}

func (s *Server) materializeParserBundle(bundle parserAssetBundle) (string, error) {
	s.parserMaterializeMu.Lock()
	defer s.parserMaterializeMu.Unlock()
	hash, _, err := bundleDigest(bundle)
	if err != nil {
		return "", err
	}
	if hash != bundle.BundleHash {
		return "", errors.New("parser bundle hash calculation failed")
	}
	encoded, err := json.Marshal(bundle)
	if err != nil {
		return "", err
	}
	dir := filepath.Join(s.cfg.StateDir, "parser-assets")
	if err := os.MkdirAll(dir, 0750); err != nil {
		return "", err
	}
	path := filepath.Join(dir, "bundle.json")
	temp := path + ".staging"
	file, err := os.OpenFile(temp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0640)
	if err != nil {
		return "", err
	}
	if _, err = file.Write(encoded); err == nil {
		err = file.Sync()
	}
	closeErr := file.Close()
	if err != nil {
		return "", err
	}
	if closeErr != nil {
		return "", closeErr
	}
	if err = os.Rename(temp, path); err != nil {
		return "", err
	}
	readback, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	var actual parserAssetBundle
	if err = json.Unmarshal(readback, &actual); err != nil {
		return "", err
	}
	readHash, _, err := bundleDigest(actual)
	if err != nil || readHash != bundle.BundleHash {
		return "", errors.New("controller parser bundle readback hash mismatch")
	}
	if err := s.materializeParserInventory(bundle); err != nil {
		return "", err
	}
	return readHash, nil
}

func (s *Server) readMaterializedParserBundleHash() (string, error) {
	data, err := os.ReadFile(filepath.Join(s.cfg.StateDir, "parser-assets", "bundle.json"))
	if err != nil {
		return "", err
	}
	var bundle parserAssetBundle
	if err := json.Unmarshal(data, &bundle); err != nil {
		return "", err
	}
	hash, _, err := bundleDigest(bundle)
	if err != nil || hash != bundle.BundleHash {
		return "", errors.New("materialized parser bundle hash is invalid")
	}
	return hash, nil
}

const (
	kvOutputEditsMinimumRelease = "22.1.0-0"
	kvInputEditsMinimumRelease  = "22.2.0-0"
	advancedKVMinimumRelease    = "22.11.0-0"
	largeKVMinimumRelease       = "22.12.0-0"
)

// Inspect native stage content so imported packs cannot bypass the release
// requirement by omitting their informational minimum_client_release metadata.
func parserBundleMinimumKVRelease(bundle parserAssetBundle) string {
	minimum := ""
	for _, asset := range bundle.Assets {
		if asset.AssetClass != "parser-packs" {
			continue
		}
		var pack struct {
			Rules []struct {
				Stages []struct {
					Type   string `json:"type"`
					Config struct {
						Fields []fieldmapping.Field `json:"fields"`
					} `json:"config"`
				} `json:"stages"`
			} `json:"rules"`
		}
		_ = json.Unmarshal(asset.Payload, &pack)
		for _, rule := range pack.Rules {
			for _, stage := range rule.Stages {
				if stage.Type != "mapping" {
					continue
				}
				for _, field := range stage.Config.Fields {
					if field.UsesLargeKV() {
						return largeKVMinimumRelease
					}
					if field.UsesAdvancedKV() {
						minimum = advancedKVMinimumRelease
					} else if len(field.KVInputEdits) > 0 && minimum != advancedKVMinimumRelease {
						minimum = kvInputEditsMinimumRelease
					} else if len(field.KeyValueEdits) > 0 && minimum == "" {
						minimum = kvOutputEditsMinimumRelease
					}
				}
			}
		}
	}
	return minimum
}

func parserClientSupportsKVRelease(version, minimumRelease string) bool {
	if minimumRelease == "" {
		return true
	}
	version = strings.TrimPrefix(strings.TrimSpace(version), "v")
	base, _, _ := strings.Cut(version, "-")
	minimum, _, _ := strings.Cut(minimumRelease, "-")
	comparison, err := protocol.CompareSemver(base, minimum)
	return err == nil && comparison >= 0
}

func parserSyncClientBlocker(node *fleet.Node, minimumRelease string, acknowledged bool, state map[string]any) (string, string) {
	if !parserClientSupportsKVRelease(node.Version, minimumRelease) {
		return "client_upgrade_required", fmt.Sprintf("Upgrade this client to v%s or newer to apply the bundle's KV edits.", minimumRelease)
	}
	return parserSyncBlocker(acknowledged, node.Status == fleet.StatusOffline, state)
}

// The shipped baseline is approved with the release. Site-specific bundles
// still need rollout approval before they are delivered to a new node.
func (s *Server) queueEnrollmentParserSync(ctx context.Context, nodeID string) {
	s.parserMutationMu.Lock()
	defer s.parserMutationMu.Unlock()
	const actor = "nli-enrollment"
	const name = "Initial parser asset synchronization"
	targets := []string{nodeID}
	bundle, err := s.buildActiveParserBundle(ctx)
	if err == nil {
		if isBuiltinEnrollmentBundle(bundle) {
			_, err = s.startParserBundleRollout(ctx, actor, name, bundle, targets)
		} else {
			_, err = s.proposeParserBundleRollout(ctx, actor, name, "Site-specific parser assets require rollout approval for this newly enrolled client.", "selected", bundle, targets)
		}
	}
	if err != nil {
		s.auditAction(nil, "parser.enrollment_sync", "nodes/"+nodeID, "failure", map[string]string{"error": err.Error()})
	}
}

func (s *Server) startParserBundleRollout(ctx context.Context, actor, name string, bundle parserAssetBundle, targetIDs []string) (*rolloutRecord, error) {
	if len(targetIDs) == 0 {
		for _, node := range s.fleet.All() {
			if node.Status != fleet.StatusOffline {
				targetIDs = append(targetIDs, node.ID)
			}
		}
	}
	if minimum := parserBundleMinimumKVRelease(bundle); minimum != "" {
		compatible := make([]string, 0, len(targetIDs))
		for _, target := range targetIDs {
			if node, ok := s.fleet.Get(target); ok && node != nil && parserClientSupportsKVRelease(node.Version, minimum) {
				compatible = append(compatible, target)
			}
		}
		targetIDs = compatible
	}
	if len(targetIDs) == 0 {
		return nil, nil
	}
	payload := map[string]any{"asset_bundle": bundle, "bundle_hash": bundle.BundleHash}
	rollout, err := s.rollouts.Create(rolloutTypeParserSync, name, actor, targetIDs, payload)
	if err != nil {
		return nil, err
	}
	if err := s.rollouts.Start(rollout.ID, targetIDs); err != nil {
		return nil, err
	}
	for _, nodeID := range targetIDs {
		s.updateDesiredNodeAssetSync(ctx, nodeID, bundle)
	}
	item, ok := s.rollouts.Get(rollout.ID)
	if !ok {
		return nil, errors.New("created parser sync rollout disappeared")
	}
	return item, nil
}

// proposeParserBundleRollout snapshots the exact published controller bundle
// and target set for independent approval. Clients do not receive the bundle
// until the separate rollout approval endpoint starts the live rollout.
func (s *Server) proposeParserBundleRollout(ctx context.Context, actor, name, reason, targetMode string, bundle parserAssetBundle, targetIDs []string) (metastore.Document, error) {
	if targetMode == "" {
		targetMode = "all_eligible"
	}
	if targetMode != "all_eligible" && targetMode != "selected" && targetMode != "origin" {
		return metastore.Document{}, errors.New("target_mode must be all_eligible, selected, or origin")
	}
	eligible := map[string]bool{}
	minimum := parserBundleMinimumKVRelease(bundle)
	for _, node := range s.fleet.All() {
		if node.Status != fleet.StatusOffline && parserClientSupportsKVRelease(node.Version, minimum) {
			eligible[node.ID] = true
		}
	}
	if targetMode == "all_eligible" {
		targetIDs = targetIDs[:0]
		for id := range eligible {
			targetIDs = append(targetIDs, id)
		}
	}
	if len(targetIDs) > parserSyncTargetLimit || (len(targetIDs) == 0 && targetMode != "all_eligible") {
		return metastore.Document{}, fmt.Errorf("parser sync rollout requires 1..%d eligible target clients", parserSyncTargetLimit)
	}
	seen := map[string]bool{}
	for _, id := range targetIDs {
		if !eligible[id] || seen[id] {
			if node, ok := s.fleet.Get(id); ok && node != nil && !parserClientSupportsKVRelease(node.Version, minimum) {
				return metastore.Document{}, fmt.Errorf("parser sync target %q must be upgraded to v%s for the bundle's KV edits", id, minimum)
			}
			return metastore.Document{}, fmt.Errorf("parser sync target %q is not an eligible enrolled client", id)
		}
		seen[id] = true
	}
	sort.Strings(targetIDs)
	// A bundle hash is the immutable identity of parser content. Reusing an
	// existing pending proposal for the same content and exact target set keeps
	// repeated UI actions idempotent. Conversely, once a newer canonical bundle
	// exists, an older pending parser proposal must not remain actionable: its
	// approval can only fail the exact-hash gate and was the source of the R6
	// "bundle hash is invalid/stale" loop. Preserve it as audit history while
	// removing it from the approval queue.
	pending, listErr := s.listParserSyncDocuments(ctx, "rollout-proposals", "", parserSyncTargetLimit)
	if listErr != nil {
		return metastore.Document{}, fmt.Errorf("list parser rollout proposals: %w", listErr)
	}
	var superseded []metastore.Document
	for _, existing := range pending {
		if existing.State != "pending_approval" || existing.Deleted {
			continue
		}
		var existingPayload map[string]any
		if json.Unmarshal(existing.Payload, &existingPayload) != nil || fmt.Sprint(existingPayload["type"]) != rolloutTypeParserSync {
			continue
		}
		existingTargets := interfaceStrings(existingPayload["target_nodes"])
		sort.Strings(existingTargets)
		existingMode := strings.TrimSpace(fmt.Sprint(existingPayload["target_mode"]))
		if existingMode == "" || existingMode == "<nil>" {
			existingMode = "all_eligible"
		}
		if strings.EqualFold(strings.TrimSpace(fmt.Sprint(existingPayload["bundle_hash"])), bundle.BundleHash) {
			if existingMode == targetMode && equalStringSlices(existingTargets, targetIDs) {
				return existing, nil
			}
			// A targeted repair must not withdraw another client’s current approval.
			continue
		}
		superseded = append(superseded, existing)
	}
	proposal := map[string]any{
		"name": name, "type": rolloutTypeParserSync, "target_nodes": targetIDs,
		"target_mode":       targetMode,
		"payload":           map[string]any{"asset_bundle": bundle, "bundle_hash": bundle.BundleHash},
		"approval_required": true, "approval_reason": reason,
		"bundle_hash": bundle.BundleHash,
	}
	encoded, err := json.Marshal(proposal)
	if err != nil {
		return metastore.Document{}, err
	}
	id := fmt.Sprintf("parser-sync-%d", time.Now().UnixNano())
	created, err := s.metadata.Put(ctx, metastore.Document{Collection: "rollout-proposals", ID: id, State: "pending_approval", Payload: encoded, CreatedBy: actor})
	if err != nil {
		return metastore.Document{}, err
	}
	for _, stale := range superseded {
		var stalePayload map[string]any
		if json.Unmarshal(stale.Payload, &stalePayload) != nil {
			continue
		}
		stalePayload["status"] = "superseded"
		stalePayload["superseded_by"] = created.ID
		stalePayload["superseded_by_bundle_hash"] = bundle.BundleHash
		stalePayload["superseded_at"] = time.Now().UTC()
		stalePayload["superseded_reason"] = "a newer canonical parser bundle replaced this pending proposal"
		staleEncoded, marshalErr := json.Marshal(stalePayload)
		if marshalErr != nil {
			continue
		}
		_, _ = s.metadata.Put(ctx, metastore.Document{Collection: stale.Collection, ID: stale.ID, State: "superseded", Payload: staleEncoded, CreatedBy: stale.CreatedBy})
	}
	return created, nil
}

func equalStringSlices(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

// hasCompletedParserDelivery recovers desired state lost by older heartbeat
// writers. The reported hash alone cannot establish that delivery was approved.
func (m *rolloutStateManager) hasCompletedParserDelivery(nodeID, hash string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, item := range m.items {
		if item.Type != rolloutTypeParserSync {
			continue
		}
		result := item.NodeResults[nodeID]
		if result == nil || result.Status != rolloutStatusCompleted {
			continue
		}
		var delivery struct {
			BundleHash string `json:"bundle_hash"`
		}
		if json.Unmarshal(item.Payload, &delivery) == nil && delivery.BundleHash == hash {
			return true
		}
	}
	return false
}

func (s *Server) updateDesiredNodeAssetSync(ctx context.Context, nodeID string, bundle parserAssetBundle) {
	s.fleetSyncMu.Lock()
	defer s.fleetSyncMu.Unlock()
	id := safeID(nodeID)
	doc, _ := s.metadata.Get(ctx, "fleet-sync", id, 0)
	state := map[string]any{"node_id": nodeID}
	_ = json.Unmarshal(doc.Payload, &state)
	revisions := map[string]int{}
	for _, asset := range bundle.Assets {
		key := asset.AssetClass + "/" + asset.AssetID
		revisions[key] = asset.Revision
	}
	state["desired_asset_bundle_hash"] = bundle.BundleHash
	state["desired_asset_revisions"] = revisions
	state["desired_asset_at"] = time.Now().UTC()
	state["asset_drifted"] = fmt.Sprint(state["active_asset_bundle_hash"]) != bundle.BundleHash
	if fmt.Sprint(state["active_asset_bundle_hash"]) != bundle.BundleHash || fmt.Sprint(state["asset_sync_status"]) != "applied" {
		state["asset_sync_status"] = "pending"
	}
	payload, _ := json.Marshal(state)
	_, _ = s.metadata.Put(ctx, metastore.Document{Collection: "fleet-sync", ID: id, State: "desired", Payload: payload, CreatedBy: doc.CreatedBy})
}

func (s *Server) markParserSyncRolloutDesired(ctx context.Context, actor, proposalID string, bundle parserAssetBundle, targets []string) {
	for _, nodeID := range targets {
		s.updateDesiredNodeAssetSync(ctx, nodeID, bundle)
	}
	_, _ = s.metadata.Put(ctx, metastore.Document{
		Collection: "parser-sync-state", ID: "controller", State: "published",
		Payload: mustJSON(map[string]any{
			"desired_hash": bundle.BundleHash, "assets": bundleRevisionMap(bundle),
			"rollout_proposal_id": proposalID, "rollout_targets": targets,
			"rollout_approved_by": actor, "rollout_approved_at": time.Now().UTC(), "status": "rolling_out",
		}), CreatedBy: actor,
	})
}

// recordParserSyncRolloutResult closes the acknowledgement gap between a
// successful client-side atomic swap and the client's next heartbeat. The
// reported hash is never trusted on its own: it must equal the exact immutable
// bundle carried by this rollout. A failure remains isolated to fleet-sync
// diagnostics and does not mutate canonical live telemetry or the active
// controller bundle.
func (s *Server) recordParserSyncRolloutResult(ctx context.Context, rolloutID string, result protocol.CommandResult) {
	s.fleetSyncMu.Lock()
	defer s.fleetSyncMu.Unlock()
	item, ok := s.rollouts.Get(rolloutID)
	if !ok || item == nil || item.Type != rolloutTypeParserSync {
		return
	}
	var payload struct {
		Bundle     parserAssetBundle `json:"asset_bundle"`
		BundleHash string            `json:"bundle_hash"`
	}
	if json.Unmarshal(item.Payload, &payload) != nil {
		return
	}
	expected := strings.TrimSpace(payload.BundleHash)
	if expected == "" {
		expected = strings.TrimSpace(payload.Bundle.BundleHash)
	}
	id := safeID(result.NodeID)
	doc, _ := s.metadata.Get(ctx, "fleet-sync", id, 0)
	state := map[string]any{"node_id": result.NodeID}
	_ = json.Unmarshal(doc.Payload, &state)
	desiredHash := strings.TrimSpace(fmt.Sprint(state["desired_asset_bundle_hash"]))
	if desiredHash == "<nil>" {
		desiredHash = ""
	}
	// A leased command can finish after a newer parser publication has already
	// changed the desired bundle. Prefer the freshly rebuilt published hash, but
	// do not discard a valid client acknowledgement merely because the catalog
	// store is briefly unavailable while the command result is being recorded.
	if current, currentErr := s.buildActiveParserBundle(ctx); currentErr == nil && current.BundleHash != "" {
		desiredHash = current.BundleHash
	}
	if desiredHash == "" {
		desiredHash = expected
	}
	creator := doc.CreatedBy
	if creator == "" {
		creator = "node:" + result.NodeID
	}
	state["asset_result_at"] = time.Now().UTC()
	state["asset_result_rollout_id"] = rolloutID
	if desiredHash == "" || expected != desiredHash {
		state["desired_asset_bundle_hash"] = desiredHash
		state["active_asset_bundle_hash"] = result.ActiveAssetBundleHash
		state["active_asset_revisions"] = result.ActiveAssetRevisions
		if validRuntimeInventory(result.RuntimeInventory) {
			state["runtime_inventory"] = result.RuntimeInventory
		}
		state["asset_sync_status"] = "drifted"
		state["asset_drifted"] = true
		state["asset_sync_error"] = fmt.Sprintf("late parser rollout acknowledgement for bundle %q; current published bundle is %q", expected, desiredHash)
	} else if result.Status != rollout.StatusCompleted {
		state["desired_asset_bundle_hash"] = desiredHash
		state["asset_sync_status"] = "failed"
		state["asset_drifted"] = true
		state["asset_sync_error"] = strings.TrimSpace(result.Message)
	} else if expected == "" || result.ActiveAssetBundleHash != expected || !runtimeInventoryMatchesBundle(result.RuntimeInventory, payload.Bundle) {
		state["desired_asset_bundle_hash"] = desiredHash
		state["asset_sync_status"] = "drifted"
		state["asset_drifted"] = true
		state["asset_sync_error"] = fmt.Sprintf("client command completed without matching parser files and bundle: reported %q; expected %q", result.ActiveAssetBundleHash, expected)
	} else {
		state["desired_asset_bundle_hash"] = desiredHash
		state["active_asset_bundle_hash"] = result.ActiveAssetBundleHash
		state["active_asset_revisions"] = result.ActiveAssetRevisions
		if validRuntimeInventory(result.RuntimeInventory) {
			state["runtime_inventory"] = result.RuntimeInventory
		}
		state["asset_sync_status"] = "applied"
		state["asset_drifted"] = false
		state["asset_sync_error"] = ""
		state["last_asset_acknowledgement_at"] = time.Now().UTC()
	}
	encoded, err := json.Marshal(state)
	if err == nil {
		_, _ = s.metadata.Put(ctx, metastore.Document{Collection: "fleet-sync", ID: id, State: "reported", Payload: encoded, CreatedBy: creator})
	}
}

func (s *Server) handleParserSync(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/parser-sync"), "/")
	if (path == "" || path == "runtime") && r.Method == http.MethodGet {
		if s.authorize(w, r, middleware.PermissionReadFleet) == nil {
			return
		}
		s.listParserSync(w, r)
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "GET, POST")
		jsonError(w, "GET or POST required", http.StatusMethodNotAllowed)
		return
	}
	if path != "verify" && path != "retry" && path != "rollout" {
		jsonError(w, "action must be verify, retry, or rollout", http.StatusNotFound)
		return
	}
	permission := middleware.PermissionReadParsers
	if path == "retry" || path == "rollout" {
		permission = middleware.PermissionTriggerRollout
	}
	principal := s.authorize(w, r, permission)
	if principal == nil {
		return
	}
	if path != "verify" {
		s.parserMutationMu.Lock()
		defer s.parserMutationMu.Unlock()
	}
	var request struct {
		AssetClass string   `json:"asset_class"`
		AssetID    string   `json:"asset_id"`
		NodeIDs    []string `json:"node_ids,omitempty"`
		TargetMode string   `json:"target_mode,omitempty"`
		OriginNode string   `json:"origin_node_id,omitempty"`
		BundleHash string   `json:"bundle_hash,omitempty"`
	}
	if err := decodeJSON(r, &request, 64<<10); err != nil {
		jsonError(w, "invalid parser sync action", http.StatusBadRequest)
		return
	}
	bundle, err := s.buildActiveParserBundle(r.Context())
	if err != nil {
		jsonError(w, "active parser bundle unavailable", http.StatusServiceUnavailable)
		return
	}
	if request.AssetClass != "" || request.AssetID != "" {
		found := false
		for _, asset := range bundle.Assets {
			if asset.AssetClass == request.AssetClass && asset.AssetID == request.AssetID {
				found = true
				break
			}
		}
		if !found {
			jsonError(w, "active asset not found", http.StatusNotFound)
			return
		}
	}
	controllerHash, controllerStatus := "", "not_materialized"
	if path == "verify" {
		if applied, readErr := s.readMaterializedParserBundleHash(); readErr == nil {
			controllerHash = applied
			controllerStatus = "drifted"
			if applied == bundle.BundleHash {
				controllerStatus = "applied"
			}
		} else if !os.IsNotExist(readErr) {
			controllerStatus = "invalid"
		}
	} else {
		var materializeErr error
		controllerHash, materializeErr = s.materializeParserBundle(bundle)
		if materializeErr != nil {
			jsonError(w, "controller parser bundle materialization failed", http.StatusServiceUnavailable)
			return
		}
		controllerStatus = "applied"
	}
	if path == "verify" {
		if request.BundleHash != "" && request.BundleHash != bundle.BundleHash {
			jsonError(w, "requested parser bundle is no longer the current published bundle", http.StatusConflict)
			return
		}
		if len(request.NodeIDs) > parserSyncTargetLimit {
			jsonError(w, fmt.Sprintf("verification exceeds the %d-client target limit", parserSyncTargetLimit), http.StatusBadRequest)
			return
		}
		selected := map[string]bool{}
		for _, id := range request.NodeIDs {
			selected[id] = true
		}
		if len(selected) > 0 {
			for id := range selected {
				if _, ok := s.fleet.Get(id); !ok {
					jsonError(w, "verification target is not an enrolled client: "+id, http.StatusBadRequest)
					return
				}
			}
		}
		states, stateErr := s.listParserSyncDocuments(r.Context(), "fleet-sync", "", parserSyncTargetLimit)
		if stateErr != nil {
			jsonError(w, "client parser acknowledgements are unavailable", http.StatusServiceUnavailable)
			return
		}
		byNode := map[string]map[string]any{}
		for _, doc := range states {
			state := map[string]any{}
			if json.Unmarshal(doc.Payload, &state) == nil {
				byNode[doc.ID] = state
			}
		}
		clients := []map[string]any{}
		minimumKVRelease := parserBundleMinimumKVRelease(bundle)
		for _, node := range s.fleet.All() {
			if len(selected) > 0 && !selected[node.ID] {
				continue
			}
			state := byNode[safeID(node.ID)]
			active := fmt.Sprint(state["active_asset_bundle_hash"])
			desired := fmt.Sprint(state["desired_asset_bundle_hash"])
			acknowledged := parserBundleAcknowledged(state, bundle.BundleHash)
			status := "pending"
			if acknowledged {
				status = "applied"
			} else if active != "<nil>" && active != "" {
				status = "drifted"
			}
			code, next := parserSyncClientBlocker(node, minimumKVRelease, acknowledged, state)
			clients = append(clients, map[string]any{"node_id": node.ID, "expected_hash": bundle.BundleHash, "desired_hash": desired, "acknowledged_hash": active, "applied_hash": active, "status": status, "exact_acknowledgement": acknowledged, "blocking_code": code, "next_action": next, "runtime_inventory": state["runtime_inventory"], "error": state["asset_sync_error"]})
		}
		jsonOK(w, map[string]any{"desired_hash": bundle.BundleHash, "controller_applied_hash": controllerHash, "controller_status": controllerStatus, "client_acknowledgements": clients, "verified_at": time.Now().UTC(), "assets": bundle.Assets})
		return
	}
	if request.BundleHash != "" && request.BundleHash != bundle.BundleHash {
		jsonError(w, "requested parser bundle is no longer the current published bundle", http.StatusConflict)
		return
	}
	if path == "rollout" {
		mode := request.TargetMode
		if mode == "" {
			mode = "all_eligible"
		}
		targets := append([]string(nil), request.NodeIDs...)
		if mode == "origin" {
			if request.OriginNode == "" {
				jsonError(w, "origin_node_id is required for origin rollout", http.StatusBadRequest)
				return
			}
			targets = []string{request.OriginNode}
		}
		proposal, err := s.proposeParserBundleRollout(r.Context(), principal.Username, "Roll out published parser bundle", "Published parser bundles require independent rollout approval before client synchronization.", mode, bundle, targets)
		if err != nil {
			jsonError(w, err.Error(), http.StatusConflict)
			return
		}
		s.auditAction(principal, "parser_sync.rollout.propose", "rollout-proposals/"+proposal.ID, "success", map[string]string{"bundle_hash": bundle.BundleHash, "target_mode": mode})
		jsonCreated(w, map[string]any{"proposal": documentView(proposal), "bundle_hash": bundle.BundleHash, "approval_gate": "pending_approval"})
		return
	}
	targets := append([]string(nil), request.NodeIDs...)
	if len(targets) == 0 {
		for _, node := range s.fleet.All() {
			if node.Status != fleet.StatusOffline {
				targets = append(targets, node.ID)
			}
		}
	}
	if len(targets) == 0 {
		jsonError(w, "no eligible online clients to retry", http.StatusConflict)
		return
	}
	saved, err := s.proposeParserBundleRollout(r.Context(), principal.Username, "Retry parser asset synchronization", "Parser sync retries require an independent rollout approval.", "selected", bundle, targets)
	if err != nil {
		jsonError(w, "parser sync retry proposal could not be saved", http.StatusInternalServerError)
		return
	}
	s.auditAction(principal, "parser_sync.retry.propose", "rollout-proposals/"+saved.ID, "success", map[string]string{"bundle_hash": bundle.BundleHash})
	jsonCreated(w, map[string]any{"proposal": documentView(saved), "bundle_hash": bundle.BundleHash, "approval_gate": "pending_approval"})
}

func (s *Server) handleParserAssetTransition(w http.ResponseWriter, r *http.Request, principal *middleware.SecurityContext, collection, id, action string, doc metastore.Document) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		jsonError(w, "POST required", http.StatusMethodNotAllowed)
		return
	}
	state, payload := doc.State, map[string]any{}
	_ = json.Unmarshal(doc.Payload, &payload)
	switch action {
	case "submit":
		if state != "draft" && state != "rejected" {
			jsonError(w, "only draft or rejected assets may be submitted", http.StatusConflict)
			return
		}
		state = "in_review"
		payload["submitted_by"] = principal.Username
		payload["submitted_at"] = time.Now().UTC()
	case "approve":
		if state != "in_review" {
			jsonError(w, "asset is not awaiting review", http.StatusConflict)
			return
		}
		if !principalCanReviewChange(principal, doc.CreatedBy) {
			jsonError(w, "creator cannot approve own parser asset", http.StatusConflict)
			return
		}
		state = "approved"
		payload["approved_by"] = principal.Username
		payload["approved_at"] = time.Now().UTC()
	case "publish":
		if state != "approved" {
			jsonError(w, "only approved assets can be published", http.StatusConflict)
			return
		}
		state = "active"
		payload["published_by"] = principal.Username
		payload["published_at"] = time.Now().UTC()
	case "revoke":
		if collection != "known-keys" || (state != "active" && state != "revoked") {
			jsonError(w, "only active known-key assets can be revoked", http.StatusConflict)
			return
		}
		if state == "active" {
			state = "revoked"
			payload["revoked_by"] = principal.Username
			payload["revoked_at"] = time.Now().UTC()
		}
	default:
		jsonError(w, "unknown asset transition", http.StatusNotFound)
		return
	}
	payload["review_state"] = state
	encoded, err := json.Marshal(payload)
	if err != nil {
		jsonError(w, "asset transition failed", http.StatusInternalServerError)
		return
	}
	saved := doc
	if doc.State != state {
		saved, err = s.metadata.Put(r.Context(), metastore.Document{Collection: collection, ID: id, State: state, Payload: encoded, CreatedBy: doc.CreatedBy})
		if err != nil {
			jsonError(w, "asset transition persistence failed", http.StatusInternalServerError)
			return
		}
	}
	response := map[string]any{"asset": documentView(saved), "state": state}
	if (action == "publish" || action == "revoke") && isParserAssetCollection(collection) {
		bundle, buildErr := s.buildActiveParserBundle(r.Context())
		if buildErr != nil {
			jsonError(w, "published asset bundle could not be built", http.StatusServiceUnavailable)
			return
		}
		appliedHash, materializeErr := s.materializeParserBundle(bundle)
		if materializeErr != nil {
			jsonError(w, "published asset bundle could not be applied on controller", http.StatusServiceUnavailable)
			return
		}
		now := time.Now().UTC()
		controllerState := map[string]any{"desired_hash": bundle.BundleHash, "applied_hash": appliedHash, "applied_at": now, "status": "controller_applied_pending_fleet_approval", "assets": bundleRevisionMap(bundle)}
		stateBytes, _ := json.Marshal(controllerState)
		_, _ = s.metadata.Put(r.Context(), metastore.Document{Collection: "parser-sync-state", ID: "controller", State: "applied", Payload: stateBytes, CreatedBy: principal.Username})
		proposal, proposalErr := s.proposeParserBundleRollout(r.Context(), principal.Username, "Synchronize parser asset bundle", "Parser asset changes require fleet rollout approval.", "all_eligible", bundle, nil)
		if proposalErr != nil {
			jsonError(w, "asset changed and controller applied, but rollout proposal could not be created: "+proposalErr.Error(), http.StatusServiceUnavailable)
			return
		}
		response["bundle_hash"] = bundle.BundleHash
		response["controller_applied_hash"] = appliedHash
		response["rollout_proposal"] = documentView(proposal)
		response["auto_rollout_status"] = "pending_approval"
	}
	actionLabel, targetPrefix := "config_asset."+action, "config-assets/"
	if isParserAssetCollection(collection) {
		actionLabel, targetPrefix = "parser_asset."+action, "parser-assets/"
	}
	s.auditAction(principal, actionLabel, targetPrefix+collection+"/"+id, "success", map[string]string{"revision": fmt.Sprint(saved.Revision), "state": state})
	jsonOK(w, response)
}

func isParserAssetCollection(collection string) bool {
	for _, candidate := range parserAssetCollections {
		if collection == candidate {
			return true
		}
	}
	return false
}

func (s *Server) listParserSync(w http.ResponseWriter, r *http.Request) {
	started := time.Now().UTC()
	ctx, cancel := context.WithTimeout(r.Context(), metadataViewReadBudget)
	defer cancel()
	bundle, err := s.buildActiveParserBundle(ctx)
	if err != nil {
		replyMetadataUnavailable(w, r, &s.parserSyncView, "parser sync metadata unavailable")
		return
	}
	controllerHash := ""
	controllerStatus := "not_materialized"
	if data, err := os.ReadFile(filepath.Join(s.cfg.StateDir, "parser-assets", "bundle.json")); err == nil {
		var applied parserAssetBundle
		if json.Unmarshal(data, &applied) == nil {
			hash, _, hashErr := bundleDigest(applied)
			if hashErr == nil && hash == bundle.BundleHash {
				controllerHash = hash
				controllerStatus = "applied"
			} else {
				controllerStatus = "drifted"
			}
		}
	}
	page, pageErr := s.listParserSyncDocuments(ctx, "fleet-sync", "", parserSyncTargetLimit)
	if pageErr != nil {
		replyMetadataUnavailable(w, r, &s.parserSyncView, "client parser acknowledgement telemetry unavailable")
		return
	}
	byNode := map[string]map[string]any{}
	for _, doc := range page {
		var state map[string]any
		if json.Unmarshal(doc.Payload, &state) == nil {
			byNode[doc.ID] = state
		}
	}
	clients := []map[string]any{}
	revisions := bundleRevisionMap(bundle)
	minimumKVRelease := parserBundleMinimumKVRelease(bundle)
	totalClients, onlineClients, exactClients := 0, 0, 0
	pendingClients, driftedClients, failedClients, offlineClients := 0, 0, 0, 0
	for _, node := range s.fleet.All() {
		state := byNode[safeID(node.ID)]
		if state == nil {
			state = map[string]any{"node_id": node.ID}
		}
		applied := fmt.Sprint(state["active_asset_bundle_hash"])
		desired := bundle.BundleHash
		drift := applied != desired
		acknowledged := parserBundleAcknowledged(state, desired)
		status := strings.TrimSpace(fmt.Sprint(state["asset_sync_status"]))
		if status == "" || status == "<nil>" {
			status = "pending"
		}
		totalClients++
		if node.Status == fleet.StatusOffline {
			offlineClients++
		} else {
			onlineClients++
		}
		if acknowledged {
			exactClients++
		} else if status == "failed" {
			failedClients++
		} else if drift {
			driftedClients++
		} else {
			pendingClients++
		}
		code, next := parserSyncClientBlocker(node, minimumKVRelease, acknowledged, state)
		clients = append(clients, map[string]any{"blocking_code": code, "next_action": next, "runtime_inventory": state["runtime_inventory"], "node_id": node.ID, "hostname": node.Hostname, "desired_revision": revisions, "desired_hash": desired, "applied_revision": state["active_asset_revisions"], "applied_hash": applied, "acknowledged_bundle_hash": applied, "exact_acknowledgement": acknowledged, "last_seen": state["last_seen"], "status": status, "drifted": drift, "error": state["asset_sync_error"]})
	}
	sort.Slice(clients, func(i, j int) bool { return fmt.Sprint(clients[i]["node_id"]) < fmt.Sprint(clients[j]["node_id"]) })
	var latest *rolloutRecord
	for _, item := range s.rollouts.List() {
		if item.Type != rolloutTypeParserSync {
			continue
		}
		var payload map[string]any
		if json.Unmarshal(item.Payload, &payload) == nil && fmt.Sprint(payload["bundle_hash"]) == bundle.BundleHash {
			latest = item
			break
		}
	}
	var pendingProposal *metastore.Document
	proposals, err := s.listParserSyncDocuments(ctx, "rollout-proposals", "pending_approval", parserSyncTargetLimit)
	if err != nil {
		replyMetadataUnavailable(w, r, &s.parserSyncView, "parser sync rollout approvals unavailable")
		return
	}
	for i := range proposals {
		var proposal map[string]any
		if json.Unmarshal(proposals[i].Payload, &proposal) == nil && fmt.Sprint(proposal["type"]) == rolloutTypeParserSync && fmt.Sprint(proposal["bundle_hash"]) == bundle.BundleHash {
			pendingProposal = &proposals[i]
			break
		}
	}
	rolloutState := rolloutStatus(latest)
	if latest == nil && pendingProposal != nil {
		rolloutState = "pending_approval"
	}
	var proposalView any
	if pendingProposal != nil {
		proposalView = documentView(*pendingProposal)
	}
	assetViews := []map[string]any{}
	for _, asset := range bundle.Assets {
		var controllerRevision any
		if controllerStatus == "applied" {
			controllerRevision = asset.Revision
		}
		desired := map[string]any{"revision": asset.Revision, "sha256": asset.SHA256, "state": "published"}
		controller := map[string]any{"revision": controllerRevision, "sha256": controllerHash, "status": controllerStatus}
		assetViews = append(assetViews, map[string]any{"asset_class": asset.AssetClass, "asset_id": asset.AssetID, "name": asset.Name, "desired": desired, "controller": controller, "desired_revision": asset.Revision, "desired_hash": asset.SHA256, "state": "published", "controller_applied_revision": controllerRevision, "controller_applied_hash": controllerHash, "controller_status": controllerStatus, "auto_rollout": map[string]any{"enabled": false, "status": rolloutState}, "rollout": rolloutSummary(latest), "rollout_proposal": proposalView})
	}
	acknowledgementPct := 0.0
	if totalClients > 0 {
		acknowledgementPct = float64(exactClients) * 100 / float64(totalClients)
	}
	telemetry := map[string]any{
		"total_clients": totalClients, "online_clients": onlineClients, "offline_clients": offlineClients,
		"exact_acknowledged_clients": exactClients, "pending_clients": pendingClients,
		"drifted_clients": driftedClients, "failed_clients": failedClients,
		"acknowledgement_pct": acknowledgementPct, "desired_bundle_hash": bundle.BundleHash,
		"controller_status": controllerStatus, "target_limit": parserSyncTargetLimit, "generated_at": time.Now().UTC(),
	}
	replyMetadataCurrent(w, &s.parserSyncView, started, map[string]any{"controller": s.controllerParserInventory(bundle), "assets": assetViews, "count": len(assetViews), "bundle": map[string]any{"revision": revisions, "desired_hash": bundle.BundleHash, "controller_applied_hash": controllerHash, "controller_status": controllerStatus}, "clients": clients, "telemetry": telemetry, "rollout": rolloutSummary(latest), "rollout_proposal": proposalView, "automatic_rollout_policy": "approval_gated_parser_bundle_sync"})
}

func bundleRevisionMap(bundle parserAssetBundle) map[string]int {
	out := map[string]int{}
	for _, asset := range bundle.Assets {
		out[asset.AssetClass+"/"+asset.AssetID] = asset.Revision
	}
	return out
}
func rolloutStatus(item *rolloutRecord) string {
	if item == nil {
		return "not_started"
	}
	return item.Status
}
func rolloutSummary(item *rolloutRecord) any {
	if item == nil {
		return map[string]any{"status": "not_started"}
	}
	return map[string]any{"id": item.ID, "status": item.Status, "total_nodes": item.TotalNodes, "done_nodes": item.DoneNodes, "failed_nodes": item.FailedNodes, "target_nodes": item.TargetNodes, "node_results": item.NodeResults}
}

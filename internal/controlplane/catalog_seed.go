package controlplane

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"

	parsercatalog "nexora.local/nli-v20/configs/catalog"
	"nexora.local/nli-v20/internal/canonicaljson"
	metastore "nexora.local/nli-v20/internal/store"
)

var catalogCollections = map[string]string{
	"kv": "kv-config", "regex": "regex-patterns", "known-keys": "known-keys",
	"static-patterns": "static-patterns", "templates": "templates", "projections": "projections",
	"semantic-extraction": "semantic-extraction",
}

func builtinCatalogPayload(asset parsercatalog.Asset) ([]byte, error) {
	return json.Marshal(map[string]any{
		"name": asset.Name, "config": asset.Config,
		"review_state": "active", "approved_by": "nli-v21-release",
		"published_by": "nli-v21-release",
	})
}

// Only the original release assets may be delivered on enrollment without
// a new operator approval. Compare their content against the binary, since
// catalog IDs and approval labels alone do not establish provenance.
func isBuiltinEnrollmentBundle(bundle parserAssetBundle) bool {
	embedded, err := parsercatalog.Assets()
	if err != nil || len(embedded) != len(bundle.Assets) {
		return false
	}
	type assetKey struct{ collection, id string }
	byKey := make(map[assetKey]parserBundleAsset, len(bundle.Assets))
	for _, asset := range bundle.Assets {
		byKey[assetKey{asset.AssetClass, asset.AssetID}] = asset
	}
	for _, asset := range embedded {
		published := byKey[assetKey{catalogCollections[asset.Collection], asset.ID}]
		if published.Revision != 1 {
			return false
		}
		payload, err := builtinCatalogPayload(asset)
		if err != nil {
			return false
		}
		canonicalPayload, err := canonicaljson.Normalize(payload)
		if err != nil || !bytes.Equal(canonicalPayload, published.Payload) {
			return false
		}
	}
	return true
}

func seedBuiltinCatalog(ctx context.Context, store metastore.Store) error {
	assets, err := parsercatalog.Assets()
	if err != nil {
		return err
	}
	for _, asset := range assets {
		collection, ok := catalogCollections[asset.Collection]
		if !ok || safeID(asset.ID) == "" || asset.Name == "" || len(asset.Config) == 0 {
			return errors.New("embedded parser catalog has an invalid asset")
		}
		if _, err := store.Get(ctx, collection, asset.ID, 0); err == nil {
			continue // Preserve user revisions and edits across restarts.
		}
		payload, err := builtinCatalogPayload(asset)
		if err != nil {
			return err
		}
		// The embedded catalog is part of the tested, signed release. Treat it as
		// pre-approved so a fresh controller immediately has a deployable baseline.
		if _, err := store.Put(ctx, metastore.Document{Collection: collection, ID: asset.ID, State: "active", Payload: payload, CreatedBy: "nli-v21-release"}); err != nil {
			return err
		}
	}
	return nil
}

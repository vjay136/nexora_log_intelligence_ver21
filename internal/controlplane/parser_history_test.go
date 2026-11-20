package controlplane

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestControllerBundleHistorySurvivesEmptyGeneration(t *testing.T) {
	f := newAPIFixture(t)
	prior, err := f.server.buildActiveParserBundle(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.server.materializeParserBundle(prior); err != nil {
		t.Fatal(err)
	}
	empty := parserAssetBundle{SchemaVersion: 1, Assets: []parserBundleAsset{}}
	empty.BundleHash, _, err = bundleDigest(empty)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.server.materializeParserBundle(empty); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(f.server.cfg.StateDir, "parser-assets-history", prior.BundleHash+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	hash, err := verifyHistoricalParserBundle(data)
	if err != nil || hash != prior.BundleHash {
		t.Fatalf("prior archive %s: %v", hash, err)
	}
	actual, err := f.server.readMaterializedParserBundleHash()
	if err != nil || actual != empty.BundleHash {
		t.Fatalf("approved empty generation not active: %s %v", actual, err)
	}
	// A broken archive store prevents the next replacement and preserves the
	// current generation rather than silently losing recovery evidence.
	if err := os.WriteFile(path, []byte(`{}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = f.server.materializeParserBundle(prior); err == nil {
		t.Fatal("tampered history did not abort materialization")
	}
	actual, err = f.server.readMaterializedParserBundleHash()
	if err != nil || actual != empty.BundleHash {
		t.Fatal("failed archive replaced active bundle")
	}
}

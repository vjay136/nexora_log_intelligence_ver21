package controlclient

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestClientBundleHistorySurvivesEmptyGeneration(t *testing.T) {
	root := t.TempDir()
	client, err := New(Config{ControllerURL: "http://127.0.0.1:12345", AllowInsecureHTTP: true, NodeTokenFile: filepath.Join(root, "node.token"), InstallIDFile: filepath.Join(root, "install.id"), JournalFile: filepath.Join(root, "journal.json"), AgentConfigPath: filepath.Join(root, "nli.json"), ParserDir: filepath.Join(root, "parsers"), AssetDir: filepath.Join(root, "parser-assets"), PackageSpoolDir: filepath.Join(root, "packages"), PollSeconds: 5}, &fakeExecutor{})
	if err != nil {
		t.Fatal(err)
	}
	payload := json.RawMessage(`{"name":"customer rule","config":{"pattern":"^customer$"}}`)
	digest := sha256.Sum256(payload)
	prior := synchronizedAssetBundle{SchemaVersion: 1, Assets: []synchronizedAsset{{AssetClass: "regex-patterns", AssetID: "customer", Revision: 4, SHA256: fmt.Sprintf("%x", digest), Payload: payload}}}
	// Use canonical payload hashing exactly as publication does.
	var canonical map[string]any
	_ = json.Unmarshal(payload, &canonical)
	encoded, _ := json.Marshal(canonical)
	sum := sha256.Sum256(encoded)
	prior.Assets[0].SHA256 = fmt.Sprintf("%x", sum)
	prior.BundleHash, err = synchronizedBundleHash(prior)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(prior)
	if _, err = client.applyAssetBundle(context.Background(), data); err != nil {
		t.Fatal(err)
	}
	empty := synchronizedAssetBundle{SchemaVersion: 1, Assets: []synchronizedAsset{}}
	empty.BundleHash, err = synchronizedBundleHash(empty)
	if err != nil {
		t.Fatal(err)
	}
	next, _ := json.Marshal(empty)
	if _, err = client.applyAssetBundle(context.Background(), next); err != nil {
		t.Fatal(err)
	}
	archive := filepath.Join(root, "parser-assets-history", prior.BundleHash+".json")
	saved, err := os.ReadFile(archive)
	if err != nil {
		t.Fatal(err)
	}
	if hash, err := verifyHistoricalAssetBundle(saved); err != nil || hash != prior.BundleHash {
		t.Fatalf("archive %s: %v", hash, err)
	}
	if _, err := os.Stat(filepath.Join(client.cfg.AssetDir, "regex", "customer.json")); !os.IsNotExist(err) {
		t.Fatal("approved deletion was blocked")
	}
	if err := os.WriteFile(archive, []byte(`{}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = client.applyAssetBundle(context.Background(), data); err == nil {
		t.Fatal("tampered history did not abort swap")
	}
	current, _ := os.ReadFile(filepath.Join(client.cfg.AssetDir, "bundle.json"))
	if string(current) != string(next) {
		t.Fatal("failed archive changed current bundle")
	}
}

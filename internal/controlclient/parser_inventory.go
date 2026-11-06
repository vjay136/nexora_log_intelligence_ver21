package controlclient

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"

	"nexora.local/nli-v20/internal/protocol"
)

func (c *Client) runtimeParserInventory() *protocol.ParserRuntimeInventory {
	bundlePath := filepath.Join(c.cfg.AssetDir, "bundle.json")
	hash, _ := activeAssetBundleStatus(bundlePath)
	inventory := &protocol.ParserRuntimeInventory{ParserDirectory: c.cfg.ParserDir, ManagedManifestPath: filepath.Join(c.cfg.ParserDir, managedParserManifest), BundlePath: bundlePath, InventoryPath: filepath.Join(c.cfg.AssetDir, "runtime-inventory.json"), BundleHash: hash, Files: []protocol.ParserRuntimeFile{}}
	managed := map[string]bool{}
	// Read only the manifest; it cannot make an unvalidated file executable.
	if data, err := os.ReadFile(inventory.ManagedManifestPath); err == nil {
		var ids []string
		if json.Unmarshal(data, &ids) == nil {
			for _, id := range ids {
				managed[id] = true
			}
		}
	}
	if len(managed) == 0 {
		if ids, err := managedParserIDsFromPreviousBundle(c.cfg.AssetDir); err == nil {
			for _, id := range ids {
				managed[id] = true
			}
		}
	}
	managedDigests := []protocol.ParserFileDigest{}
	entries, _ := os.ReadDir(c.cfg.ParserDir)
	for _, entry := range entries {

		name := entry.Name()
		if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 || filepath.Ext(name) != ".json" || name == managedParserManifest {
			continue
		}
		inventory.TotalFiles++
		id := name[:len(name)-5]
		path := filepath.Join(c.cfg.ParserDir, name)
		digest := fileSHA256(path)
		if digest == "" {
			continue
		}
		if managed[id] {
			managedDigests = append(managedDigests, protocol.ParserFileDigest{ID: id, SHA256: digest})
		}
		if len(inventory.Files) >= 128 {
			inventory.Truncated = true
			continue
		}
		inventory.Files = append(inventory.Files, protocol.ParserRuntimeFile{ID: id, Path: path, SHA256: digest, ControllerManaged: managed[id]})
	}
	inventory.ManagedFilesDigest = protocol.ManagedParserFilesDigest(managedDigests)
	sort.Slice(inventory.Files, func(i, j int) bool { return inventory.Files[i].ID < inventory.Files[j].ID })
	if data, err := json.MarshalIndent(inventory, "", "  "); err == nil {
		if os.MkdirAll(c.cfg.AssetDir, 0750) == nil {
			_ = os.WriteFile(inventory.InventoryPath, data, 0640)
		}
	}
	return inventory
}

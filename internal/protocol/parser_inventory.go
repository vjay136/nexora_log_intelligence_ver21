package protocol

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
)

type ParserFileDigest struct {
	ID     string `json:"id"`
	SHA256 string `json:"sha256"`
}

// ManagedParserFilesDigest has a shared stable encoding for controller desired
// bytes and client on-disk bytes, independent of paths and descriptor limits.
func ManagedParserFilesDigest(files []ParserFileDigest) string {
	ordered := append([]ParserFileDigest{}, files...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].ID < ordered[j].ID })
	data, _ := json.Marshal(ordered)
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}

// Package recovery creates encrypted, bounded controller identity capsules.
package recovery

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

const MaxBytes = 32 << 20
const iterations = 600000

var stateFiles = []string{"users.json", "fleet.json", "rollouts.json", "audit.key", "csrf.key", "assurance.key", "consumed-enrollment-tokens.sha256", "audit.jsonl", "tls/ca.key", "tls/ca.crt", "tls/server.key", "tls/server.crt", "tls/discovered-server.key", "tls/discovered-server.crt", "tls/corporate-server.key", "tls/corporate-server.crt", "tls/corporate-ca.crt", "quarantine-metadata/quarantine.key"}

type File struct {
	Mode uint32 `json:"mode"`
	Root string `json:"root"`
	Path string `json:"path"`
	Data []byte `json:"data"`
}
type Contents struct {
	StateDir  string    `json:"state_dir"`
	ConfigDir string    `json:"config_dir"`
	Schema    int       `json:"schema_version"`
	CreatedAt time.Time `json:"created_at"`
	Files     []File    `json:"files"`
}
type Envelope struct {
	Schema     int    `json:"schema_version"`
	KDF        string `json:"kdf"`
	Iterations int    `json:"iterations"`
	Salt       string `json:"salt"`
	Nonce      string `json:"nonce"`
	Ciphertext string `json:"ciphertext"`
}

func allowed(file File) bool {
	if file.Root == "state" {
		return slices.Contains(stateFiles, file.Path)
	}
	if (file.Root != "config" && file.Root != "metadata") || filepath.IsAbs(file.Path) || strings.Contains(file.Path, "\\") || strings.Contains(file.Path, ":") || strings.ContainsRune(file.Path, 0) || filepath.ToSlash(filepath.Clean(file.Path)) != file.Path || file.Path == "." || strings.HasPrefix(file.Path, "../") {
		return false
	}
	if file.Root == "metadata" {
		parts := strings.Split(file.Path, "/")
		return len(parts) >= 2 && !strings.HasPrefix(parts[0], "recovery-") && strings.HasSuffix(file.Path, ".json")
	}
	return true
}
func Capture(state, config string) (Contents, error) {
	out := Contents{Schema: 1, StateDir: state, ConfigDir: config, CreatedAt: time.Now().UTC(), Files: []File{}}
	total := 0
	add := func(root, base, name string) error {
		path := filepath.Join(base, filepath.FromSlash(name))
		info, err := os.Lstat(path)
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() || info.Size() > MaxBytes {
			return fmt.Errorf("unsupported recovery file %s", name)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		total += len(data)
		if total > MaxBytes {
			return errors.New("controller recovery contents exceed 32 MiB")
		}
		out.Files = append(out.Files, File{Root: root, Path: name, Data: data, Mode: uint32(info.Mode().Perm())})
		return nil
	}
	for _, name := range stateFiles {
		if err := add("state", state, name); err != nil {
			return out, err
		}
	}
	metadata := filepath.Join(state, "metadata")
	if err := filepath.WalkDir(metadata, func(path string, item os.DirEntry, walkErr error) error {
		if walkErr != nil {
			if errors.Is(walkErr, os.ErrNotExist) && path == metadata {
				return nil
			}
			return walkErr
		}
		if item.Type()&os.ModeSymlink != 0 {
			return errors.New("symlink in local metadata backup")
		}
		name, err := filepath.Rel(metadata, path)
		if err != nil {
			return err
		}
		name = filepath.ToSlash(name)
		if item.IsDir() {
			if path != metadata && strings.HasPrefix(item.Name(), "recovery-") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(name, ".json") {
			return nil
		}
		return add("metadata", metadata, name)
	}); err != nil {
		return out, err
	}
	if config != "" {
		err := filepath.WalkDir(config, func(path string, item os.DirEntry, walkErr error) error {
			if walkErr != nil {
				if errors.Is(walkErr, os.ErrNotExist) && path == config {
					return nil
				}
				return walkErr
			}
			if item.Type()&os.ModeSymlink != 0 {
				return fmt.Errorf("symlink in recovery config: %s", item.Name())
			}
			if item.IsDir() {
				return nil
			}
			name, err := filepath.Rel(config, path)
			if err != nil {
				return err
			}
			return add("config", config, filepath.ToSlash(name))
		})
		if err != nil {
			return out, err
		}
	}
	if !slices.ContainsFunc(out.Files, func(file File) bool { return file.Root == "state" && file.Path == "users.json" }) {
		return out, errors.New("controller user identity is unavailable")
	}
	return out, nil
}
func Seal(contents Contents, passphrase string) ([]byte, error) {
	if len(passphrase) < 16 || len(passphrase) > 1024 {
		return nil, errors.New("recovery passphrase must contain 16 to 1024 bytes")
	}
	plain, err := json.Marshal(contents)
	if err != nil || len(plain) > MaxBytes*2 {
		return nil, errors.New("recovery contents exceed bounds")
	}
	salt := make([]byte, 32)
	nonce := make([]byte, 12)
	if _, err = rand.Read(salt); err != nil {
		return nil, err
	}
	if _, err = rand.Read(nonce); err != nil {
		return nil, err
	}
	key, err := pbkdf2.Key(sha256.New, passphrase, salt, iterations, 32)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	encrypted := gcm.Seal(nil, nonce, plain, []byte("nli-controller-recovery-v1"))
	return json.Marshal(Envelope{Schema: 1, KDF: "pbkdf2-sha256", Iterations: iterations, Salt: base64.StdEncoding.EncodeToString(salt), Nonce: base64.StdEncoding.EncodeToString(nonce), Ciphertext: base64.StdEncoding.EncodeToString(encrypted)})
}
func Open(encoded []byte, passphrase string) (Contents, error) {
	var out Contents
	var envelope Envelope
	if len(encoded) > MaxBytes*3 || json.Unmarshal(encoded, &envelope) != nil || envelope.Schema != 1 || envelope.KDF != "pbkdf2-sha256" || envelope.Iterations != iterations {
		return out, errors.New("invalid recovery capsule format")
	}
	if len(passphrase) < 16 || len(passphrase) > 1024 {
		return out, errors.New("invalid recovery passphrase")
	}
	salt, e1 := base64.StdEncoding.DecodeString(envelope.Salt)
	nonce, e2 := base64.StdEncoding.DecodeString(envelope.Nonce)
	encrypted, e3 := base64.StdEncoding.DecodeString(envelope.Ciphertext)
	if e1 != nil || e2 != nil || e3 != nil || len(salt) != 32 || len(nonce) != 12 {
		return out, errors.New("invalid recovery capsule encoding")
	}
	key, err := pbkdf2.Key(sha256.New, passphrase, salt, iterations, 32)
	if err != nil {
		return out, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return out, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return out, err
	}
	plain, err := gcm.Open(nil, nonce, encrypted, []byte("nli-controller-recovery-v1"))
	if err != nil {
		return out, errors.New("recovery passphrase or capsule authentication is invalid")
	}
	if json.Unmarshal(plain, &out) != nil || out.Schema != 1 || len(out.Files) > 10000 {
		return out, errors.New("invalid recovery contents")
	}
	total := 0
	seen := map[string]bool{}
	for _, file := range out.Files {
		key := file.Root + "/" + file.Path
		if !allowed(file) || seen[key] || len(file.Data) > MaxBytes || file.Mode & ^uint32(0777) != 0 {
			return out, errors.New("invalid or duplicate recovery file")
		}
		seen[key] = true
		total += len(file.Data)
	}
	if total > MaxBytes || !seen["state/users.json"] {
		return out, errors.New("invalid recovery file inventory")
	}
	return out, nil
}

// Restore validates all contents before writing. Each existing target is saved
// for rollback. It is an offline operation that the caller must run while the
// controller service is stopped.
func Restore(contents Contents, state, config string) error {
	return RestoreOwned(contents, state, config, -1, -1)
}
func RestoreOwned(contents Contents, state, config string, uid, gid int) error {
	var err error
	state, err = filepath.Abs(state)
	if err != nil {
		return err
	}
	config, err = filepath.Abs(config)
	if err != nil {
		return err
	}
	if state == filepath.Dir(state) || config == filepath.Dir(config) || state == config {
		return errors.New("distinct controller state and configuration directories are required")
	}
	if len(contents.Files) > 10000 {
		return errors.New("recovery file count exceeds limit")
	}
	seen := map[string]bool{}
	total := 0
	for _, file := range contents.Files {
		key := file.Root + "/" + file.Path
		total += len(file.Data)
		if !allowed(file) || seen[key] || file.Mode & ^uint32(0777) != 0 || total > MaxBytes {
			return errors.New("invalid recovery file inventory")
		}
		seen[key] = true
	}
	if !seen["state/users.json"] {
		return errors.New("controller user identity is missing")
	}

	type change struct {
		target  string
		old     []byte
		existed bool
		mode    os.FileMode
	}
	changes := []change{}
	rollback := func() {
		for i := len(changes) - 1; i >= 0; i-- {
			item := changes[i]
			if item.existed {
				_ = atomicFile(item.target, item.old)
				_ = os.Chmod(item.target, item.mode)
			} else {
				_ = os.Remove(item.target)
			}
		}
	}
	for _, file := range contents.Files {
		if !allowed(file) {
			return errors.New("unsupported recovery path")
		}
		base := state
		if file.Root == "metadata" {
			base = filepath.Join(state, "metadata")
		}
		if file.Root == "config" {
			base = config
		}
		if base == "" {
			return errors.New("restore roots required")
		}
		target := filepath.Join(base, filepath.FromSlash(file.Path))
		// Reject links in every existing parent, including the target itself.
		for candidate := target; ; candidate = filepath.Dir(candidate) {
			info, err := os.Lstat(candidate)
			if err == nil && info.Mode()&os.ModeSymlink != 0 {
				return errors.New("restore path includes a symlink")
			}
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				return err
			}
			parent := filepath.Dir(candidate)
			boundary := state
			if file.Root == "config" {
				boundary = config
			}
			if candidate == boundary || candidate == parent {
				break
			}
		}
	}
	for _, file := range contents.Files {
		base := state
		if file.Root == "metadata" {
			base = filepath.Join(state, "metadata")
		}
		if file.Root == "config" {
			base = config
		}
		target := filepath.Join(base, filepath.FromSlash(file.Path))
		old, err := os.ReadFile(target)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			rollback()
			return err
		}
		mode := os.FileMode(0600)
		if info, statErr := os.Stat(target); statErr == nil {
			mode = info.Mode().Perm()
		}
		changes = append(changes, change{target: target, old: old, existed: err == nil, mode: mode})
		if err = atomicFile(target, file.Data); err == nil {
			mode := os.FileMode(file.Mode)
			if mode == 0 {
				mode = 0600
			}
			err = os.Chmod(target, mode)
			if err == nil && uid >= 0 {
				owner, group := uid, gid
				if file.Root == "config" || strings.HasPrefix(file.Path, "tls/") {
					owner = 0
				}
				if file.Root == "state" && file.Path == "tls/ca.key" {
					group = 0
				}
				err = os.Chown(target, owner, group)
			}
		}
		if err != nil {
			rollback()
			return err
		}
	}
	if uid >= 0 {
		for _, file := range contents.Files {
			base := state
			if file.Root == "metadata" {
				base = filepath.Join(state, "metadata")
			}
			owner := uid
			if file.Root == "config" {
				base = config
				owner = 0
			}
			for dir := filepath.Dir(filepath.Join(base, filepath.FromSlash(file.Path))); ; dir = filepath.Dir(dir) {
				if err := os.Chown(dir, owner, gid); err != nil {
					return err
				}
				if err := os.Chmod(dir, 0750); err != nil {
					return err
				}
				if dir == filepath.Clean(base) {
					break
				}
			}
		}
	}
	return nil
}
func atomicFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".nli-recovery-*")
	if err != nil {
		return err
	}
	temp := file.Name()
	defer os.Remove(temp)
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
	return os.Rename(temp, path)
}
func ReadCapsule(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return io.ReadAll(io.LimitReader(file, MaxBytes*3+1))
}

func ValidateEnvelope(encoded []byte) error {
	var envelope Envelope
	if len(encoded) > MaxBytes*3 || json.Unmarshal(encoded, &envelope) != nil || envelope.Schema != 1 || envelope.KDF != "pbkdf2-sha256" || envelope.Iterations != iterations {
		return errors.New("invalid encrypted recovery capsule")
	}
	salt, e1 := base64.StdEncoding.DecodeString(envelope.Salt)
	nonce, e2 := base64.StdEncoding.DecodeString(envelope.Nonce)
	ciphertext, e3 := base64.StdEncoding.DecodeString(envelope.Ciphertext)
	if e1 != nil || e2 != nil || e3 != nil || len(salt) != 32 || len(nonce) != 12 || len(ciphertext) < 16 {
		return errors.New("invalid encrypted recovery capsule encoding")
	}
	return nil
}

// WriteCapsule writes a new encrypted export with owner-only permissions.
func WriteCapsule(path string, data []byte) error {
	if err := ValidateEnvelope(data); err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	if _, err = file.Write(data); err == nil {
		err = file.Sync()
	}
	closeErr := file.Close()
	if err != nil {
		return err
	}
	return closeErr
}

package aiprovider

import (
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const maxSecretBytes = 16 * 1024

type SecretResolver struct {
	AEAD       cipher.AEAD
	SecretRoot string
	Getenv     func(string) string
}

func SealInline(aead cipher.AEAD, providerID, plaintext string) (Credential, error) {
	if aead == nil {
		return Credential{}, errors.New("credential encryption is not configured")
	}
	if !idPattern.MatchString(providerID) {
		return Credential{}, errors.New("valid provider id is required before sealing a credential")
	}
	if err := validatePlainSecret(plaintext); err != nil {
		return Credential{}, err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return Credential{}, fmt.Errorf("generate credential nonce: %w", err)
	}
	ciphertext := aead.Seal(nil, nonce, []byte(plaintext), credentialAAD(providerID))
	return Credential{
		Source:     CredentialEncryptedInline,
		Nonce:      base64.RawStdEncoding.EncodeToString(nonce),
		Ciphertext: base64.RawStdEncoding.EncodeToString(ciphertext),
		Last4:      secretLast4(plaintext),
	}, nil
}

func (r SecretResolver) Resolve(p Profile) (string, error) {
	p = Normalize(p)
	switch p.Credential.Source {
	case CredentialEncryptedInline:
		if r.AEAD == nil {
			return "", errors.New("credential encryption is not configured")
		}
		nonce, err := base64.RawStdEncoding.DecodeString(p.Credential.Nonce)
		if err != nil || len(nonce) != r.AEAD.NonceSize() {
			return "", errors.New("encrypted credential nonce is invalid")
		}
		ciphertext, err := base64.RawStdEncoding.DecodeString(p.Credential.Ciphertext)
		if err != nil {
			return "", errors.New("encrypted credential payload is invalid")
		}
		plain, err := r.AEAD.Open(nil, nonce, ciphertext, credentialAAD(p.ID))
		if err != nil {
			return "", errors.New("encrypted credential authentication failed")
		}
		secret := string(plain)
		if err := validatePlainSecret(secret); err != nil {
			return "", errors.New("decrypted credential is invalid")
		}
		return secret, nil
	case CredentialEnvironment:
		getenv := r.Getenv
		if getenv == nil {
			getenv = os.Getenv
		}
		secret := getenv(p.Credential.Reference)
		if err := validatePlainSecret(secret); err != nil {
			return "", fmt.Errorf("credential environment reference is unavailable: %w", err)
		}
		return secret, nil
	case CredentialFile:
		path, err := secureSecretPath(r.SecretRoot, p.Credential.Reference)
		if err != nil {
			return "", err
		}
		info, err := os.Lstat(path)
		if err != nil {
			return "", fmt.Errorf("credential file is unavailable: %w", err)
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return "", errors.New("credential file must be a regular non-symlink file")
		}
		if info.Mode().Perm()&0o077 != 0 {
			return "", errors.New("credential file permissions must not grant group or other access")
		}
		realRoot, rootErr := filepath.EvalSymlinks(filepath.Clean(r.SecretRoot))
		realPath, pathErr := filepath.EvalSymlinks(path)
		if rootErr != nil || pathErr != nil {
			return "", errors.New("credential file canonical path cannot be verified")
		}
		rel, relErr := filepath.Rel(realRoot, realPath)
		if relErr != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return "", errors.New("credential file resolves outside the configured secret root")
		}
		f, err := os.Open(path)
		if err != nil {
			return "", fmt.Errorf("open credential file: %w", err)
		}
		defer f.Close()
		data, err := io.ReadAll(io.LimitReader(f, maxSecretBytes+1))
		if err != nil {
			return "", fmt.Errorf("read credential file: %w", err)
		}
		if len(data) > maxSecretBytes {
			return "", errors.New("credential file exceeds 16384 bytes")
		}
		secret := strings.TrimSpace(string(data))
		if err := validatePlainSecret(secret); err != nil {
			return "", fmt.Errorf("credential file content is invalid: %w", err)
		}
		return secret, nil
	case CredentialNone:
		if p.Kind != KindOllama {
			return "", errors.New("credential-free access is allowed only for Ollama")
		}
		return "", nil
	default:
		return "", errors.New("unsupported credential source")
	}
}

func credentialAAD(providerID string) []byte {
	return []byte("nli/ai-provider/credential/v1/" + providerID)
}

func validatePlainSecret(secret string) error {
	if len(secret) < 8 {
		return errors.New("credential must contain at least 8 bytes")
	}
	if len(secret) > maxSecretBytes {
		return errors.New("credential exceeds 16384 bytes")
	}
	if strings.ContainsAny(secret, "\r\n\x00") {
		return errors.New("credential contains forbidden control characters")
	}
	return nil
}

func secretLast4(secret string) string {
	if len(secret) < 8 {
		return ""
	}
	return secret[len(secret)-4:]
}

func secureSecretPath(root, reference string) (string, error) {
	if strings.TrimSpace(root) == "" {
		return "", errors.New("secret root is not configured")
	}
	if strings.TrimSpace(reference) == "" || strings.ContainsRune(reference, '\x00') {
		return "", errors.New("credential file reference is invalid")
	}
	cleanRoot, err := filepath.Abs(filepath.Clean(root))
	if err != nil {
		return "", errors.New("secret root is invalid")
	}
	var candidate string
	if filepath.IsAbs(reference) {
		candidate = filepath.Clean(reference)
	} else {
		candidate = filepath.Join(cleanRoot, reference)
	}
	absCandidate, err := filepath.Abs(candidate)
	if err != nil {
		return "", errors.New("credential file reference is invalid")
	}
	rel, err := filepath.Rel(cleanRoot, absCandidate)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", errors.New("credential file must be below the configured secret root")
	}
	return absCandidate, nil
}

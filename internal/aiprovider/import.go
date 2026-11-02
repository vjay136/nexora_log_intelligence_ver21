package aiprovider

import (
	"bytes"
	"crypto/cipher"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

const DefaultMaxImportBytes int64 = 64 * 1024

type importCredential struct {
	Source    CredentialSource `json:"source"`
	Reference string           `json:"reference,omitempty"`
	Secret    string           `json:"secret,omitempty"`
}

type importWire struct {
	ID               string            `json:"id"`
	Name             string            `json:"name"`
	Kind             Kind              `json:"kind"`
	BaseURL          string            `json:"base_url"`
	Model            string            `json:"model"`
	Enabled          bool              `json:"enabled"`
	Priority         int               `json:"priority"`
	FallbackGroup    string            `json:"fallback_group,omitempty"`
	TimeoutSeconds   int               `json:"timeout_seconds,omitempty"`
	MaxResponseBytes int64             `json:"max_response_bytes,omitempty"`
	Headers          map[string]string `json:"headers,omitempty"`
	Credential       importCredential  `json:"credential"`
}

// PendingImport keeps a plaintext imported secret private and transient. It
// cannot be serialized and must be materialized into an encrypted Profile.
type PendingImport struct {
	profile   Profile
	secret    string
	reference string
	source    CredentialSource
}

func ParseImport(reader io.Reader, maxBytes int64) (PendingImport, error) {
	if maxBytes <= 0 || maxBytes > DefaultMaxImportBytes {
		maxBytes = DefaultMaxImportBytes
	}
	data, err := io.ReadAll(io.LimitReader(reader, maxBytes+1))
	if err != nil {
		return PendingImport{}, fmt.Errorf("read provider import: %w", err)
	}
	if int64(len(data)) > maxBytes {
		return PendingImport{}, errors.New("provider import exceeds size limit")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var wire importWire
	if err := decoder.Decode(&wire); err != nil {
		return PendingImport{}, fmt.Errorf("invalid provider import: %w", err)
	}
	if err := ensureJSONEOF(decoder); err != nil {
		return PendingImport{}, err
	}
	p := Normalize(Profile{
		ID: wire.ID, Name: wire.Name, Kind: wire.Kind, BaseURL: wire.BaseURL,
		Model: wire.Model, Enabled: wire.Enabled, Priority: wire.Priority,
		FallbackGroup: wire.FallbackGroup, TimeoutSeconds: wire.TimeoutSeconds,
		MaxResponseBytes: wire.MaxResponseBytes, Headers: wire.Headers,
	})
	if wire.Credential.Source == CredentialEncryptedInline {
		if wire.Credential.Reference != "" || validatePlainSecret(wire.Credential.Secret) != nil {
			return PendingImport{}, errors.New("encrypted_inline import requires only a valid plaintext secret")
		}
	} else {
		if wire.Credential.Secret != "" {
			return PendingImport{}, errors.New("plaintext secret is allowed only for encrypted_inline import")
		}
	}
	return PendingImport{profile: p, secret: wire.Credential.Secret, reference: wire.Credential.Reference, source: wire.Credential.Source}, nil
}

func ensureJSONEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return errors.New("provider import must contain exactly one JSON object")
		}
		return fmt.Errorf("invalid trailing provider import data: %w", err)
	}
	return nil
}

func (p PendingImport) Preview() View {
	profile := p.profile
	profile.Credential = Credential{Source: p.source}
	if p.source == CredentialEnvironment {
		profile.Credential.Reference = p.reference
	}
	if p.source == CredentialEncryptedInline {
		profile.Credential.Last4 = secretLast4(p.secret)
	}
	return Redacted(profile)
}

func (p PendingImport) Materialize(aead cipher.AEAD, policy ValidationPolicy) (Profile, error) {
	profile := p.profile
	var err error
	switch p.source {
	case CredentialEncryptedInline:
		profile.Credential, err = SealInline(aead, profile.ID, p.secret)
	case CredentialEnvironment:
		profile.Credential = Credential{Source: CredentialEnvironment, Reference: p.reference}
	case CredentialFile:
		profile.Credential = Credential{Source: CredentialFile, Reference: p.reference}
	case CredentialNone:
		profile.Credential = Credential{Source: CredentialNone}
	default:
		err = errors.New("unsupported imported credential source")
	}
	if err != nil {
		return Profile{}, err
	}
	profile = Normalize(profile)
	if err := Validate(profile, policy); err != nil {
		return Profile{}, err
	}
	return profile, nil
}

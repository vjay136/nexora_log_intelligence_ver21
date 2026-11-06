// Package aiprovider provides the controller-authoritative configuration and
// transport primitives for optional AI assistance. It is deliberately
// independent from the live parsing path: callers may use it to obtain review
// suggestions, but a provider failure must never stop log processing.
package aiprovider

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"
)

type Kind string

const (
	KindOpenAI           Kind = "openai"
	KindNVIDIANIM        Kind = "nvidia_nim"
	KindAnthropic        Kind = "anthropic"
	KindMiniMax          Kind = "minimax"
	KindQwenDashScope    Kind = "qwen_dashscope"
	KindGemini           Kind = "gemini"
	KindOpenAICompatible Kind = "openai_compatible"
	KindOllama           Kind = "ollama"
)

type CredentialSource string

const (
	CredentialEncryptedInline CredentialSource = "encrypted_inline"
	CredentialEnvironment     CredentialSource = "env_ref"
	CredentialFile            CredentialSource = "file_ref"
	CredentialNone            CredentialSource = "none"
)

// Credential contains metadata and, for inline credentials, authenticated
// ciphertext. Plaintext credentials never belong in a stored Profile.
type Credential struct {
	Source     CredentialSource `json:"source"`
	Reference  string           `json:"reference,omitempty"`
	Nonce      string           `json:"nonce,omitempty"`
	Ciphertext string           `json:"ciphertext,omitempty"`
	Last4      string           `json:"last4,omitempty"`
}

// Profile is the durable, revisioned provider configuration. Headers are
// intentionally non-secret. Provider authentication is represented only by
// Credential.
type Profile struct {
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
	Credential       Credential        `json:"credential"`
	Revision         uint64            `json:"revision"`
	CreatedAt        time.Time         `json:"created_at,omitempty"`
	UpdatedAt        time.Time         `json:"updated_at,omitempty"`
	UpdatedBy        string            `json:"updated_by,omitempty"`
}

type ValidationPolicy struct {
	AllowLoopback bool
	AllowPrivate  bool
	SecretRoot    string
}

type View struct {
	ID               string            `json:"id"`
	Name             string            `json:"name"`
	Kind             Kind              `json:"kind"`
	BaseURL          string            `json:"base_url"`
	Model            string            `json:"model"`
	Enabled          bool              `json:"enabled"`
	Priority         int               `json:"priority"`
	FallbackGroup    string            `json:"fallback_group,omitempty"`
	TimeoutSeconds   int               `json:"timeout_seconds"`
	MaxResponseBytes int64             `json:"max_response_bytes"`
	Headers          map[string]string `json:"headers,omitempty"`
	CredentialSource CredentialSource  `json:"credential_source"`
	CredentialRef    string            `json:"credential_ref,omitempty"`
	CredentialLast4  string            `json:"credential_last4,omitempty"`
	Configured       bool              `json:"configured"`
	Revision         uint64            `json:"revision"`
	UpdatedAt        time.Time         `json:"updated_at,omitempty"`
	UpdatedBy        string            `json:"updated_by,omitempty"`
}

var (
	idPattern       = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,63}$`)
	groupPattern    = regexp.MustCompile(`^[a-zA-Z0-9_.-]{0,64}$`)
	envPattern      = regexp.MustCompile(`^[A-Z_][A-Z0-9_]{0,127}$`)
	headerName      = regexp.MustCompile(`^[A-Za-z0-9!#$%&'*+.^_` + "`" + `|~-]{1,128}$`)
	forbiddenHeader = map[string]struct{}{
		"authorization": {}, "proxy-authorization": {}, "x-api-key": {},
		"api-key": {}, "x-goog-api-key": {}, "host": {}, "content-length": {},
		"connection": {}, "transfer-encoding": {}, "cookie": {}, "set-cookie": {},
	}
)

func knownKind(kind Kind) bool {
	switch kind {
	case KindOpenAI, KindNVIDIANIM, KindAnthropic, KindMiniMax, KindQwenDashScope,
		KindGemini, KindOpenAICompatible, KindOllama:
		return true
	default:
		return false
	}
}

func Normalize(p Profile) Profile {
	p.ID = strings.ToLower(strings.TrimSpace(p.ID))
	p.Name = strings.TrimSpace(p.Name)
	p.Kind = Kind(strings.ToLower(strings.TrimSpace(string(p.Kind))))
	p.BaseURL = strings.TrimRight(strings.TrimSpace(p.BaseURL), "/")
	p.Model = strings.TrimSpace(p.Model)
	p.FallbackGroup = strings.TrimSpace(p.FallbackGroup)
	p.Credential.Reference = strings.TrimSpace(p.Credential.Reference)
	if p.TimeoutSeconds == 0 {
		p.TimeoutSeconds = 15
	}
	if p.MaxResponseBytes == 0 {
		p.MaxResponseBytes = 1 << 20
	}
	return p
}

func Validate(p Profile, policy ValidationPolicy) error {
	p = Normalize(p)
	if err := ValidateMetadata(p, policy); err != nil {
		return err
	}
	return validateCredential(p.Kind, p.Credential, policy)
}

func ValidateMetadata(p Profile, policy ValidationPolicy) error {
	p = Normalize(p)
	if !idPattern.MatchString(p.ID) {
		return errors.New("provider id must start with a letter and contain only lowercase letters, digits, underscore or hyphen")
	}
	if p.Name == "" || len(p.Name) > 128 {
		return errors.New("provider name is required and must not exceed 128 characters")
	}
	if !knownKind(p.Kind) {
		return fmt.Errorf("unsupported provider kind %q", p.Kind)
	}
	if p.Model == "" || len(p.Model) > 256 {
		return errors.New("model is required and must not exceed 256 characters")
	}
	if p.Priority < 0 || p.Priority > 100000 {
		return errors.New("priority must be between 0 and 100000")
	}
	if !groupPattern.MatchString(p.FallbackGroup) {
		return errors.New("fallback group contains invalid characters")
	}
	if p.TimeoutSeconds < 1 || p.TimeoutSeconds > 120 {
		return errors.New("timeout_seconds must be between 1 and 120")
	}
	if p.MaxResponseBytes < 1024 || p.MaxResponseBytes > 8<<20 {
		return errors.New("max_response_bytes must be between 1024 and 8388608")
	}
	if err := validateBaseURL(p, policy); err != nil {
		return err
	}
	if err := validateHeaders(p.Headers); err != nil {
		return err
	}
	return nil
}

func validateBaseURL(p Profile, policy ValidationPolicy) error {
	if p.BaseURL == "" || len(p.BaseURL) > 2048 {
		return errors.New("base_url is required and must not exceed 2048 characters")
	}
	u, err := url.Parse(p.BaseURL)
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return errors.New("base_url must be an absolute HTTP(S) URL without credentials, query, or fragment")
	}
	if u.Scheme != "https" && u.Scheme != "http" {
		return errors.New("base_url scheme must be https or http")
	}
	allowLoopback := policy.AllowLoopback || p.Kind == KindOllama
	loopbackName := strings.EqualFold(u.Hostname(), "localhost")
	if ip := net.ParseIP(u.Hostname()); ip != nil {
		addressPolicy := policy
		addressPolicy.AllowLoopback = allowLoopback
		if err := ValidateResolvedIP(ip, addressPolicy); err != nil {
			return fmt.Errorf("base_url: %w", err)
		}
		loopbackName = ip.IsLoopback()
	}
	if loopbackName && !allowLoopback {
		return errors.New("loopback base_url is not allowed for this provider")
	}
	if u.Scheme == "http" && !(allowLoopback && loopbackName) {
		return errors.New("base_url must use HTTPS except for an explicitly allowed loopback provider")
	}
	return nil
}

func ValidateResolvedIP(ip net.IP, policy ValidationPolicy) error {
	if ip == nil {
		return errors.New("endpoint did not resolve to a valid IP address")
	}
	if ip.IsLoopback() {
		if policy.AllowLoopback {
			return nil
		}
		return errors.New("loopback endpoint is not allowed")
	}
	if ip.IsUnspecified() || ip.IsMulticast() || ip.IsLinkLocalMulticast() || ip.IsLinkLocalUnicast() {
		return errors.New("unspecified, multicast, or link-local endpoint is not allowed")
	}
	if ip.IsPrivate() && !policy.AllowPrivate {
		return errors.New("private endpoint is not allowed")
	}
	return nil
}

func validateHeaders(headers map[string]string) error {
	if len(headers) > 16 {
		return errors.New("at most 16 custom headers are allowed")
	}
	for name, value := range headers {
		if !headerName.MatchString(name) {
			return fmt.Errorf("invalid custom header name %q", name)
		}
		if _, blocked := forbiddenHeader[strings.ToLower(name)]; blocked {
			return fmt.Errorf("custom header %q is controlled by the provider adapter", name)
		}
		if len(value) > 1024 || strings.ContainsAny(value, "\r\n") {
			return fmt.Errorf("custom header %q has an invalid value", name)
		}
	}
	return nil
}

func validateCredential(kind Kind, c Credential, policy ValidationPolicy) error {
	switch c.Source {
	case CredentialEncryptedInline:
		if c.Reference != "" || c.Nonce == "" || c.Ciphertext == "" {
			return errors.New("encrypted_inline credential requires nonce and ciphertext only")
		}
		nonce, nonceErr := base64.RawStdEncoding.DecodeString(c.Nonce)
		ciphertext, ciphertextErr := base64.RawStdEncoding.DecodeString(c.Ciphertext)
		if nonceErr != nil || len(nonce) < 8 || len(nonce) > 32 || ciphertextErr != nil || len(ciphertext) < 24 || len(ciphertext) > maxSecretBytes+64 {
			return errors.New("encrypted_inline credential metadata is invalid")
		}
		if len(c.Last4) > 4 || strings.ContainsAny(c.Last4, "\r\n\x00") {
			return errors.New("encrypted_inline credential suffix is invalid")
		}
	case CredentialEnvironment:
		if !envPattern.MatchString(c.Reference) || c.Nonce != "" || c.Ciphertext != "" || c.Last4 != "" {
			return errors.New("env_ref credential requires a valid uppercase environment variable name")
		}
	case CredentialFile:
		if c.Reference == "" || c.Nonce != "" || c.Ciphertext != "" || c.Last4 != "" {
			return errors.New("file_ref credential requires a path reference")
		}
		if _, err := secureSecretPath(policy.SecretRoot, c.Reference); err != nil {
			return err
		}
	case CredentialNone:
		if kind != KindOllama || c.Reference != "" || c.Nonce != "" || c.Ciphertext != "" || c.Last4 != "" {
			return errors.New("credential source none is allowed only for Ollama")
		}
	default:
		return errors.New("credential source must be encrypted_inline, env_ref, file_ref, or none")
	}
	return nil
}

func Redacted(p Profile) View {
	p = Normalize(p)
	ref := ""
	if p.Credential.Source == CredentialEnvironment {
		ref = p.Credential.Reference
	} else if p.Credential.Source == CredentialFile {
		ref = "managed secret file"
	}
	headers := make(map[string]string, len(p.Headers))
	for k := range p.Headers {
		headers[k] = "[configured]"
	}
	return View{
		ID: p.ID, Name: p.Name, Kind: p.Kind, BaseURL: p.BaseURL, Model: p.Model,
		Enabled: p.Enabled, Priority: p.Priority, FallbackGroup: p.FallbackGroup,
		TimeoutSeconds: p.TimeoutSeconds, MaxResponseBytes: p.MaxResponseBytes,
		Headers: headers, CredentialSource: p.Credential.Source, CredentialRef: ref,
		CredentialLast4: p.Credential.Last4, Configured: p.Credential.Source != "",
		Revision: p.Revision, UpdatedAt: p.UpdatedAt, UpdatedBy: p.UpdatedBy,
	}
}

// Route returns enabled providers in a deterministic order. Lower numeric
// priority wins; equal priorities are ordered by stable provider ID.
func Route(profiles []Profile, fallbackGroup string) []Profile {
	result := make([]Profile, 0, len(profiles))
	for _, p := range profiles {
		p = Normalize(p)
		if p.Enabled && (fallbackGroup == "" || p.FallbackGroup == fallbackGroup) {
			result = append(result, p)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Priority == result[j].Priority {
			return result[i].ID < result[j].ID
		}
		return result[i].Priority < result[j].Priority
	})
	return result
}

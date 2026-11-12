package aiprovider

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode"
)

const maxDiscoveredModels = 250
const maxCatalogPages = 3

var ErrModelDiscoveryUnsupported = errors.New("provider profile does not expose a supported model catalog")
var discoveredModelID = regexp.MustCompile(`^[A-Za-z0-9@][A-Za-z0-9._:/+@-]{0,255}$`)

type DiscoveredModel struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

type ModelCatalog struct {
	Models    []DiscoveredModel `json:"models"`
	Truncated bool              `json:"truncated"`
}

type providerModel struct {
	ID                         string   `json:"id"`
	Name                       string   `json:"name"`
	Model                      string   `json:"model"`
	DisplayName                string   `json:"display_name"`
	GeminiDisplayName          string   `json:"displayName"`
	SupportedGenerationMethods []string `json:"supportedGenerationMethods"`
}

// DiscoverModels issues read-only catalog calls. A listed model is advertised
// by the endpoint and still requires a generation test for its intended use.
func DiscoverModels(ctx context.Context, profile Profile, resolver SecretResolver, policy ValidationPolicy) (ModelCatalog, error) {
	profile = Normalize(profile)
	if profile.Kind == KindOllama {
		policy.AllowLoopback = true
	}
	if err := Validate(profile, policy); err != nil {
		return ModelCatalog{}, errors.New("invalid provider profile for discovery")
	}
	suffix, err := modelCatalogPath(profile)
	if err != nil {
		return ModelCatalog{}, err
	}
	resolver.SecretRoot = policy.SecretRoot
	credential, err := resolver.Resolve(profile)
	if err != nil {
		return ModelCatalog{}, errors.New("provider credential is unavailable")
	}
	timeout := minDuration(time.Duration(profile.TimeoutSeconds)*time.Second, 20*time.Second)
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	client := NewSafeHTTPClient(policy, timeout)
	defer client.CloseIdleConnections()
	catalog := ModelCatalog{Models: []DiscoveredModel{}}
	seen := make(map[string]bool)
	cursor := ""
	for page := 0; page < maxCatalogPages; page++ {
		items, next, err := readModelCatalogPage(ctx, client, profile, credential, suffix, cursor)
		if err != nil {
			return ModelCatalog{}, err
		}
		for _, item := range items {
			id, label := catalogModelNames(profile.Kind, item)
			if !discoveredModelID.MatchString(id) || seen[id] ||
				(credential != "" && (strings.Contains(id, credential) || strings.Contains(label, credential))) {
				continue
			}
			if len(catalog.Models) == maxDiscoveredModels {
				catalog.Truncated = true
				return catalog, nil
			}
			seen[id] = true
			catalog.Models = append(catalog.Models, DiscoveredModel{ID: id, Label: label})
		}
		if next == "" {
			return catalog, nil
		}
		if next == cursor {
			break
		}
		cursor = next
	}
	catalog.Truncated = true
	return catalog, nil
}

func modelCatalogPath(profile Profile) (string, error) {
	switch profile.Kind {
	case KindOpenAI, KindNVIDIANIM, KindAnthropic:
		return "models", nil
	case KindGemini:
		base, err := url.Parse(profile.BaseURL)
		if err == nil && (base.Path == "/v1beta" || base.Path == "/v1") {
			return "models", nil
		}
	case KindOllama:
		return "api/tags", nil
	case KindOpenAICompatible:
		base, err := url.Parse(profile.BaseURL)
		if err == nil {
			switch strings.ToLower(base.Hostname()) {
			case "api.openai.com", "api.groq.com", "openrouter.ai", "api.deepseek.com", "api.mistral.ai", "api.cerebras.ai", "ai-gateway.vercel.sh", "router.huggingface.co", "qianfan.baidubce.com":
				return "models", nil
			}
		}
	}
	return "", ErrModelDiscoveryUnsupported
}

func readModelCatalogPage(ctx context.Context, client *http.Client, profile Profile, credential, suffix, cursor string) ([]providerModel, string, error) {
	endpoint, err := url.Parse(appendEndpoint(profile.BaseURL, suffix))
	if err != nil {
		return nil, "", ErrModelDiscoveryUnsupported
	}
	query := endpoint.Query()
	switch profile.Kind {
	case KindGemini:
		query.Set("pageSize", "100")
		if cursor != "" {
			query.Set("pageToken", cursor)
		}
	case KindAnthropic:
		query.Set("limit", "100")
		if cursor != "" {
			query.Set("after_id", cursor)
		}
	}
	endpoint.RawQuery = query.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, "", ErrModelDiscoveryUnsupported
	}
	req.Header.Set("Accept", "application/json")
	switch profile.Kind {
	case KindGemini:
		req.Header.Set("x-goog-api-key", credential)
	case KindAnthropic:
		req.Header.Set("x-api-key", credential)
		req.Header.Set("anthropic-version", "2023-06-01")
	default:
		if credential != "" {
			req.Header.Set("Authorization", "Bearer "+credential)
		}
	}
	for name, value := range profile.Headers {
		req.Header.Set(name, value)
	}
	response, err := client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, "", ctx.Err()
		}
		return nil, "", ErrRequestTransport
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, "", &HTTPStatusError{StatusCode: response.StatusCode}
	}
	limit := min(profile.MaxResponseBytes, 2<<20)
	body, err := io.ReadAll(io.LimitReader(response.Body, limit+1))
	if err != nil || int64(len(body)) > limit {
		return nil, "", errors.New("provider model catalog exceeded the response limit or could not be read")
	}
	var envelope struct {
		Models        []providerModel `json:"models"`
		Data          []providerModel `json:"data"`
		NextPageToken string          `json:"nextPageToken"`
		HasMore       bool            `json:"has_more"`
		LastID        string          `json:"last_id"`
	}
	if json.Unmarshal(body, &envelope) != nil {
		return nil, "", errors.New("provider returned an invalid model catalog")
	}
	switch profile.Kind {
	case KindGemini:
		if envelope.Models != nil {
			return envelope.Models, envelope.NextPageToken, nil
		}
	case KindOllama:
		if envelope.Models != nil {
			return envelope.Models, "", nil
		}
	default:
		if envelope.Data != nil {
			if profile.Kind == KindAnthropic && envelope.HasMore && envelope.LastID != "" {
				return envelope.Data, envelope.LastID, nil
			}
			return envelope.Data, "", nil
		}
	}
	return nil, "", errors.New("provider did not return a recognized model list")
}

func catalogModelNames(kind Kind, item providerModel) (string, string) {
	id, label := item.ID, item.DisplayName
	switch kind {
	case KindGemini:
		if !slices.Contains(item.SupportedGenerationMethods, "generateContent") || !strings.HasPrefix(item.Name, "models/") {
			return "", ""
		}
		id, label = strings.TrimPrefix(item.Name, "models/"), item.GeminiDisplayName
	case KindOllama:
		id = item.Model
		if id == "" {
			id = item.Name
		}
	default:
		if label == "" {
			label = item.Name
		}
	}
	label = strings.TrimSpace(strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, label))
	if label == "" || len(label) > 256 {
		label = id
	}
	return id, label
}

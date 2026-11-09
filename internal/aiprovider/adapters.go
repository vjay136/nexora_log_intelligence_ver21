package aiprovider

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type ConnectionResult struct {
	OK           bool          `json:"ok"`
	StatusCode   int           `json:"status_code,omitempty"`
	ErrorCode    string        `json:"error_code,omitempty"`
	Latency      time.Duration `json:"latency"`
	Acknowledged bool          `json:"acknowledged,omitempty"`
	TestedAt     time.Time     `json:"tested_at"`
}

type Tester struct {
	Policy   ValidationPolicy
	Resolver SecretResolver
}

func BuildRequest(ctx context.Context, profile Profile, credential, prompt string) (*http.Request, error) {
	return BuildRequestWithOptions(ctx, profile, credential, prompt, RequestOptions{})
}

// RequestOptions adjusts generation without changing the provider profile.
// Connection tests intentionally use the small defaults while review requests
// can ask for a bounded, useful response.
type RequestOptions struct {
	SystemPrompt string
	JSONResponse bool
	MaxTokens    int
	Stream       bool
	ImageURL     string
}

func BuildRequestWithOptions(ctx context.Context, profile Profile, credential, prompt string, options RequestOptions) (*http.Request, error) {
	profile = Normalize(profile)
	if len(prompt) == 0 || len(prompt) > 64*1024 {
		return nil, errors.New("prompt must contain between 1 and 65536 bytes")
	}
	maxTokens := options.MaxTokens
	if maxTokens == 0 {
		maxTokens = 16
	}
	if maxTokens < 1 || maxTokens > 8192 {
		return nil, errors.New("max tokens must be between 1 and 8192")
	}
	if (options.Stream || options.ImageURL != "") && (profile.Kind == KindAnthropic || profile.Kind == KindGemini || profile.Kind == KindOllama) {
		return nil, errors.New("streaming/vision probe is supported only by OpenAI-compatible adapters")
	}
	var endpoint string
	var body any
	switch profile.Kind {
	case KindAnthropic:
		endpoint = appendEndpoint(profile.BaseURL, "messages")
		body = map[string]any{"model": profile.Model, "max_tokens": maxTokens, "messages": []map[string]string{{"role": "user", "content": prompt}}}
		if options.SystemPrompt != "" {
			body.(map[string]any)["system"] = options.SystemPrompt
		}
	case KindGemini:
		endpoint = appendEndpoint(profile.BaseURL, "models/"+url.PathEscape(profile.Model)+":generateContent")
		generation := map[string]any{"maxOutputTokens": maxTokens}
		if options.JSONResponse {
			generation["responseMimeType"] = "application/json"
		}
		body = map[string]any{"contents": []map[string]any{{"role": "user", "parts": []map[string]string{{"text": prompt}}}}, "generationConfig": generation}
		if options.SystemPrompt != "" {
			body.(map[string]any)["systemInstruction"] = map[string]any{"parts": []map[string]string{{"text": options.SystemPrompt}}}
		}
	case KindOllama:
		endpoint = appendEndpoint(profile.BaseURL, "api/chat")
		messages := []map[string]string{}
		if options.SystemPrompt != "" {
			messages = append(messages, map[string]string{"role": "system", "content": options.SystemPrompt})
		}
		messages = append(messages, map[string]string{"role": "user", "content": prompt})
		body = map[string]any{"model": profile.Model, "stream": false, "messages": messages, "options": map[string]int{"num_predict": maxTokens}}
	default:
		if !knownKind(profile.Kind) {
			return nil, errors.New("unsupported provider kind")
		}
		endpoint = appendEndpoint(profile.BaseURL, "chat/completions")
		messages := []map[string]string{}
		if options.SystemPrompt != "" {
			messages = append(messages, map[string]string{"role": "system", "content": options.SystemPrompt})
		}
		messages = append(messages, map[string]string{"role": "user", "content": prompt})
		tokenParam := "max_tokens"
		if profile.Kind == KindOpenAI && (strings.HasPrefix(profile.Model, "gpt-5") || strings.HasPrefix(profile.Model, "o3") || strings.HasPrefix(profile.Model, "o4")) {
			tokenParam = "max_completion_tokens"
		}
		body = map[string]any{"model": profile.Model, tokenParam: maxTokens, "messages": messages}
		if profile.Kind == KindOpenAI && options.JSONResponse {
			body.(map[string]any)["response_format"] = map[string]string{"type": "json_object"}
		}
		if options.Stream {
			body.(map[string]any)["stream"] = true
		}
		if options.ImageURL != "" {
			if err := ValidateImageProbeURL(options.ImageURL); err != nil {
				return nil, err
			}
			body.(map[string]any)["messages"] = []map[string]any{{"role": "user", "content": []map[string]any{{"type": "text", "text": prompt}, {"type": "image_url", "image_url": map[string]string{"url": options.ImageURL}}}}}
		}
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, errors.New("encode provider request")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, errors.New("build provider request")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if options.Stream {
		req.Header.Set("Accept", "text/event-stream")
	}
	switch profile.Kind {
	case KindAnthropic:
		req.Header.Set("x-api-key", credential)
		req.Header.Set("anthropic-version", "2023-06-01")
	case KindGemini:
		req.Header.Set("x-goog-api-key", credential)
	case KindOllama:
		// Local Ollama may be credential-free; authenticated proxies still use the supplied protected credential.
		if credential != "" {
			req.Header.Set("Authorization", "Bearer "+credential)
		}
	default:
		req.Header.Set("Authorization", "Bearer "+credential)
	}
	for name, value := range profile.Headers {
		req.Header.Set(name, value)
	}
	return req, nil
}

func (t Tester) Test(ctx context.Context, profile Profile) ConnectionResult {
	return t.TestWithOptions(ctx, profile, RequestOptions{MaxTokens: 64})
}
func (t Tester) TestWithOptions(ctx context.Context, profile Profile, options RequestOptions) ConnectionResult {
	started := time.Now()
	result := ConnectionResult{TestedAt: started.UTC()}
	profile = Normalize(profile)
	policy := t.Policy
	if profile.Kind == KindOllama {
		policy.AllowLoopback = true
	}
	if err := Validate(profile, policy); err != nil {
		result.ErrorCode = "invalid_profile"
		return result
	}
	resolver := t.Resolver
	resolver.SecretRoot = policy.SecretRoot
	credential, err := resolver.Resolve(profile)
	if err != nil {
		result.ErrorCode = "credential_unavailable"
		return result
	}
	timeout := time.Duration(profile.TimeoutSeconds) * time.Second
	testCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	if options.MaxTokens == 0 {
		options.MaxTokens = 64
	}
	if options.MaxTokens < 16 || options.MaxTokens > 1024 {
		result.ErrorCode = "invalid_probe_budget"
		return result
	}
	req, err := BuildRequestWithOptions(testCtx, profile, credential, "Reply with OK. If an image is supplied, describe it briefly.", options)
	if err != nil {
		result.ErrorCode = "request_invalid"
		return result
	}
	client := NewSafeHTTPClient(policy, timeout)
	resp, err := client.Do(req)
	result.Latency = time.Since(started)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(testCtx.Err(), context.DeadlineExceeded) {
			result.ErrorCode = "timeout"
		} else {
			result.ErrorCode = "network_error"
		}
		return result
	}
	defer resp.Body.Close()
	data, copyErr := io.ReadAll(io.LimitReader(resp.Body, profile.MaxResponseBytes+1))
	if copyErr != nil {
		result.ErrorCode = "response_read_error"
		return result
	}
	if int64(len(data)) > profile.MaxResponseBytes {
		result.ErrorCode = "response_too_large"
		return result
	}
	result.StatusCode = resp.StatusCode
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		result.ErrorCode = "http_" + strconv.Itoa(resp.StatusCode)
		return result
	}
	var usable bool
	if options.Stream {
		usable, result.Acknowledged, result.ErrorCode = InspectStreamProbe(data)
	} else {
		usable, result.Acknowledged, result.ErrorCode = InspectJSONProbe(profile.Kind, data)
	}
	if !usable {
		return result
	}
	result.OK = true
	return result
}

func appendEndpoint(baseURL, suffix string) string {
	base := strings.TrimRight(baseURL, "/")
	suffix = strings.TrimLeft(suffix, "/")
	if strings.HasSuffix(base, "/"+suffix) {
		return base
	}
	return base + "/" + suffix
}

func NewSafeHTTPClient(policy ValidationPolicy, timeout time.Duration) *http.Client {
	dialer := &net.Dialer{Timeout: minDuration(timeout, 10*time.Second), KeepAlive: 30 * time.Second}
	transport := &http.Transport{
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          16,
		MaxIdleConnsPerHost:   4,
		IdleConnTimeout:       30 * time.Second,
		TLSHandshakeTimeout:   minDuration(timeout, 10*time.Second),
		ResponseHeaderTimeout: timeout,
	}
	transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, errors.New("provider address is invalid")
		}
		ips, err := net.DefaultResolver.LookupIP(ctx, "ip", host)
		if err != nil || len(ips) == 0 {
			return nil, errors.New("provider DNS resolution failed")
		}
		var approved net.IP
		for _, ip := range ips {
			if err := ValidateResolvedIP(ip, policy); err != nil {
				return nil, errors.New("provider endpoint resolved to a forbidden network")
			}
			if approved == nil {
				approved = ip
			}
		}
		return dialer.DialContext(ctx, network, net.JoinHostPort(approved.String(), port))
	}
	return &http.Client{
		Transport: transport,
		Timeout:   timeout,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return errors.New("provider redirects are disabled")
		},
	}
}

func minDuration(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}

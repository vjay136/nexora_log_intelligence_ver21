package aiprovider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
)

// Generate uses the same provider adapter and network/credential policy as the
// explicit connection test. It never persists secrets or activates a parser.
func Generate(ctx context.Context, profile Profile, resolver SecretResolver, policy ValidationPolicy, prompt string, options RequestOptions) (string, error) {
	profile = Normalize(profile)
	if profile.Kind == KindOllama {
		policy.AllowLoopback = true
	}
	if err := Validate(profile, policy); err != nil {
		return "", err
	}
	resolver.SecretRoot = policy.SecretRoot
	credential, err := resolver.Resolve(profile)
	if err != nil {
		return "", errors.New("provider credential is unavailable")
	}
	return GenerateWithCredential(ctx, profile, credential, policy, prompt, options)
}

func GenerateWithCredential(ctx context.Context, profile Profile, credential string, policy ValidationPolicy, prompt string, options RequestOptions) (string, error) {
	if credential != "" {
		if err := validatePlainSecret(credential); err != nil {
			return "", err
		}
	}
	profile = Normalize(profile)
	if profile.Kind == KindOllama {
		policy.AllowLoopback = true
	}
	if err := Validate(profile, policy); err != nil {
		return "", err
	}
	timeout := time.Duration(profile.TimeoutSeconds) * time.Second
	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	request, err := BuildRequestWithOptions(callCtx, profile, credential, prompt, options)
	if err != nil {
		return "", err
	}
	client := NewSafeHTTPClient(policy, timeout)
	response, err := client.Do(request)
	if err != nil {
		return "", errors.New("provider request failed or timed out")
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, profile.MaxResponseBytes+1))
	if err != nil {
		return "", errors.New("provider response could not be read")
	}
	if int64(len(data)) > profile.MaxResponseBytes {
		return "", errors.New("provider response exceeded configured limit")
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", fmt.Errorf("provider returned HTTP %d", response.StatusCode)
	}
	return ResponseText(profile.Kind, data)
}

// ResponseText shares existing supported response envelopes with review calls.
func ResponseText(kind Kind, data []byte) (string, error) {
	var envelope struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text    string `json:"text"`
					Thought bool   `json:"thought"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if json.Unmarshal(data, &envelope) != nil {
		return "", errors.New("provider returned invalid JSON")
	}
	var text strings.Builder
	switch kind {
	case KindAnthropic:
		for _, block := range envelope.Content {
			if block.Type == "text" {
				text.WriteString(block.Text)
			}
		}
	case KindGemini:
		if len(envelope.Candidates) > 0 {
			for _, part := range envelope.Candidates[0].Content.Parts {
				if !part.Thought {
					text.WriteString(part.Text)
				}
			}
		}
	case KindOllama:
		text.WriteString(envelope.Message.Content)
	default:
		if len(envelope.Choices) > 0 {
			text.WriteString(envelope.Choices[0].Message.Content)
		}
	}
	if strings.TrimSpace(text.String()) == "" {
		return "", errors.New("provider returned no text content")
	}
	return text.String(), nil
}

package aiprovider

import (
	"encoding/json"
	"errors"
	"net"
	"net/url"
	"strings"
)

func ValidateImageProbeURL(value string) error {
	if len(value) > 2048 {
		return errors.New("public image URL exceeds limit")
	}
	u, err := url.Parse(value)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.Fragment != "" || u.RawQuery != "" {
		return errors.New("vision probe needs a public HTTPS image URL without credentials, query or fragment")
	}
	host := strings.ToLower(u.Hostname())
	if host == "localhost" || strings.HasSuffix(host, ".local") || strings.HasSuffix(host, ".internal") {
		return errors.New("vision image host must be public")
	}
	if ip := net.ParseIP(host); ip != nil {
		if err := ValidateResolvedIP(ip, ValidationPolicy{}); err != nil {
			return errors.New("vision image address must be public")
		}
	}
	return nil
}
func InspectJSONProbe(kind Kind, data []byte) (usable, acknowledged bool, errorCode string) {
	text, err := ResponseText(kind, data)
	if errors.Is(err, ErrOutputTokenLimit) {
		return false, true, "incomplete_output"
	}
	var envelope struct {
		Choices []struct {
			Message struct {
				Reasoning string `json:"reasoning_content"`
			} `json:"message"`
			Finish string `json:"finish_reason"`
		} `json:"choices"`
	}
	_ = json.Unmarshal(data, &envelope)
	if len(envelope.Choices) > 0 {
		choice := envelope.Choices[0]
		if choice.Finish == "length" || err != nil && strings.TrimSpace(choice.Message.Reasoning) != "" {
			return false, true, "incomplete_output"
		}
	}
	if err != nil || strings.TrimSpace(text) == "" {
		return false, false, "invalid_protocol_response"
	}
	return true, true, ""
}

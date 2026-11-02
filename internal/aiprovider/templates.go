package aiprovider

type Template struct {
	ID         string           `json:"id"`
	Name       string           `json:"name"`
	Kind       Kind             `json:"kind"`
	BaseURL    string           `json:"base_url"`
	ModelHint  string           `json:"model_hint"`
	Credential CredentialSource `json:"credential_source"`
}

// Templates contains only public endpoint metadata. It never contains keys,
// encrypted values, environment names, or secret file paths.
func Templates() []Template {
	return []Template{
		{ID: "openai", Name: "OpenAI", Kind: KindOpenAI, BaseURL: "https://api.openai.com/v1", ModelHint: "gpt-4.1-mini", Credential: CredentialEncryptedInline},
		{ID: "nvidia-nim", Name: "NVIDIA NIM", Kind: KindNVIDIANIM, BaseURL: "https://integrate.api.nvidia.com/v1", ModelHint: "provider/model", Credential: CredentialEncryptedInline},
		{ID: "anthropic", Name: "Anthropic Claude", Kind: KindAnthropic, BaseURL: "https://api.anthropic.com/v1", ModelHint: "claude-sonnet", Credential: CredentialEncryptedInline},
		{ID: "minimax", Name: "MiniMax", Kind: KindMiniMax, BaseURL: "https://api.minimax.io/v1", ModelHint: "MiniMax model", Credential: CredentialEncryptedInline},
		{ID: "qwen-dashscope", Name: "Qwen / DashScope", Kind: KindQwenDashScope, BaseURL: "https://dashscope-intl.aliyuncs.com/compatible-mode/v1", ModelHint: "qwen model", Credential: CredentialEncryptedInline},
		{ID: "gemini", Name: "Gemini / Google", Kind: KindGemini, BaseURL: "https://generativelanguage.googleapis.com/v1beta", ModelHint: "gemini model", Credential: CredentialEncryptedInline},
		{ID: "openai-compatible", Name: "OpenAI-compatible endpoint", Kind: KindOpenAICompatible, BaseURL: "https://example.invalid/v1", ModelHint: "model name", Credential: CredentialEncryptedInline},
		{ID: "ollama", Name: "Ollama (controller-local)", Kind: KindOllama, BaseURL: "http://127.0.0.1:11434", ModelHint: "local model", Credential: CredentialNone},
	}
}

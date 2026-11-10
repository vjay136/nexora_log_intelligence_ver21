package aiprovider

type Template struct {
	ID         string           `json:"id"`
	Name       string           `json:"name"`
	Kind       Kind             `json:"kind"`
	BaseURL    string           `json:"base_url"`
	ModelHint  string           `json:"model_hint"`
	Models     []string         `json:"models,omitempty"`
	FillModel  bool             `json:"fill_model,omitempty"`
	Credential CredentialSource `json:"credential_source"`
}

// Templates contains only public endpoint metadata. It never contains keys,
// encrypted values, environment names, or secret file paths.
func Templates() []Template {
	return []Template{
		{ID: "openai", Name: "OpenAI", Kind: KindOpenAI, BaseURL: "https://api.openai.com/v1", ModelHint: "gpt-4.1-mini", Models: []string{"gpt-4.1", "gpt-4.1-nano"}, FillModel: true, Credential: CredentialEncryptedInline},
		{ID: "nvidia-nim", Name: "NVIDIA NIM", Kind: KindNVIDIANIM, BaseURL: "https://integrate.api.nvidia.com/v1", ModelHint: "minimaxai/minimax-m3.1-flash", FillModel: true, Credential: CredentialEncryptedInline},
		{ID: "anthropic", Name: "Anthropic Claude", Kind: KindAnthropic, BaseURL: "https://api.anthropic.com/v1", ModelHint: "claude-haiku-5-5", FillModel: true, Credential: CredentialEncryptedInline},
		{ID: "minimax", Name: "MiniMax", Kind: KindMiniMax, BaseURL: "https://api.minimax.io/v1", ModelHint: "MiniMax model", Credential: CredentialEncryptedInline},
		{ID: "qwen-dashscope", Name: "Qwen / DashScope", Kind: KindQwenDashScope, BaseURL: "https://dashscope-intl.aliyuncs.com/compatible-mode/v1", ModelHint: "qwen model", Credential: CredentialEncryptedInline},
		{ID: "gemini", Name: "Gemini / Google", Kind: KindGemini, BaseURL: "https://generativelanguage.googleapis.com/v1beta", ModelHint: "gemini-3.5-flash-lite", Models: []string{"gemini-3.5-flash"}, FillModel: true, Credential: CredentialEncryptedInline},
		{ID: "groq", Name: "Groq", Kind: KindOpenAICompatible, BaseURL: "https://api.groq.com/openai/v1", ModelHint: "openai/gpt-oss-20b", Models: []string{"openai/gpt-oss-120b"}, FillModel: true, Credential: CredentialEncryptedInline},
		{ID: "openrouter", Name: "OpenRouter", Kind: KindOpenAICompatible, BaseURL: "https://openrouter.ai/api/v1", ModelHint: "openai/gpt-4o", FillModel: true, Credential: CredentialEncryptedInline},
		{ID: "deepseek", Name: "DeepSeek", Kind: KindOpenAICompatible, BaseURL: "https://api.deepseek.com", ModelHint: "deepseek-flash", FillModel: true, Credential: CredentialEncryptedInline},
		{ID: "cohere", Name: "Cohere", Kind: KindOpenAICompatible, BaseURL: "https://api.cohere.ai/compatibility/v1", ModelHint: "command-a-plus-05-2026", FillModel: true, Credential: CredentialEncryptedInline},
		{ID: "vertex-express", Name: "Google Vertex AI Express", Kind: KindGemini, BaseURL: "https://aiplatform.googleapis.com/v1/publishers/google", ModelHint: "gemini-3.5-flash", FillModel: true, Credential: CredentialEncryptedInline},
		{ID: "baidu-qianfan", Name: "Baidu Qianfan", Kind: KindOpenAICompatible, BaseURL: "https://qianfan.baidubce.com/v2", ModelHint: "ernie-4.5-turbo-32k", FillModel: true, Credential: CredentialEncryptedInline},
		{ID: "mistral", Name: "Mistral", Kind: KindOpenAICompatible, BaseURL: "https://api.mistral.ai/v1", ModelHint: "mistral-small-latest", FillModel: true, Credential: CredentialEncryptedInline},
		{ID: "vercel-ai-gateway", Name: "Vercel AI Gateway", Kind: KindOpenAICompatible, BaseURL: "https://ai-gateway.vercel.sh/v1", ModelHint: "openai/gpt-4.1-mini", FillModel: true, Credential: CredentialEncryptedInline},
		{ID: "cerebras", Name: "Cerebras", Kind: KindOpenAICompatible, BaseURL: "https://api.cerebras.ai/v1", ModelHint: "gpt-oss-120b", FillModel: true, Credential: CredentialEncryptedInline},
		{ID: "cloudflare-workers-ai", Name: "Cloudflare Workers AI", Kind: KindOpenAICompatible, BaseURL: "https://api.cloudflare.com/client/v4/accounts/{account_id}/ai/v1", ModelHint: "@cf/meta/llama-3.1-8b-instruct", FillModel: true, Credential: CredentialEncryptedInline},
		{ID: "hugging-face", Name: "Hugging Face Inference Providers", Kind: KindOpenAICompatible, BaseURL: "https://router.huggingface.co/v1", ModelHint: "Qwen/Qwen3-Coder-480B-A35B-Instruct", FillModel: true, Credential: CredentialEncryptedInline},
		{ID: "openai-compatible", Name: "OpenAI-compatible endpoint", Kind: KindOpenAICompatible, BaseURL: "https://example.invalid/v1", ModelHint: "model name", Credential: CredentialEncryptedInline},
		{ID: "ollama", Name: "Ollama (controller-local)", Kind: KindOllama, BaseURL: "http://127.0.0.1:11434", ModelHint: "local model", Credential: CredentialNone},
	}
}

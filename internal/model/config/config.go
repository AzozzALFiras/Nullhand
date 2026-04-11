package config

// Config holds all runtime configuration for Nullhand.
type Config struct {
	TelegramToken string `json:"telegram_token"`
	AllowedUserID int64  `json:"allowed_user_id"`
	// AIProvider selects the AI backend:
	// cloud:  claude | openai | gemini | deepseek | grok
	// local:  local  — built-in rule-based parser (zero cost, zero API)
	// local:  ollama — local LLM via Ollama server (OpenAI-compatible)
	AIProvider string `json:"ai_provider"`
	AIAPIKey   string `json:"ai_api_key,omitempty"`
	AIModel    string `json:"ai_model,omitempty"`
	// AIBaseURL overrides the API endpoint for ollama (and any OpenAI-compatible
	// provider). Empty uses the provider's default. For ollama, the default is
	// http://localhost:11434.
	AIBaseURL string `json:"ai_base_url,omitempty"`
}

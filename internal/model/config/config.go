package config

// Config holds all runtime configuration for Nullhand.
type Config struct {
	TelegramToken string `json:"telegram_token"`
	// AllowedUserID is the legacy single-user gate. AllowedUserIDs takes
	// precedence when populated; AllowedUserID is preserved for backward
	// compatibility with existing config files.
	AllowedUserID  int64   `json:"allowed_user_id"`
	AllowedUserIDs []int64 `json:"allowed_user_ids,omitempty"`
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

	// RateLimitBurst is the number of messages a single user may send
	// instantaneously before throttling kicks in. Zero means use the default.
	RateLimitBurst int `json:"rate_limit_burst,omitempty"`
	// RateLimitPerMinute is the steady-state cap (messages per minute) once
	// the burst budget is exhausted. Zero means use the default.
	// Set either field to a negative value to disable rate limiting.
	RateLimitPerMinute int `json:"rate_limit_per_minute,omitempty"`
}

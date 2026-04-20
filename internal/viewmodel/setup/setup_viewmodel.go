package setup

import (
	"fmt"

	configmodel "github.com/AzozzALFiras/Nullhand/internal/model/config"
	configrepo "github.com/AzozzALFiras/Nullhand/internal/repository/config"
	tgsvc "github.com/AzozzALFiras/Nullhand/internal/service/telegram"
	wizview "github.com/AzozzALFiras/Nullhand/internal/view/setup"
)

// ViewModel orchestrates the first-run setup wizard.
type ViewModel struct {
	view *wizview.WizardView
}

// New creates a setup ViewModel.
func New() *ViewModel {
	return &ViewModel{view: wizview.New()}
}

// Run guides the user through setup and persists the resulting config.
func (vm *ViewModel) Run() (*configmodel.Config, error) {
	vm.view.PrintBanner()
	fmt.Println("  Welcome! Let's get Nullhand configured.")

	cfg := &configmodel.Config{}

	// Step 1 — Telegram Bot Token
	vm.view.PrintStep(1, "Telegram Bot Token")
	vm.view.PrintInfo("Create a bot at https://t.me/BotFather and paste the token below.")
	for {
		token, err := vm.view.AskSecret("Bot token")
		if err != nil {
			return nil, err
		}
		if token == "" {
			vm.view.PrintError("Token cannot be empty.")
			continue
		}
		vm.view.PrintInfo("Validating token...")
		if err := tgsvc.Validate(token); err != nil {
			vm.view.PrintError(fmt.Sprintf("Invalid token: %v", err))
			continue
		}
		cfg.TelegramToken = token
		vm.view.PrintSuccess("Token valid.")
		break
	}

	// Step 2 — Allowed User ID
	vm.view.PrintStep(2, "Your Telegram User ID")
	vm.view.PrintInfo("Only this user ID will be allowed to control Nullhand.")
	vm.view.PrintInfo("Send /start to @userinfobot on Telegram to find your ID.")
	for {
		raw, err := vm.view.Ask("User ID")
		if err != nil {
			return nil, err
		}
		var id int64
		if _, err := fmt.Sscanf(raw, "%d", &id); err != nil || id <= 0 {
			vm.view.PrintError("Please enter a valid numeric Telegram user ID.")
			continue
		}
		cfg.AllowedUserID = id
		vm.view.PrintSuccess(fmt.Sprintf("Allowed user ID set to %d.", id))
		break
	}

	// Step 3 — AI Provider
	vm.view.PrintStep(3, "AI Provider")
	providers := []string{"Claude (Anthropic)", "OpenAI GPT-4o", "Google Gemini", "DeepSeek", "Grok (xAI)"}
	providerKeys := []string{"claude", "openai", "gemini", "deepseek", "grok"}

	idx, err := vm.view.Choose("Which AI provider do you want to use?", providers)
	if err != nil {
		return nil, err
	}
	cfg.AIProvider = providerKeys[idx]
	vm.view.PrintSuccess(fmt.Sprintf("Using %s.", providers[idx]))

	// Step 4 — AI API Key
	vm.view.PrintStep(4, "AI API Key")
	apiKey, err := vm.view.AskSecret("API key")
	if err != nil {
		return nil, err
	}
	if apiKey == "" {
		return nil, fmt.Errorf("API key cannot be empty")
	}
	cfg.AIAPIKey = apiKey
	cfg.AIModel = defaultModel(cfg.AIProvider)
	vm.view.PrintSuccess("API key saved.")

	// Persist
	if err := configrepo.Save(cfg); err != nil {
		return nil, fmt.Errorf("failed to save config: %w", err)
	}

	vm.view.PrintDone("")
	return cfg, nil
}

// defaultModel returns a sensible default model name for each provider.
func defaultModel(provider string) string {
	switch provider {
	case "claude":
		return "claude-opus-4-6"
	case "openai":
		return "gpt-4o"
	case "gemini":
		return "gemini-1.5-pro"
	case "deepseek":
		return "deepseek-chat"
	case "grok":
		return "grok-2-vision-1212"
	default:
		return ""
	}
}

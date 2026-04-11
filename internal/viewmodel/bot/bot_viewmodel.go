package bot

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	configmodel "github.com/AzozzALFiras/nullhand/internal/model/config"
	msgmodel "github.com/AzozzALFiras/nullhand/internal/model/message"
	"github.com/AzozzALFiras/nullhand/internal/safety"
	reciperepo "github.com/AzozzALFiras/nullhand/internal/repository/recipe"
	aisvc "github.com/AzozzALFiras/nullhand/internal/service/ai"
	"github.com/AzozzALFiras/nullhand/internal/service/ai/claude"
	"github.com/AzozzALFiras/nullhand/internal/service/ai/deepseek"
	"github.com/AzozzALFiras/nullhand/internal/service/ai/gemini"
	"github.com/AzozzALFiras/nullhand/internal/service/ai/grok"
	"github.com/AzozzALFiras/nullhand/internal/service/ai/local"
	"github.com/AzozzALFiras/nullhand/internal/service/ai/ollama"
	"github.com/AzozzALFiras/nullhand/internal/service/ai/openai"
	recipesvc "github.com/AzozzALFiras/nullhand/internal/service/recipe"
	tgsvc "github.com/AzozzALFiras/nullhand/internal/service/telegram"
	tgfmt "github.com/AzozzALFiras/nullhand/internal/view/telegram"
	agentvm "github.com/AzozzALFiras/nullhand/internal/viewmodel/agent"
	cmdvm "github.com/AzozzALFiras/nullhand/internal/viewmodel/command"
	routervm "github.com/AzozzALFiras/nullhand/internal/viewmodel/router"
)

// ViewModel is the top-level orchestrator: it wires the poller, router,
// command handler, and AI agent together.
type ViewModel struct {
	cfg     *configmodel.Config
	tg      *tgsvc.Client
	poller  *tgsvc.Poller
	guard   *safety.Guard
	router  *routervm.ViewModel
	cmdExec *cmdvm.ViewModel
	agent   *agentvm.ViewModel

	stopMu  sync.Mutex
	stopCtx context.CancelFunc

	// pending holds chat IDs that are waiting for a manual-focus confirmation.
	// When the agent calls request_manual_focus, it adds a channel here and
	// blocks on it until the next user message for that chat arrives.
	pendingMu sync.Mutex
	pending   map[int64]chan string
}

// New creates and wires all components for the bot.
func New(cfg *configmodel.Config) (*ViewModel, error) {
	tgClient := tgsvc.NewClient(cfg.TelegramToken)
	aiProvider, err := buildProvider(cfg)
	if err != nil {
		return nil, err
	}

	// Load recipes: built-in defaults merged with ~/.nullhand/recipes.json overrides.
	recipes := recipesvc.New(reciperepo.Load(recipesvc.Defaults()))

	vm := &ViewModel{
		cfg:     cfg,
		tg:      tgClient,
		guard:   safety.New(cfg.AllowedUserID),
		router:  routervm.New(),
		cmdExec: cmdvm.New(),
		agent:   agentvm.New(aiProvider, recipes),
		pending: make(map[int64]chan string),
	}

	vm.poller = tgsvc.NewPoller(tgClient, vm.handleUpdate)
	return vm, nil
}

// Start begins polling for Telegram messages (blocking).
func (vm *ViewModel) Start() {
	if err := vm.tg.SetMyCommands(defaultMenu()); err != nil {
		log.Printf("warning: failed to register bot command menu: %v", err)
	}
	log.Println("Nullhand started. Waiting for messages...")
	vm.poller.Start()
}

// defaultMenu is the list of commands shown in the Telegram UI menu.
func defaultMenu() []tgsvc.BotCommand {
	return []tgsvc.BotCommand{
		{Command: "start", Description: "Welcome + command list"},
		{Command: "help", Description: "Show all commands"},
		{Command: "screenshot", Description: "Capture the screen"},
		{Command: "status", Description: "CPU, memory, active app"},
		{Command: "apps", Description: "List running applications"},
		{Command: "open", Description: "Open an application"},
		{Command: "ls", Description: "List directory contents"},
		{Command: "read", Description: "Read a file"},
		{Command: "shell", Description: "Run a whitelisted shell command"},
		{Command: "click", Description: "Click at x y coordinates"},
		{Command: "type", Description: "Type text"},
		{Command: "key", Description: "Press a key or shortcut"},
		{Command: "paste", Description: "Get clipboard contents"},
		{Command: "stop", Description: "Cancel current AI task"},
		{Command: "diag", Description: "Show diagnostic info (frontmost app, apps, screen)"},
		{Command: "inspect", Description: "Dump AX tree of frontmost window (debug)"},
	}
}

// Stop halts the polling loop.
func (vm *ViewModel) Stop() {
	vm.poller.Stop()
	vm.stopMu.Lock()
	if vm.stopCtx != nil {
		vm.stopCtx()
	}
	vm.stopMu.Unlock()
}

// handleUpdate is called by the poller for every incoming Telegram update.
func (vm *ViewModel) handleUpdate(update msgmodel.Update) {
	if update.Message == nil {
		return
	}
	msg := update.Message

	if msg.From == nil || !vm.guard.IsAllowed(msg.From.ID) {
		return // silently ignore unauthorised senders
	}

	// If the agent is waiting for a manual-focus confirmation from this chat,
	// consume the message as the confirmation instead of routing it normally.
	if vm.deliverPendingConfirmation(msg.Chat.ID, msg.Text) {
		return
	}

	route := vm.router.Dispatch(msg)

	switch route.Type {
	case routervm.RouteConfirmYes:
		vm.handleConfirmYes(msg.Chat.ID)

	case routervm.RouteConfirmNo:
		vm.guard.ClearPending()
		vm.send(msg.Chat.ID, tgfmt.OKWith("Action cancelled."))

	case routervm.RouteStop:
		vm.stopMu.Lock()
		if vm.stopCtx != nil {
			vm.stopCtx()
			vm.stopCtx = nil
		}
		vm.stopMu.Unlock()
		vm.send(msg.Chat.ID, tgfmt.OKWith("Task stopped."))

	case routervm.RouteManual:
		result := vm.cmdExec.Execute(route.Command)
		if result.ImageData != nil {
			if err := vm.tg.SendPhoto(msg.Chat.ID, result.ImageData, result.Text); err != nil {
				log.Printf("sendPhoto error: %v", err)
			}
		} else {
			vm.send(msg.Chat.ID, result.Text)
		}

	case routervm.RouteAIAgent:
		if route.Text == "" {
			return
		}
		go vm.runAgent(msg.Chat.ID, route.Text)
	}
}

// handleConfirmYes executes the pending confirmed action.
func (vm *ViewModel) handleConfirmYes(chatID int64) {
	result, had, err := vm.guard.ConfirmPending()
	if !had {
		vm.send(chatID, "No pending action to confirm.")
		return
	}
	if err != nil {
		vm.send(chatID, tgfmt.Fail(err))
		return
	}
	vm.send(chatID, tgfmt.OKWith(result))
}

// runAgent executes an AI task in a goroutine, streaming progress to Telegram.
func (vm *ViewModel) runAgent(chatID int64, task string) {
	ctx, cancel := context.WithCancel(context.Background())
	vm.stopMu.Lock()
	vm.stopCtx = cancel
	vm.stopMu.Unlock()
	defer cancel()

	vm.send(chatID, "⏳ Working...")

	// Photo callback: when the AI calls take_screenshot, the PNG is sent
	// straight to this chat. The image bytes never reach the AI provider.
	sendPhoto := func(data []byte, caption string) error {
		return vm.tg.SendPhoto(chatID, data, caption)
	}

	// Manual-focus callback: when the AI calls request_manual_focus, the
	// bot sends the user a prompt and blocks on this function until the
	// user replies with any message (or the 60s timeout fires).
	manualFocus := func(reason string) (string, error) {
		vm.send(chatID, "⚠️ Manual help needed: "+reason+"\n\nPlease click the target input now, then reply with any message (e.g. 'ok') to continue.")
		return vm.waitForConfirmation(ctx, chatID, 60*time.Second)
	}

	// Intentionally silent: no per-tool progress messages.
	result, err := vm.agent.Run(ctx, task, nil, sendPhoto, manualFocus)
	if err != nil {
		if ctx.Err() != nil {
			return // user stopped the task
		}
		vm.send(chatID, tgfmt.Fail(err))
		return
	}

	vm.send(chatID, tgfmt.AgentDone(result))
}

// waitForConfirmation registers a pending channel for chatID and blocks
// until the next user message arrives or the timeout fires.
func (vm *ViewModel) waitForConfirmation(ctx context.Context, chatID int64, timeout time.Duration) (string, error) {
	ch := make(chan string, 1)

	vm.pendingMu.Lock()
	vm.pending[chatID] = ch
	vm.pendingMu.Unlock()

	defer func() {
		vm.pendingMu.Lock()
		delete(vm.pending, chatID)
		vm.pendingMu.Unlock()
	}()

	select {
	case reply := <-ch:
		return reply, nil
	case <-time.After(timeout):
		return "", fmt.Errorf("manual-focus confirmation timed out after %s", timeout)
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

// deliverPendingConfirmation checks whether a chat is waiting for a manual
// confirmation and, if so, forwards the message text to that channel.
// Returns true if the message was consumed as a confirmation.
func (vm *ViewModel) deliverPendingConfirmation(chatID int64, text string) bool {
	vm.pendingMu.Lock()
	ch, ok := vm.pending[chatID]
	vm.pendingMu.Unlock()
	if !ok {
		return false
	}
	select {
	case ch <- text:
	default:
	}
	return true
}

// send is a convenience wrapper that logs errors.
func (vm *ViewModel) send(chatID int64, text string) {
	if text == "" {
		return
	}
	if err := vm.tg.SendMessage(chatID, text); err != nil {
		log.Printf("sendMessage error: %v", err)
	}
}

// buildProvider constructs the AI provider from config.
func buildProvider(cfg *configmodel.Config) (aisvc.Provider, error) {
	switch cfg.AIProvider {
	case "claude":
		return claude.New(cfg.AIAPIKey, cfg.AIModel), nil
	case "openai":
		return openai.New(cfg.AIAPIKey, cfg.AIModel, cfg.AIBaseURL), nil
	case "gemini":
		return gemini.New(cfg.AIAPIKey, cfg.AIModel), nil
	case "deepseek":
		return deepseek.New(cfg.AIAPIKey, cfg.AIModel), nil
	case "grok":
		return grok.New(cfg.AIAPIKey, cfg.AIModel), nil
	case "local":
		// Built-in rule-based parser: zero external dependency, zero cost.
		return local.New(), nil
	case "ollama":
		// Local LLM via Ollama. Developer may override base URL and model.
		return ollama.New(cfg.AIBaseURL, cfg.AIModel), nil
	default:
		return nil, fmt.Errorf("unknown AI provider: %q", cfg.AIProvider)
	}
}

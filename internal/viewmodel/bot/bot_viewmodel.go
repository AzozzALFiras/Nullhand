package bot

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/iamakillah/Nullhand_Linux/internal/audit"
	auth "github.com/iamakillah/Nullhand_Linux/internal/auth"
	cmdmodel "github.com/iamakillah/Nullhand_Linux/internal/model/command"
	configmodel "github.com/iamakillah/Nullhand_Linux/internal/model/config"
	msgmodel "github.com/iamakillah/Nullhand_Linux/internal/model/message"
	reciperepo "github.com/iamakillah/Nullhand_Linux/internal/repository/recipe"
	"github.com/iamakillah/Nullhand_Linux/internal/safety"
	"github.com/iamakillah/Nullhand_Linux/internal/scheduler"
	aisvc "github.com/iamakillah/Nullhand_Linux/internal/service/ai"
	"github.com/iamakillah/Nullhand_Linux/internal/service/ai/claude"
	"github.com/iamakillah/Nullhand_Linux/internal/service/ai/deepseek"
	"github.com/iamakillah/Nullhand_Linux/internal/service/ai/gemini"
	"github.com/iamakillah/Nullhand_Linux/internal/service/ai/grok"
	"github.com/iamakillah/Nullhand_Linux/internal/service/ai/local"
	"github.com/iamakillah/Nullhand_Linux/internal/service/ai/ollama"
	"github.com/iamakillah/Nullhand_Linux/internal/service/ai/openai"
	filetransfer "github.com/iamakillah/Nullhand_Linux/internal/service/linux/filetransfer"
	ocrsvc "github.com/iamakillah/Nullhand_Linux/internal/service/linux/ocr"
	recipesvc "github.com/iamakillah/Nullhand_Linux/internal/service/recipe"
	tgsvc "github.com/iamakillah/Nullhand_Linux/internal/service/telegram"
	tgfmt "github.com/iamakillah/Nullhand_Linux/internal/view/telegram"
	agentvm "github.com/iamakillah/Nullhand_Linux/internal/viewmodel/agent"
	cmdvm "github.com/iamakillah/Nullhand_Linux/internal/viewmodel/command"
	menuvm "github.com/iamakillah/Nullhand_Linux/internal/viewmodel/menu"
	routervm "github.com/iamakillah/Nullhand_Linux/internal/viewmodel/router"
	sessionvm "github.com/iamakillah/Nullhand_Linux/internal/viewmodel/session"
)

// ViewModel is the top-level orchestrator: it wires the poller, router,
// command handler, and AI agent together.
type ViewModel struct {
	cfg       *configmodel.Config
	tg        *tgsvc.Client
	poller    *tgsvc.Poller
	guard     *safety.Guard
	router    *routervm.ViewModel
	cmdExec   *cmdvm.ViewModel
	agent     *agentvm.ViewModel
	menu      *menuvm.ViewModel
	session   *sessionvm.Manager
	otp       *auth.OTPGate
	audit     *audit.Logger
	scheduler *scheduler.Scheduler

	// pendingDownloads holds file receive sessions waiting for destination choice.
	pendingDownloads   map[int64]*filetransfer.PendingDownload
	pendingDownloadsMu sync.Mutex

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

	auditLog, err := audit.New()
	if err != nil {
		// Audit logging is non-fatal: warn and continue with a no-op logger.
		log.Printf("warning: could not open audit log: %v", err)
		auditLog = nil
	}

	vm := &ViewModel{
		cfg:              cfg,
		tg:               tgClient,
		guard:            safety.New(cfg.AllowedUserID),
		router:           routervm.New(),
		cmdExec:          cmdvm.New(),
		agent:            agentvm.New(aiProvider, recipes),
		menu:             menuvm.New(),
		session:          sessionvm.NewManager(),
		otp:              auth.NewOTPGate(),
		audit:            auditLog,
		scheduler:        scheduler.New(),
		pendingDownloads: make(map[int64]*filetransfer.PendingDownload),
		pending:          make(map[int64]chan string),
	}

	// Wire OTP print callback so new codes are printed to terminal.
	vm.otp.StartExpiry(func(newCode string) {
		vm.otp.PrintCurrentCode()
	})
	vm.otp.PrintCurrentCode() // print initial code

	vm.scheduler.Start()
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
		{Command: "help", Description: "Show help message"},
		{Command: "screenshot", Description: "Capture the screen"},
		{Command: "status", Description: "CPU, memory, active app"},
		{Command: "ocr", Description: "Read text from screen"},
		{Command: "shell", Description: "Run a shell command"},
		{Command: "type", Description: "Type text on screen"},
		{Command: "click", Description: "Click at coordinates"},
		{Command: "key", Description: "Press a key shortcut"},
		{Command: "open", Description: "Open an application"},
		{Command: "paste", Description: "Get clipboard content"},
		{Command: "copy", Description: "Set clipboard content"},
		{Command: "ls", Description: "List directory contents"},
		{Command: "read", Description: "Read a file"},
		{Command: "apps", Description: "List running apps"},
		{Command: "schedule", Description: "Manage scheduled tasks"},
		{Command: "menu", Description: "Show quick action toolbar"},
		{Command: "stop", Description: "Stop current AI task"},
	}
}

// Stop halts the polling loop, scheduler, and audit log.
func (vm *ViewModel) Stop() {
	vm.poller.Stop()
	vm.scheduler.Stop()
	vm.stopMu.Lock()
	if vm.stopCtx != nil {
		vm.stopCtx()
	}
	vm.stopMu.Unlock()
	if vm.audit != nil {
		vm.audit.Close()
	}
}

// auditLog writes an audit line silently — never crashes on failure.
func (vm *ViewModel) auditLog(userID int64, action string, extras ...string) {
	if vm.audit == nil {
		return
	}
	_ = vm.audit.Log(userID, action, extras...)
}

// handleUpdate is called by the poller for every incoming Telegram update.
func (vm *ViewModel) handleUpdate(update msgmodel.Update) {
	// Handle inline keyboard button presses.
	if update.CallbackQuery != nil {
		vm.handleCallback(update.CallbackQuery)
		return
	}

	if update.Message == nil {
		return
	}
	msg := update.Message

	if msg.From == nil || !vm.guard.IsAllowed(msg.From.ID) {
		return // silently ignore unauthorised senders
	}

	// OTP gate: must be unlocked before any commands are processed.
	if !vm.otp.IsUnlocked() {
		if vm.otp.TryUnlock(strings.TrimSpace(msg.Text)) {
			vm.auditLog(msg.From.ID, "otp_unlock")
			vm.send(msg.Chat.ID, "✅ Session unlocked. Welcome.")
			log.Println("OTP authentication successful — session unlocked")
		} else {
			vm.send(msg.Chat.ID, "🔒 Bot is locked. Enter the OTP shown in the terminal.")
		}
		return
	}

	// OCR trigger detection — checked before AI routing to avoid token spend.
	if isOCRTrigger(msg.Text) {
		vm.auditLog(msg.From.ID, "ocr")
		go vm.runOCR(msg.Chat.ID)
		return
	}

	// Schedule NL detection — before AI routing.
	if hour, minute, label, action := vm.parseScheduleNL(msg.Text, msg.Chat.ID, msg.From.ID); action != nil {
		id := vm.scheduler.Add(msg.Chat.ID, label, hour, minute, action)
		vm.auditLog(msg.From.ID, "schedule_create", fmt.Sprintf(`id=%q`, id))
		vm.send(msg.Chat.ID, fmt.Sprintf(
			"✅ Scheduled: %s\n🕐 Every day at %02d:%02d\n🆔 ID: %s\nUse /schedule list to see all tasks.",
			label, hour, minute, id,
		))
		return
	}

	// File send detection for natural language ("send me /path", "upload /path", etc).
	textLower := strings.ToLower(strings.TrimSpace(msg.Text))
	if (strings.Contains(textLower, "send") || strings.Contains(textLower, "upload")) && strings.Contains(msg.Text, "/") {
		parts := strings.Fields(msg.Text)
		for _, p := range parts {
			if strings.HasPrefix(p, "/") {
				vm.auditLog(msg.From.ID, "file_send", fmt.Sprintf(`path=%q`, p))
				go filetransfer.SendFile(vm.tg, msg.Chat.ID, p)
				return
			}
		}
	}

	// File receive from Telegram (document, photo, video, audio)
	if msg.Document != nil {
		d := msg.Document
		vm.auditLog(msg.From.ID, "file_receive", fmt.Sprintf(`filename=%q`, d.FileName))
		vm.startFileReceive(msg.Chat.ID, d.FileID, d.FileName, d.MimeType)
		return
	}
	if len(msg.Photo) > 0 {
		p := msg.Photo[len(msg.Photo)-1] // largest photo
		vm.auditLog(msg.From.ID, "file_receive", `filename="photo.jpg"`)
		vm.startFileReceive(msg.Chat.ID, p.FileID, "photo.jpg", "image/jpeg")
		return
	}
	if msg.Video != nil {
		v := msg.Video
		vm.auditLog(msg.From.ID, "file_receive", fmt.Sprintf(`filename=%q`, v.FileName))
		vm.startFileReceive(msg.Chat.ID, v.FileID, v.FileName, v.MimeType)
		return
	}
	if msg.Audio != nil {
		a := msg.Audio
		vm.auditLog(msg.From.ID, "file_receive", fmt.Sprintf(`filename=%q`, a.FileName))
		vm.startFileReceive(msg.Chat.ID, a.FileID, a.FileName, a.MimeType)
		return
	}

	// Custom path for pending download
	vm.pendingDownloadsMu.Lock()
	pendingDownload, hasPendingDownload := vm.pendingDownloads[msg.Chat.ID]
	vm.pendingDownloadsMu.Unlock()
	if hasPendingDownload {
		destDir := strings.TrimSpace(msg.Text)
		vm.pendingDownloadsMu.Lock()
		delete(vm.pendingDownloads, msg.Chat.ID)
		vm.pendingDownloadsMu.Unlock()

		if err := filetransfer.DownloadAndSave(vm.tg, msg.Chat.ID, pendingDownload.FileID, pendingDownload.Filename, destDir); err != nil {
			vm.send(msg.Chat.ID, fmt.Sprintf("❌ Failed to save: %v", err))
		} else {
			vm.send(msg.Chat.ID, fmt.Sprintf("✅ Saved to %s", destDir))
		}
		return
	}

	// If the agent is waiting for a manual-focus confirmation from this chat,
	// consume the message as the confirmation instead of routing it normally.
	if vm.deliverPendingConfirmation(msg.Chat.ID, msg.Text) {
		return
	}

	// Reply keyboard button text detection (non-slash persistent toolbar buttons).
	chatID := msg.Chat.ID
	userID := msg.From.ID
	switch strings.TrimSpace(msg.Text) {
	case "🐚 Run Command":
		vm.auditLog(userID, "shell")
		vm.send(chatID, "🐚 Enter the shell command:")
		vm.pendingMu.Lock()
		vm.pending[chatID] = make(chan string, 1)
		vm.pendingMu.Unlock()
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()
			reply, err := vm.waitForConfirmation(ctx, chatID, 5*time.Minute)
			if err != nil || reply == "" {
				return
			}
			vm.auditLog(userID, "shell", fmt.Sprintf(`cmd=%q`, reply))
			workingMsgID, _ := vm.sendWorking(chatID)
			result := vm.cmdExec.Execute(&cmdmodel.Command{Name: "shell", Args: strings.Fields(reply)})
			if workingMsgID > 0 {
				_ = vm.tg.EditMessage(chatID, workingMsgID, result.Text, nil)
			} else {
				vm.send(chatID, result.Text)
			}
		}()
		return
	case "📤 Send File":
		vm.auditLog(userID, "file_send")
		vm.send(chatID, "📤 Enter the full file path to send:")
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()
			reply, err := vm.waitForConfirmation(ctx, chatID, 5*time.Minute)
			if err != nil || reply == "" {
				return
			}
			path := strings.TrimSpace(reply)
			vm.auditLog(userID, "file_send", fmt.Sprintf(`path=%q`, path))
			_ = filetransfer.SendFile(vm.tg, chatID, path)
		}()
		return
	case "🔒 Lock Bot":
		vm.auditLog(userID, "otp_lock")
		vm.otp.Lock()
		vm.send(chatID, "🔒 Bot locked. Enter new OTP to unlock.")
		return
	}

	// If a menu is active and user sends a number, treat it as menu selection.
	if vm.menu.IsActive(msg.Chat.ID) {
		text := strings.TrimSpace(msg.Text)
		var num int
		if _, err := fmt.Sscanf(text, "%d", &num); err == nil && num > 0 {
			if err := vm.menu.HandleNumberSelection(vm.tg, msg.Chat.ID, num); err != nil {
				log.Printf("menu number selection error: %v", err)
			}
			return
		}
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
		if route.Command.Name == "schedule" {
			vm.handleScheduleCommand(msg.Chat.ID, msg.From.ID, route.Command.Args)
			return
		}
		// /ocr is handled directly here — no cmdExec involvement.
		if route.Command.Name == "ocr" {
			vm.auditLog(msg.From.ID, "ocr")
			go vm.runOCR(msg.Chat.ID)
			return
		}
		// /menu sends the inline toolbar directly.
		if route.Command.Name == "menu" {
			vm.auditLog(msg.From.ID, "menu")
			if err := SendMenu(vm.tg, msg.Chat.ID); err != nil {
				vm.send(msg.Chat.ID, "❌ Failed to send menu")
			}
			return
		}
		// Audit the manual command before executing it.
		switch route.Command.Name {
		case "screenshot":
			vm.auditLog(msg.From.ID, "screenshot")
		case "shell":
			vm.auditLog(msg.From.ID, "shell", fmt.Sprintf(`cmd=%q`, strings.Join(route.Command.Args, " ")))
		case "open":
			vm.auditLog(msg.From.ID, "app_open", fmt.Sprintf(`app=%q`, strings.Join(route.Command.Args, " ")))
		case "paste":
			vm.auditLog(msg.From.ID, "clipboard")
		case "status":
			vm.auditLog(msg.From.ID, "sysinfo")
		default:
			vm.auditLog(msg.From.ID, route.Command.Name)
		}
		workingMsgID, _ := vm.sendWorking(msg.Chat.ID)
		result := vm.cmdExec.Execute(route.Command)
		if workingMsgID > 0 {
			if result.ImageData != nil {
				_ = vm.tg.DeleteMessage(msg.Chat.ID, workingMsgID)
				if err := vm.tg.SendPhoto(msg.Chat.ID, result.ImageData, result.Text); err != nil {
					log.Printf("sendPhoto error: %v", err)
				}
			} else {
				_ = vm.tg.EditMessage(msg.Chat.ID, workingMsgID, result.Text, nil)
			}
		} else {
			if result.ImageData != nil {
				if err := vm.tg.SendPhoto(msg.Chat.ID, result.ImageData, result.Text); err != nil {
					log.Printf("sendPhoto error: %v", err)
				}
			} else {
				vm.send(msg.Chat.ID, result.Text)
			}
		}

	case routervm.RouteAIAgent:
		if route.Text == "" {
			return
		}
		input := route.Text
		if len(input) > 80 {
			input = input[:80]
		}
		vm.auditLog(msg.From.ID, "natural_language", fmt.Sprintf(`input=%q`, input))
		go vm.runAgent(msg.Chat.ID, msg.From.ID, route.Text)
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
func (vm *ViewModel) runAgent(chatID int64, userID int64, task string) {
	ctx, cancel := context.WithCancel(context.Background())
	vm.stopMu.Lock()
	vm.stopCtx = cancel
	vm.stopMu.Unlock()
	defer cancel()

	workingMsgID, _ := vm.sendWorking(chatID)

	// Set session context on the local provider so it can handle
	// context-dependent commands like bare "ls" in terminal mode.
	if localProvider, ok := vm.agent.Provider().(*local.Provider); ok {
		if sess := vm.session.Get(chatID); sess != nil {
			localProvider.SetSessionContext(sess.ActiveApp, sess.ActiveMode)
		} else {
			localProvider.SetSessionContext("", "")
		}
	}

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

	// Browse callback: when the AI calls browse_folder, open the interactive
	// file browser with inline keyboard buttons.
	browse := func(path string) {
		vm.BrowseFolder(chatID, path)
	}

	// Intentionally silent: no per-tool progress messages.
	result, err := vm.agent.Run(ctx, task, nil, sendPhoto, manualFocus, browse)

	// Update session context based on what tools were executed.
	for _, tc := range vm.agent.LastToolCalls() {
		app, mode := sessionvm.InferContextFromAction(tc.ToolName, tc.Arguments, "")
		if app != "" && mode != "" {
			vm.session.Set(chatID, app, mode, "")
			break // use the last meaningful action
		}
	}

	if err != nil {
		if ctx.Err() != nil {
			return // user stopped the task
		}
		if workingMsgID > 0 {
			_ = vm.tg.EditMessage(chatID, workingMsgID, tgfmt.Fail(err), nil)
		} else {
			vm.send(chatID, tgfmt.Fail(err))
		}
		return
	}

	if workingMsgID > 0 {
		_ = vm.tg.EditMessage(chatID, workingMsgID, tgfmt.AgentDone(result), nil)
	} else {
		vm.send(chatID, tgfmt.AgentDone(result))
	}
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

// sendWorking sends "⏳ Working..." immediately and returns the messageID so it can be edited later.
// Also sends the typing action for immediate UI feedback.
func (vm *ViewModel) sendWorking(chatID int64) (int, error) {
	_ = vm.tg.SendTyping(chatID)
	return vm.tg.SendMessageWithKeyboard(chatID, "⏳ Working...", nil)
}

// handleCallback processes inline keyboard button presses.
func (vm *ViewModel) handleCallback(cb *msgmodel.CallbackQuery) {
	if cb.From == nil || !vm.guard.IsAllowed(cb.From.ID) {
		return
	}

	if strings.HasPrefix(cb.Data, "save|") {
		vm.handleSaveCallback(cb)
		return
	}

	if strings.HasPrefix(cb.Data, "menu:") {
		vm.handleMenuCallback(cb)
		return
	}

	chatID := cb.Message.Chat.ID

	if err := vm.menu.HandleBrowseCallback(vm.tg, chatID, cb.ID, cb.Data); err != nil {
		log.Printf("menu callback error: %v", err)
	}
}

// handleMenuCallback handles toolbar button presses (menu:* callbacks).
func (vm *ViewModel) handleMenuCallback(cb *msgmodel.CallbackQuery) {
	chatID := cb.Message.Chat.ID
	userID := cb.From.ID
	_ = vm.tg.AnswerCallbackQuery(cb.ID, "")

	switch cb.Data {
	case "menu:screenshot":
		vm.auditLog(userID, "screenshot")
		workingMsgID, _ := vm.sendWorking(chatID)
		result := vm.cmdExec.Execute(&cmdmodel.Command{Name: "screenshot"})
		if workingMsgID > 0 && result.ImageData != nil {
			_ = vm.tg.DeleteMessage(chatID, workingMsgID)
			_ = vm.tg.SendPhoto(chatID, result.ImageData, "")
		} else if result.ImageData != nil {
			_ = vm.tg.SendPhoto(chatID, result.ImageData, "")
		} else {
			_ = vm.tg.EditMessage(chatID, workingMsgID, result.Text, nil)
		}

	case "menu:sysinfo":
		vm.auditLog(userID, "sysinfo")
		workingMsgID, _ := vm.sendWorking(chatID)
		result := vm.cmdExec.Execute(&cmdmodel.Command{Name: "status"})
		if workingMsgID > 0 {
			_ = vm.tg.EditMessage(chatID, workingMsgID, result.Text, nil)
		} else {
			vm.send(chatID, result.Text)
		}

	case "menu:clipboard":
		vm.auditLog(userID, "clipboard")
		workingMsgID, _ := vm.sendWorking(chatID)
		result := vm.cmdExec.Execute(&cmdmodel.Command{Name: "paste"})
		if workingMsgID > 0 {
			_ = vm.tg.EditMessage(chatID, workingMsgID, result.Text, nil)
		} else {
			vm.send(chatID, result.Text)
		}

	case "menu:shell":
		vm.auditLog(userID, "shell")
		vm.send(chatID, "🐚 Enter the shell command:")
		vm.pendingMu.Lock()
		vm.pending[chatID] = make(chan string, 1)
		vm.pendingMu.Unlock()
		// Next message will be delivered via deliverPendingConfirmation;
		// we launch a goroutine to wait for it and execute.
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()
			reply, err := vm.waitForConfirmation(ctx, chatID, 5*time.Minute)
			if err != nil || reply == "" {
				return
			}
			vm.auditLog(userID, "shell", fmt.Sprintf(`cmd=%q`, reply))
			workingMsgID, _ := vm.sendWorking(chatID)
			result := vm.cmdExec.Execute(&cmdmodel.Command{Name: "shell", Args: strings.Fields(reply)})
			if workingMsgID > 0 {
				_ = vm.tg.EditMessage(chatID, workingMsgID, result.Text, nil)
			} else {
				vm.send(chatID, result.Text)
			}
		}()

	case "menu:sendfile":
		vm.auditLog(userID, "file_send")
		vm.send(chatID, "📤 Enter the full file path to send:")
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()
			reply, err := vm.waitForConfirmation(ctx, chatID, 5*time.Minute)
			if err != nil || reply == "" {
				return
			}
			path := strings.TrimSpace(reply)
			vm.auditLog(userID, "file_send", fmt.Sprintf(`path=%q`, path))
			_ = filetransfer.SendFile(vm.tg, chatID, path)
		}()

	case "menu:downloads":
		vm.auditLog(userID, "downloads")
		workingMsgID, _ := vm.sendWorking(chatID)
		result := vm.cmdExec.Execute(&cmdmodel.Command{Name: "ls", Args: []string{"~/Downloads"}})
		if workingMsgID > 0 {
			_ = vm.tg.EditMessage(chatID, workingMsgID, result.Text, nil)
		} else {
			vm.send(chatID, result.Text)
		}

	case "menu:ocr":
		vm.auditLog(userID, "ocr")
		go vm.runOCR(chatID)

	case "menu:lock":
		vm.auditLog(userID, "otp_lock")
		vm.otp.Lock()
		vm.send(chatID, "🔒 Bot locked. Enter new OTP to unlock.")

	case "menu:help":
		vm.auditLog(userID, "menu")
		helpText := `🤖 *Nullhand — Linux Agent*

I understand natural language. Just tell me what to do:

"take a screenshot"
"what's my CPU usage"
"open Firefox"
"type Hello World"
"send me /home/user/file.pdf"
"run git status in terminal"

Or use /menu for quick actions.`
		vm.send(chatID, helpText)

	default:
		log.Printf("handleMenuCallback: unknown action %q", cb.Data)
	}
}

// BrowseFolder opens the interactive file browser for a path.
func (vm *ViewModel) BrowseFolder(chatID int64, path string) {
	if err := vm.menu.BrowsePath(vm.tg, chatID, path); err != nil {
		vm.send(chatID, fmt.Sprintf("❌ %v", err))
	}
}

// startFileReceive sends the destination picker keyboard and stores the pending download state.
func (vm *ViewModel) startFileReceive(chatID int64, fileID, filename, mimeType string) {
	if filename == "" {
		filename = "file"
	}

	vm.pendingDownloadsMu.Lock()
	vm.pendingDownloads[chatID] = &filetransfer.PendingDownload{
		FileID:   fileID,
		Filename: filename,
		MimeType: mimeType,
	}
	vm.pendingDownloadsMu.Unlock()

	keyboard := &tgsvc.InlineKeyboardMarkup{
		InlineKeyboard: [][]tgsvc.InlineKeyboardButton{
			{
				{Text: "🏠 Home", CallbackData: "save|home|" + fileID + "|" + filename},
				{Text: "🖥️ Desktop", CallbackData: "save|desktop|" + fileID + "|" + filename},
			},
			{
				{Text: "📥 Downloads", CallbackData: "save|downloads|" + fileID + "|" + filename},
				{Text: "✏️ Custom path", CallbackData: "save|custom|" + fileID + "|" + filename},
			},
		},
	}

	text := fmt.Sprintf(`📥 Where should I save "%s"?`, filename)
	_, _ = vm.tg.SendMessageWithKeyboard(chatID, text, keyboard)
}

// handleSaveCallback processes "save|..." callback data from the destination picker.
func (vm *ViewModel) handleSaveCallback(cb *msgmodel.CallbackQuery) {
	parts := strings.Split(cb.Data, "|")
	if len(parts) < 4 {
		_ = vm.tg.AnswerCallbackQuery(cb.ID, "Invalid callback")
		return
	}

	action := parts[1]
	fileID := parts[2]
	filename := parts[3]
	chatID := cb.Message.Chat.ID

	vm.pendingDownloadsMu.Lock()
	pending, ok := vm.pendingDownloads[chatID]
	vm.pendingDownloadsMu.Unlock()

	if !ok || pending.FileID != fileID {
		_ = vm.tg.AnswerCallbackQuery(cb.ID, "Session expired. Send the file again.")
		return
	}

	var destDir string
	switch action {
	case "home":
		home, _ := os.UserHomeDir()
		destDir = home
	case "desktop":
		home, _ := os.UserHomeDir()
		destDir = filepath.Join(home, "Desktop")
	case "downloads":
		home, _ := os.UserHomeDir()
		destDir = filepath.Join(home, "Downloads")
	case "custom":
		_ = vm.tg.AnswerCallbackQuery(cb.ID, "Enter path")
		vm.send(chatID, "📁 Enter the full destination path (e.g. /home/user/documents/):")
		return
	default:
		_ = vm.tg.AnswerCallbackQuery(cb.ID, "Unknown destination")
		return
	}

	vm.pendingDownloadsMu.Lock()
	delete(vm.pendingDownloads, chatID)
	vm.pendingDownloadsMu.Unlock()

	if err := filetransfer.DownloadAndSave(vm.tg, chatID, fileID, filename, destDir); err != nil {
		vm.send(chatID, fmt.Sprintf("❌ Failed to save: %v", err))
	} else {
		vm.send(chatID, fmt.Sprintf("✅ Saved to %s/%s", destDir, filename))
	}
	_ = vm.tg.AnswerCallbackQuery(cb.ID, "Saved!")
}

// ── Scheduler helpers ────────────────────────────────────────────────────────

// handleScheduleCommand processes /schedule list|cancel <id>|clear.
func (vm *ViewModel) handleScheduleCommand(chatID int64, userID int64, args []string) {
	if len(args) == 0 {
		vm.send(chatID, "Usage: /schedule list | /schedule cancel [id] | /schedule clear")
		return
	}
	switch args[0] {
	case "list":
		tasks := vm.scheduler.List()
		if len(tasks) == 0 {
			vm.send(chatID, "📋 No scheduled tasks.")
			return
		}
		var sb strings.Builder
		sb.WriteString("📋 Active scheduled tasks:\n")
		for _, t := range tasks {
			sb.WriteString(fmt.Sprintf("🆔 %s — %s — every day at %02d:%02d\n", t.ID, t.Label, t.Hour, t.Minute))
		}
		sb.WriteString("\nUse /schedule cancel [id] to remove a task.")
		vm.send(chatID, sb.String())
	case "cancel":
		if len(args) < 2 {
			vm.send(chatID, "Usage: /schedule cancel [id]")
			return
		}
		id := args[1]
		if vm.scheduler.Cancel(id) {
			vm.auditLog(userID, "schedule_cancel", fmt.Sprintf(`id=%q`, id))
			vm.send(chatID, fmt.Sprintf("✅ Cancelled %s", id))
		} else {
			vm.send(chatID, fmt.Sprintf("❌ No task with ID %s", id))
		}
	case "clear":
		vm.scheduler.Clear()
		vm.auditLog(userID, "schedule_cancel", `id="all"`)
		vm.send(chatID, "✅ All tasks cleared.")
	default:
		vm.send(chatID, "Usage: /schedule list | /schedule cancel [id] | /schedule clear")
	}
}

// parseScheduleNL tries to extract a schedule from a natural language message.
// Returns (hour, minute, label, actionFunc) on success; actionFunc is nil if parsing fails.
func (vm *ViewModel) parseScheduleNL(text string, chatID int64, userID int64) (int, int, string, func()) {
	lower := strings.ToLower(strings.TrimSpace(text))

	// Must look like a schedule request.
	isSchedule := strings.Contains(lower, "every day at") ||
		strings.Contains(lower, "every") && strings.Contains(lower, "at") ||
		strings.Contains(lower, "schedule") ||
		strings.Contains(lower, "remind me to") ||
		strings.Contains(lower, "run") && strings.Contains(lower, "every day at")
	if !isSchedule {
		return 0, 0, "", nil
	}

	// Extract time token (e.g. "8am", "8:30am", "14:00", "9pm").
	fields := strings.Fields(lower)
	hour, minute, timeIdx := -1, 0, -1
	for i, f := range fields {
		if h, m, ok := parseTimeToken(f); ok {
			hour, minute, timeIdx = h, m, i
			break
		}
	}
	if timeIdx < 0 {
		return 0, 0, "", nil
	}

	// Everything after the time token is the action description.
	actionWords := fields[timeIdx+1:]
	if len(actionWords) == 0 {
		return 0, 0, "", nil
	}
	actionText := strings.Join(actionWords, " ")

	// Map action description to an actual function.
	action, label := vm.resolveScheduledAction(actionText, chatID, userID, "")
	if action == nil {
		return 0, 0, "", nil
	}
	return hour, minute, label, action
}

// resolveScheduledAction maps a natural language action string to a func() and label.
func (vm *ViewModel) resolveScheduledAction(action string, chatID int64, userID int64, taskID string) (func(), string) {
	lower := strings.ToLower(strings.TrimSpace(action))

	logAndRun := func(label string, fn func()) func() {
		return func() {
			vm.auditLog(userID, "scheduled_task", fmt.Sprintf(`id=%q`, taskID))
			fn()
		}
	}

	switch {
	case strings.Contains(lower, "screenshot"):
		return logAndRun("screenshot", func() {
			result := vm.cmdExec.Execute(&cmdmodel.Command{Name: "screenshot"})
			if result.ImageData != nil {
				_ = vm.tg.SendPhoto(chatID, result.ImageData, "📸 Scheduled screenshot")
			}
		}), "screenshot"

	case strings.Contains(lower, "system info") || strings.Contains(lower, "sysinfo") || strings.Contains(lower, "cpu") || strings.Contains(lower, "status"):
		return logAndRun("sysinfo", func() {
			result := vm.cmdExec.Execute(&cmdmodel.Command{Name: "status"})
			vm.send(chatID, result.Text)
		}), "sysinfo"

	case strings.Contains(lower, "read") && strings.Contains(lower, "screen") || lower == "ocr":
		return logAndRun("ocr", func() {
			vm.runOCR(chatID)
		}), "read screen"

	case strings.Contains(lower, "run ") || strings.Contains(lower, "shell "):
		// Extract command after "run " or "shell "
		cmd := strings.TrimPrefix(lower, "run ")
		cmd = strings.TrimPrefix(cmd, "shell ")
		return logAndRun("shell: "+cmd, func() {
			result := vm.cmdExec.Execute(&cmdmodel.Command{Name: "shell", Args: strings.Fields(cmd)})
			vm.send(chatID, result.Text)
		}), "shell: " + cmd

	case strings.Contains(lower, "send") && strings.Contains(action, "/"):
		// Extract path
		for _, w := range strings.Fields(action) {
			if strings.HasPrefix(w, "/") {
				path := w
				return logAndRun("file_send: "+path, func() {
					_ = filetransfer.SendFile(vm.tg, chatID, path)
				}), "send " + path
			}
		}
		return nil, ""

	default:
		return nil, ""
	}
}

// parseTimeToken parses strings like "8am", "8:30am", "14:00", "9pm" into (hour, minute, ok).
func parseTimeToken(s string) (int, int, bool) {
	s = strings.ToLower(strings.TrimSpace(s))

	isPM := strings.HasSuffix(s, "pm")
	isAM := strings.HasSuffix(s, "am")
	if isPM || isAM {
		s = s[:len(s)-2]
	}

	var hour, minute int
	if strings.Contains(s, ":") {
		var h, m int
		if _, err := fmt.Sscanf(s, "%d:%d", &h, &m); err != nil {
			return 0, 0, false
		}
		hour, minute = h, m
	} else {
		var h int
		if _, err := fmt.Sscanf(s, "%d", &h); err != nil {
			return 0, 0, false
		}
		hour = h
	}

	if hour < 0 || hour > 23 || minute < 0 || minute > 59 {
		return 0, 0, false
	}

	if isPM && hour != 12 {
		hour += 12
	}
	if isAM && hour == 12 {
		hour = 0
	}

	return hour, minute, true
}

// runOCR captures the screen, runs Tesseract, and edits the working message with results.
func (vm *ViewModel) runOCR(chatID int64) {
	workingMsgID, _ := vm.sendWorking(chatID)

	text, err := ocrsvc.ReadScreen()

	var reply string
	switch {
	case err == ocrsvc.ErrNotInstalled:
		reply = "❌ Tesseract is not installed. Run: sudo apt install tesseract-ocr"
	case err != nil:
		reply = fmt.Sprintf("❌ OCR failed: %v", err)
	case text == "":
		reply = "🔍 No text found on screen."
	default:
		reply = "🔍 Screen text:\n```\n" + text + "\n```"
	}

	if workingMsgID > 0 {
		_ = vm.tg.EditMessage(chatID, workingMsgID, reply, nil)
	} else {
		vm.send(chatID, reply)
	}
}

// isOCRTrigger reports whether the message text is an OCR request.
func isOCRTrigger(text string) bool {
	lower := strings.ToLower(strings.TrimSpace(text))
	triggers := []string{
		"read the screen",
		"what does the screen say",
		"read text on screen",
		"ocr",
		"extract text from screen",
		"what's written on screen",
		"read screen",
	}
	for _, t := range triggers {
		if strings.Contains(lower, t) {
			return true
		}
	}
	return false
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

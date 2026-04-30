//go:build darwin

package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	configmodel "github.com/AzozzALFiras/Nullhand/internal/model/config"
	configrepo "github.com/AzozzALFiras/Nullhand/internal/repository/config"
	permissions "github.com/AzozzALFiras/Nullhand/internal/service/linux/permissions"
	botvm "github.com/AzozzALFiras/Nullhand/internal/viewmodel/bot"
	permvm "github.com/AzozzALFiras/Nullhand/internal/viewmodel/permissions"
	setupvm "github.com/AzozzALFiras/Nullhand/internal/viewmodel/setup"
)

func main() {
	log.SetFlags(log.Ltime | log.Lshortfile)

	// macOS doesn't need an X11 check, but we verify the required built-in
	// CLI tools (osascript, screencapture, sips) and warn about optional
	// Homebrew helpers (cliclick, tesseract).
	if err := permissions.CheckDependencies(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	cfg, err := loadOrSetup()
	if err != nil {
		fmt.Fprintf(os.Stderr, "setup failed: %v\n", err)
		os.Exit(1)
	}

	// Probe Accessibility + Screen Recording permissions. Non-blocking — print
	// guidance but always continue so the user can grant on first failure.
	if !permvm.New().Ensure() {
		fmt.Println("⚠ Warning: some capabilities are missing. Certain features may not work.")
		fmt.Println("  Continuing anyway — restart after granting the missing capabilities if needed.")
	}

	bot, err := botvm.New(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to start bot: %v\n", err)
		os.Exit(1)
	}

	// Graceful shutdown on SIGINT / SIGTERM.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-quit
		log.Println("Shutting down...")
		bot.Stop()
		os.Exit(0)
	}()

	bot.Start()
}

// loadOrSetup loads the existing config or runs the first-run setup wizard.
func loadOrSetup() (*configmodel.Config, error) {
	if configrepo.Exists() {
		return configrepo.Load()
	}
	wizard := setupvm.New()
	return wizard.Run()
}

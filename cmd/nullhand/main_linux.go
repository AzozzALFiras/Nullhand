//go:build linux

package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	configmodel "github.com/iamakillah/Nullhand_Linux/internal/model/config"
	configrepo "github.com/iamakillah/Nullhand_Linux/internal/repository/config"
	permissions "github.com/iamakillah/Nullhand_Linux/internal/service/linux/permissions"
	botvm "github.com/iamakillah/Nullhand_Linux/internal/viewmodel/bot"
	permvm "github.com/iamakillah/Nullhand_Linux/internal/viewmodel/permissions"
	setupvm "github.com/iamakillah/Nullhand_Linux/internal/viewmodel/setup"
)

func main() {
	log.SetFlags(log.Ltime | log.Lshortfile)

	// Check X11 session first
	if err := permissions.CheckX11Session(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	cfg, err := loadOrSetup()
	if err != nil {
		fmt.Fprintf(os.Stderr, "setup failed: %v\n", err)
		os.Exit(1)
	}

	// Check Linux capabilities before starting the bot.
	// Non-blocking: print a warning but always continue so the user can
	// decide whether the missing capability matters for their use case.
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

package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	configmodel "github.com/AzozzALFiras/nullhand/internal/model/config"
	configrepo "github.com/AzozzALFiras/nullhand/internal/repository/config"
	botvm "github.com/AzozzALFiras/nullhand/internal/viewmodel/bot"
	permvm "github.com/AzozzALFiras/nullhand/internal/viewmodel/permissions"
	setupvm "github.com/AzozzALFiras/nullhand/internal/viewmodel/setup"
)

func main() {
	log.SetFlags(log.Ltime | log.Lshortfile)

	cfg, err := loadOrSetup()
	if err != nil {
		fmt.Fprintf(os.Stderr, "setup failed: %v\n", err)
		os.Exit(1)
	}

	// Check macOS privacy permissions before starting the bot.
	permvm.New().Ensure()

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

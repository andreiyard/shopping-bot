package main

import (
	"log"
	"log/slog"
	"os"
	"strings"

	"shopping-bot/internal/config"
	"shopping-bot/internal/telegram"
)

func handleUpdate(u telegram.Update) {
	if u.Message.ID != 0 {
		handleMessage(u.Message)
	}
}

func handleMessage(m telegram.Message) {
	// Skip if no text (For now ignore the images and other media)
	if m.Text == "" {
		return
	}
	// If starts with '/' -> parse args and handle command
	if m.Text[0] == '/' {
		args := strings.Fields(m.Text)
		handleCommand(args...)
		return
	}

}

func handleCommand(args ...string) {
	cmd := args[0]
	switch cmd {
	case "/add":
		// TODO: Add args to DB (how do I pass DB to this function?)
		slog.Debug("Added", "entry", strings.Join(args[1:], " "))
	default:
		slog.Debug("Command not implemented", "command", cmd)
	}
}

func main() {
	// Load configuration
	cfg := config.Load()

	// Setup logging
	SetupLogging(cfg.Debug)

	// Initialize Telegram client
	tg := telegram.NewClient(cfg.TelegramToken)

	// Check that bot is working and is able to query API
	res, err := tg.GetMe()
	if err != nil {
		panic(err)
	} else if res.StatusCode != 200 {
		log.Fatalf("Unable to query API, got %s\n", res.Status)
	}

	// Setup long polling in goroutine that sends events in channel
	updates := tg.StartPolling()

	// Read continuously from the channel
	// Should block when no updates
	for {
		u := <-updates
		slog.Debug("Received update", "update", u)
		handleUpdate(u)
	}
}

func SetupLogging(debugEnabled bool) {
	level := slog.LevelInfo
	if debugEnabled {
		level = slog.LevelDebug
	}
	opts := &slog.HandlerOptions{
		Level: level,
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, opts))
	slog.SetDefault(logger)
}

package main

import (
	"fmt"
	"log"

	"shopping-bot/internal/config"
	"shopping-bot/internal/telegram"
)

func main() {
	// Load configuration
	cfg := config.Load()

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
		fmt.Println(u)
	}
}

package main

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"slices"
	"strconv"
	"strings"

	"shopping-bot/internal/config"
	"shopping-bot/internal/database"
	"shopping-bot/internal/telegram"
)

// Bot holds all dependencies for the application
type Bot struct {
	db     *database.DB
	tg     *telegram.Client
	config *config.Config
}

// NewBot creates a new Bot instance with all dependencies
func NewBot(cfg *config.Config) (*Bot, error) {
	// Create Telegram client
	tg := telegram.NewClient(cfg.TelegramToken)

	// Check that bot is working and is able to query API
	res, err := tg.GetMe()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Telegram: %w", err)
	}
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("unable to query Telegram API, got %s", res.Status)
	}

	// Connect to database
	db, err := database.Open(cfg.DatabasePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	return &Bot{
		db:     db,
		tg:     tg,
		config: cfg,
	}, nil
}

// Close cleans up Bot resources
func (b *Bot) Close() error {
	return b.db.Close()
}

// isAuthorized checks if a user is in the allowed users list
func (b *Bot) isAuthorized(userID int64) bool {
	// If no users specified, allow all
	if len(b.config.AllowedUsers) == 0 {
		return true
	}

	return slices.Contains(b.config.AllowedUsers, userID)
}

// handleUpdate processes incoming Telegram updates
func (b *Bot) handleUpdate(u telegram.Update) {
	if u.Message.ID != 0 {
		b.handleMessage(u.Message)
	}
}

// handleMessage processes incoming messages
func (b *Bot) handleMessage(m telegram.Message) {
	// Skip if no text (for now ignore images and other media)
	if m.Text == "" {
		return
	}

	// Check authorization
	if !b.isAuthorized(m.From.ID) {
		slog.Warn("Unauthorized access attempt", "user_id", m.From.ID, "username", m.From.Username)
		return
	}

	// If starts with '/' -> handle command
	if strings.HasPrefix(m.Text, "/") {
		b.handleCommand(m)
		return
	}
}

// handleCommand routes commands to appropriate handlers
func (b *Bot) handleCommand(m telegram.Message) {
	args := strings.Fields(m.Text)
	if len(args) == 0 {
		return
	}

	cmd := args[0]
	chatID := m.Chat.ID
	userID := m.From.ID

	switch cmd {
	case "/start":
		b.handleStart(chatID)
	case "/help":
		b.handleHelp(chatID)
	case "/set":
		b.handleSetList(chatID, userID, args[1:])
	case "/add":
		b.handleAdd(chatID, userID, args[1:])
	case "/list":
		b.handleList(chatID, userID)
	case "/bought":
		b.handleBought(chatID, userID, args[1:])
	case "/history":
		b.handleHistory(chatID, userID)
	default:
		b.tg.SendMessage(chatID, "❓ Unknown command. Use /help to see available commands.")
	}
}

// getCurrentListOrPrompt gets the user's current list or prompts them to select one
func (b *Bot) getCurrentListOrPrompt(chatID, userID int64) (string, bool) {
	listID, err := b.db.GetCurrentList(userID)
	if err != nil {
		slog.Error("Failed to get current list", "error", err, "user_id", userID)
		b.tg.SendMessage(chatID, "❌ Error getting your current list. Please try again.")
		return "", false
	}

	if listID == "" {
		b.tg.SendMessage(chatID, "❌ Please select a list first: /set <list_id>")
		return "", false
	}

	return listID, true
}

// handleSetList selects or creates a shopping list
func (b *Bot) handleSetList(chatID, userID int64, args []string) {
	if len(args) == 0 {
		b.tg.SendMessage(chatID, "❌ Please specify a list ID.\nUsage: /set <list_id>")
		return
	}

	listID := args[0]

	// Check if list exists
	exists, err := b.db.ListExists(listID)
	if err != nil {
		slog.Error("Failed to check list existence", "error", err, "list_id", listID)
		b.tg.SendMessage(chatID, "❌ Error checking list. Please try again.")
		return
	}

	// Create list if it doesn't exist (ntfy.sh style)
	if !exists {
		if err := b.db.CreateList(listID, userID); err != nil {
			slog.Error("Failed to create list", "error", err, "list_id", listID)
			b.tg.SendMessage(chatID, "❌ Error creating list. Please try again.")
			return
		}
		slog.Info("Created new list", "list_id", listID, "created_by", userID)
	}

	// Set as current list for user
	if err := b.db.SetCurrentList(userID, listID); err != nil {
		slog.Error("Failed to set current list", "error", err, "user_id", userID, "list_id", listID)
		b.tg.SendMessage(chatID, "❌ Error selecting list. Please try again.")
		return
	}

	slog.Debug("User selected list", "user_id", userID, "list_id", listID)
	b.tg.SendMessage(chatID, fmt.Sprintf("✅ Selected list: %s", listID))
}

// handleStart sends a welcome message
func (b *Bot) handleStart(chatID int64) {
	msg := "👋 Welcome to Shopping Bot!\n\n"
	msg += "I help you manage shared shopping lists.\n\n"
	msg += "Use /help to see available commands."
	b.tg.SendMessage(chatID, msg)
}

// handleHelp sends the list of available commands
func (b *Bot) handleHelp(chatID int64) {
	msg := "📝 Available commands:\n\n"
	msg += "/set <list_id> - Select/create shopping list\n"
	msg += "/add <item> - Add item to current list\n"
	msg += "/list - Show current shopping list\n"
	msg += "/bought <number> - Mark item as bought\n"
	msg += "/history - Show recently bought items\n"
	msg += "/help - Show this help message\n\n"
	msg += "💡 Tip: List IDs work like passwords - share them with others to collaborate!"
	b.tg.SendMessage(chatID, msg)
}

// handleAdd adds an item to the shopping list
func (b *Bot) handleAdd(chatID, userID int64, args []string) {
	// Get current list
	listID, ok := b.getCurrentListOrPrompt(chatID, userID)
	if !ok {
		return
	}

	if len(args) == 0 {
		b.tg.SendMessage(chatID, "❌ Please specify an item to add.\nUsage: /add <item>")
		return
	}

	itemName := strings.Join(args, " ")

	if err := b.db.AddItem(listID, itemName, userID); err != nil {
		slog.Error("Failed to add item", "error", err, "list_id", listID, "user_id", userID)
		b.tg.SendMessage(chatID, "❌ Failed to add item. Please try again.")
		return
	}

	slog.Debug("Item added", "list_id", listID, "user_id", userID, "item", itemName)
	b.tg.SendMessage(chatID, fmt.Sprintf("✅ Added: %s", itemName))
}

// handleList shows the current shopping list
func (b *Bot) handleList(chatID, userID int64) {
	// Get current list
	listID, ok := b.getCurrentListOrPrompt(chatID, userID)
	if !ok {
		return
	}

	items, err := b.db.GetItems(listID)
	if err != nil {
		slog.Error("Failed to get items", "error", err, "list_id", listID)
		b.tg.SendMessage(chatID, "❌ Failed to load shopping list. Please try again.")
		return
	}

	if len(items) == 0 {
		b.tg.SendMessage(chatID, fmt.Sprintf("📝 Shopping list '%s' is empty.\n\nUse /add to add items.", listID))
		return
	}

	var msg strings.Builder
	msg.WriteString(fmt.Sprintf("🛒 Shopping list '%s':\n\n", listID))
	for i, item := range items {
		msg.WriteString(fmt.Sprintf("%d. %s\n", i+1, item.Name))
	}
	msg.WriteString("\nUse /bought <number> to mark items as bought.")

	b.tg.SendMessage(chatID, msg.String())
}

// handleBought marks an item as bought
func (b *Bot) handleBought(chatID, userID int64, args []string) {
	// Get current list
	listID, ok := b.getCurrentListOrPrompt(chatID, userID)
	if !ok {
		return
	}

	if len(args) == 0 {
		b.tg.SendMessage(chatID, "❌ Please specify item number.\nUsage: /bought <number>")
		return
	}

	// Get current items to map number to ID
	items, err := b.db.GetItems(listID)
	if err != nil {
		slog.Error("Failed to get items", "error", err, "list_id", listID)
		b.tg.SendMessage(chatID, "❌ Failed to load shopping list. Please try again.")
		return
	}

	if len(items) == 0 {
		b.tg.SendMessage(chatID, "📝 Shopping list is empty.")
		return
	}

	// Parse item number
	itemNum, err := strconv.Atoi(args[0])
	if err != nil || itemNum < 1 || itemNum > len(items) {
		b.tg.SendMessage(chatID, fmt.Sprintf("❌ Invalid item number. Please use a number between 1 and %d.", len(items)))
		return
	}

	// Get the item by index (1-based to 0-based)
	item := items[itemNum-1]

	// Mark as bought
	if err := b.db.MarkBought(item.ID, listID, userID); err != nil {
		slog.Error("Failed to mark item as bought", "error", err, "item_id", item.ID, "list_id", listID)
		b.tg.SendMessage(chatID, "❌ Failed to mark item as bought. Please try again.")
		return
	}

	slog.Debug("Item marked as bought", "list_id", listID, "user_id", userID, "item_id", item.ID, "item", item.Name)
	b.tg.SendMessage(chatID, fmt.Sprintf("✅ Marked as bought: %s", item.Name))
}

// handleHistory shows recently bought items
func (b *Bot) handleHistory(chatID, userID int64) {
	// Get current list
	listID, ok := b.getCurrentListOrPrompt(chatID, userID)
	if !ok {
		return
	}

	items, err := b.db.GetHistory(listID, 10)
	if err != nil {
		slog.Error("Failed to get history", "error", err, "list_id", listID)
		b.tg.SendMessage(chatID, "❌ Failed to load history. Please try again.")
		return
	}

	if len(items) == 0 {
		b.tg.SendMessage(chatID, fmt.Sprintf("📜 No purchase history for '%s' yet.", listID))
		return
	}

	var msg strings.Builder
	msg.WriteString(fmt.Sprintf("📜 Recently bought from '%s':\n\n", listID))
	for i, item := range items {
		msg.WriteString(fmt.Sprintf("%d. %s\n", i+1, item.Name))
	}

	b.tg.SendMessage(chatID, msg.String())
}

func main() {
	// Load configuration
	cfg := config.Load()

	// Setup logging
	SetupLogging(cfg.Debug)

	// Initialize bot with all dependencies
	bot, err := NewBot(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize bot: %v", err)
	}
	defer bot.Close()

	slog.Info("Bot started successfully")

	// Setup long polling in goroutine that sends events in channel
	updates := bot.tg.StartPolling()

	// Read continuously from the channel
	// Should block when no updates
	for u := range updates {
		slog.Debug("Received update", "update", u)
		bot.handleUpdate(u)
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

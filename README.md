# Shopping List Telegram Bot

A Telegram bot for managing shared shopping lists.  
Built with Go using standard library and minimal dependencies.

## Features

- Add items with details (quantity, volume)
- Mark items as purchased
- View purchase history
- Quick re-add from history
- User whitelist for access control

## Tech Stack

- Go (stdlib-focused)
- SQLite (`modernc.org/sqlite`)
- Telegram Bot API via `net/http`

## MVP Plan

### Phase 1: Core
- [ ] Telegram API client (long polling)
- [ ] User whitelist authentication
- [ ] SQLite setup and schema

### Phase 2: Basic Commands
- [ ] `/add <item>` - add to list
- [ ] `/list` - show active items
- [ ] `/bought <item>` - mark purchased
- [ ] `/help` - show commands

### Phase 3: History
- [ ] `/history` - recent purchases
- [ ] Quick-add from history

## Configuration
```bash
TELEGRAM_TOKEN=your_bot_token
ALLOWED_USERS=123456789,987654321
```

## Future Features

- Adding items in bulk (fuzzy match with history)
- Store/category grouping
- Purchase frequency analytics
- Smart suggestions based on history
- OCR receipt scanning
- Price tracking and budgets
- Reminders for regular purchases

## Build
```bash
go mod download
go build -o shopping-bot
```

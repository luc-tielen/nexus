package telegram

import (
	"fmt"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Sender is the subset of tgbotapi.BotAPI used for sending messages.
// Extracted as an interface to allow test doubles.
type Sender interface {
	Send(c tgbotapi.Chattable) (tgbotapi.Message, error)
}

// Client holds a Telegram bot and the target chat ID.
type Client struct {
	Bot    Sender
	ChatID int64
}

// NewClient returns a ready client for the given token and chatID string.
// Returns an error if either value is empty or if the bot cannot be created.
func NewClient(token, chatID string) (*Client, error) {
	if token == "" {
		return nil, fmt.Errorf("TELEGRAM_BOT_TOKEN is not set")
	}
	if chatID == "" {
		return nil, fmt.Errorf("TELEGRAM_CHAT_ID is not set")
	}

	id, err := strconv.ParseInt(chatID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("TELEGRAM_CHAT_ID must be a numeric chat ID: %w", err)
	}

	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("creating Telegram bot: %w", err)
	}

	return &Client{Bot: bot, ChatID: id}, nil
}

// SendTo posts content to an arbitrary chat ID.
func (c *Client) SendTo(chatID int64, content string) error {
	_, err := c.Bot.Send(tgbotapi.NewMessage(chatID, content))
	return err
}

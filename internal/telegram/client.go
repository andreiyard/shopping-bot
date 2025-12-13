package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
)

type Client struct {
	baseUrl string
	token   string
}

func NewClient(token string, baseUrlOptional ...string) *Client {
	baseUrl := "https://api.telegram.org"
	if len(baseUrlOptional) > 0 {
		baseUrl = baseUrlOptional[0]
	}
	return &Client{baseUrl, token}
}

func (c *Client) getMethod(method string, params url.Values) (*http.Response, error) {
	slog.Debug("Making telegram API request", "method", method, "params", params)

	// Parse URL and add params
	constructedURL, err := url.JoinPath(c.baseUrl, "bot"+c.token, method)
	if err != nil {
		return nil, err
	}
	parsedURL, err := url.Parse(constructedURL)
	if err != nil {
		return nil, err
	}
	parsedURL.RawQuery = params.Encode()

	// Make a request
	res, err := http.Get(parsedURL.String())
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (c *Client) StartPolling() chan Update {
	slog.Info("Starting polling")
	updates := make(chan Update)

	// TODO: Get timeout from config
	// TODO: Move for loop body to separate CheckUpdates function
	go func(updates chan Update) {
		var currentOffset int64
		params := url.Values{}
		params.Add("offset", "0")
		params.Add("timeout", "10")
		for {
			params.Set("offset", strconv.FormatInt(currentOffset, 10))
			res, err := c.getMethod("getUpdates", params)
			if err != nil {
				// Don't crash if one request failed
				slog.Debug("Got error", "err", err)
				continue
			}

			body, err := io.ReadAll(res.Body)
			if err != nil {
				slog.Debug("Got error", "err", err)
				continue
			}

			var response TgResponse
			if err := json.Unmarshal([]byte(body), &response); err != nil {
				slog.Debug("Got error", "err", err)
				continue
			}
			if !response.Ok {
				slog.Debug("getUpdates returned Ok:false")
			}

			for _, u := range response.Result {
				updates <- u
				// After reading update, set offset to avoid duplicate updates
				currentOffset = u.UpdateID + 1
			}
		}
	}(updates)
	return updates
}

// GetMe checks that the bot token is valid
func (c *Client) GetMe() (*http.Response, error) {
	return c.getMethod("getMe", nil)
}

// SendMessage sends a text message to a chat
func (c *Client) SendMessage(chatID int64, text string) error {
	url := fmt.Sprintf("%s/bot%s/sendMessage", c.baseUrl, c.token)

	reqBody := SendMessageRequest{
		ChatID: chatID,
		Text:   text,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	defer resp.Body.Close()

	var result SendMessageResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if !result.Ok {
		return fmt.Errorf("telegram API returned ok=false")
	}

	slog.Debug("Message sent successfully", "chat_id", chatID)
	return nil
}

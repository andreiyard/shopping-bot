package telegram

import (
	"encoding/json"
	"io"
	"log"
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
	// TODO: Add slog to write debug logs
	updates := make(chan Update)
	go func(updates chan Update) {
		currentOffset := 0
		params := url.Values{}
		params.Add("offset", "0")
		params.Add("timeout", "10")
		for {
			params.Set("offset", strconv.Itoa(currentOffset))
			res, err := c.getMethod("getUpdates", params)
			if err != nil {
				// Don't crash if one request failed
				log.Println(err)
				continue
			}
			var response TgResponse
			body, err := io.ReadAll(res.Body)
			if err != nil {
				log.Println(err)
				continue
			}
			if err := json.Unmarshal([]byte(body), &response); err != nil {
				log.Println(err)
				continue
			}
			if !response.Ok {
				log.Println("getUpdates returned Ok:false")
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

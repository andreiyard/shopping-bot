package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type TgAPI struct {
	baseUrl string
	token   string
}

func NewTgAPI(token string, baseUrlOptional ...string) TgAPI {
	baseUrl := "https://api.telegram.org"
	if len(baseUrlOptional) > 0 {
		baseUrl = baseUrlOptional[0]
	}
	return TgAPI{baseUrl, token}
}

type TgResponse struct {
	Ok     bool     `json:"ok"`
	Result []Update `json:"result"`
}

type Update struct {
	UpdateID int     `json:"update_id"`
	Message  Message `json:"message"`
}

type Message struct {
	MessageID int    `json:"message_id"`
	From      User   `json:"from"`
	Chat      Chat   `json:"chat"`
	Date      int64  `json:"date"`
	Text      string `json:"text"`
}

type User struct {
	ID           int    `json:"id"`
	IsBot        bool   `json:"is_bot"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	Username     string `json:"username"`
	LanguageCode string `json:"language_code"`
}

type Chat struct {
	ID        int    `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Username  string `json:"username"`
	Type      string `json:"type"`
}

func (tg TgAPI) getMethod(method string, params url.Values) (*http.Response, error) {
	// Parse URL and add params
	constructedURL, err := url.JoinPath(tg.baseUrl, "bot"+tg.token, method)
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

func (tg TgAPI) startPolling() chan Update {
	// TODO: Add slog to write debug logs
	updates := make(chan Update)
	go func(updates chan Update) {
		currentOffset := 0
		params := url.Values{}
		params.Add("offset", "0")
		params.Add("timeout", "10")
		for {
			params.Set("offset", strconv.Itoa(currentOffset))
			res, err := tg.getMethod("getUpdates", params)
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

func main() {
	// Load .env file
	godotenv.Load()
	// Get TOKEN
	token := os.Getenv("TG_TOKEN")

	tg := NewTgAPI(token)

	// Check that bot is working and is able to query API
	res, err := tg.getMethod("getMe", nil)
	if err != nil {
		panic(err)
	} else if res.StatusCode != 200 {
		log.Fatalf("Unable to query API, got %s\n", res.Status)
	}

	// Setup long polling in goroutine that sends events in channel
	updates := tg.startPolling()

	// Read continously from the channel
	// Should block when no updates
	for {
		u := <-updates
		fmt.Println(u)
	}

}

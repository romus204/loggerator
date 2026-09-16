package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/romus204/loggerator/internal/config"
)

type SendMessageRequest struct {
	ChatID                int    `json:"chat_id"`
	Text                  string `json:"text"`
	ParseMode             string `json:"parse_mode,omitempty"`
	MessageThreadID       int    `json:"message_thread_id,omitempty"`
	DisableWebPagePreview bool   `json:"disable_web_page_preview,omitempty"`
}

type Telegram struct {
	ctx        context.Context
	UrlSend    string         // main telegram api url
	Token      string         // telegram bot token
	Chat       int            // main chat id
	Topics     map[string]int // topics list
	queue      chan SendMessageRequest
	ticker     *time.Ticker
	httpClient *http.Client
}

func NewBot(ctx context.Context, cfg config.Telegram) *Telegram {
	bot := &Telegram{
		ctx:        ctx,
		UrlSend:    fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", cfg.Token),
		Token:      cfg.Token,
		Chat:       cfg.Chat,
		Topics:     cfg.Topics,
		queue:      make(chan SendMessageRequest, 1000),
		ticker:     time.NewTicker(time.Minute / 20),
		httpClient: newHTTPClient(cfg.Proxy),
	}

	return bot
}

// newHTTPClient builds an http.Client that routes requests through proxyURL,
// if set. Supported schemes: http, https, socks5.
func newHTTPClient(proxyURL string) *http.Client {
	if proxyURL == "" {
		return http.DefaultClient
	}

	u, err := url.Parse(proxyURL)
	if err != nil {
		log.Printf("telegram: invalid proxy url %q, ignoring: %v", proxyURL, err)
		return http.DefaultClient
	}

	return &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(u),
		},
	}
}

func (b *Telegram) StartSendWorker(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer b.ticker.Stop()

		for {
			select {
			case <-b.ctx.Done():
				return
			case msg := <-b.queue:
				select {
				case <-b.ticker.C:
				case <-b.ctx.Done():
					return
				}
				b.sendToApi(msg)
			}
		}
	}()
}

func (b *Telegram) Send(msg string, container string) {
	var formattedLine string
	if b.isJSON(msg) {
		formattedLine = fmt.Sprintf("```json\n%s\n```", b.escapeCodeBlock(b.prettyPrintJSON(msg)))
	} else {
		formattedLine = fmt.Sprintf("```\n%s\n```", b.escapeCodeBlock(msg))
	}

	message := SendMessageRequest{
		ChatID:    b.Chat,
		Text:      formattedLine,
		ParseMode: "MarkdownV2",
	}

	t, ok := b.Topics[container] // if find't topic, just send in main chat
	if ok {
		message.MessageThreadID = t
	}

	select {
	case <-b.ctx.Done():
		return
	case b.queue <- message:
	}
}

func (b *Telegram) sendToApi(msg SendMessageRequest) {
	jsonData, err := json.Marshal(msg)
	if err != nil {
		fmt.Println("Error marshalling JSON:", err)
		return
	}

	client := b.httpClient
	if client == nil {
		client = http.DefaultClient
	}

	resp, err := client.Post(b.UrlSend, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Println("Error sending request:", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Println("Error:", resp.Status)
		return
	}

}

// escapeCodeBlock escapes the characters that MarkdownV2 requires to be
// escaped inside a ``` code block: backslash and backtick.
func (b *Telegram) escapeCodeBlock(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, "`", "\\`")
	return s
}

func (b *Telegram) isJSON(str string) bool {
	var js json.RawMessage
	return json.Unmarshal([]byte(str), &js) == nil
}

func (b *Telegram) prettyPrintJSON(str string) string {
	var obj map[string]interface{}
	if err := json.Unmarshal([]byte(str), &obj); err != nil {
		return str
	}

	prettyJSON, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return str
	}

	return string(prettyJSON)
}

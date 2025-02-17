package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

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
	UrlSend string         // main telegram api url
	Token   string         // telegram bot token
	Chat    int            // main chat id
	Topics  map[string]int // topics list
}

func NewBot(cfg config.Telegram) *Telegram {
	return &Telegram{
		UrlSend: fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", cfg.Token),
		Token:   cfg.Token,
		Chat:    cfg.Chat,
		Topics:  cfg.Topics,
	}
}

func (b *Telegram) Send(msg string, container string) {
	var formattedLine string
	if b.isJSON(msg) {
		formattedLine = fmt.Sprintf("```json\n%s\n```", b.prettyPrintJSON(msg))
	} else {
		formattedLine = fmt.Sprintf("```\n%s\n```", msg)
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

	jsonData, err := json.Marshal(message)
	if err != nil {
		fmt.Println("Error marshalling JSON:", err)
		return
	}

	resp, err := http.Post(b.UrlSend, "application/json", bytes.NewBuffer(jsonData))
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

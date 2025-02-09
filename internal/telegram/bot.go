package telegram

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/romus204/loggerator/internal/config"
)

type Bot struct {
	bot *tgbotapi.BotAPI
	cfg config.Telegram
}

func NewBot(cfg config.Telegram) *Bot {
	bot, err := tgbotapi.NewBotAPI(cfg.Token)
	if err != nil {
		log.Fatalf("bot init failed: %v", err)
	}

	bot.Debug = cfg.Debug

	return &Bot{bot: bot, cfg: cfg}
}

func (b *Bot) Send(msg string) {
	var formattedLine string
	if b.isJSON(msg) {
		formattedLine = fmt.Sprintf("```json\n%s\n```", b.prettyPrintJSON(msg))
	} else {
		formattedLine = fmt.Sprintf("```\n%s\n```", msg)
	}

	for _, c := range b.cfg.Chat {
		msg := tgbotapi.NewMessage(c, formattedLine)
		msg.ParseMode = "MarkdownV2"
		_, err := b.bot.Send(msg)
		if err != nil {
			log.Printf("msg not send: %v", err)
		}

		time.Sleep(b.cfg.DelayMsg)
	}

}

func (b *Bot) isJSON(str string) bool {
	var js json.RawMessage
	return json.Unmarshal([]byte(str), &js) == nil
}

func (b *Bot) prettyPrintJSON(str string) string {
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

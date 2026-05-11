package mattermost

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/romus204/loggerator/internal/config"
)

type postPayload struct {
	ChannelID string `json:"channel_id"`
	Message   string `json:"message"`
}

type sendRequest struct {
	payload postPayload
}

type Mattermost struct {
	ctx       context.Context
	apiURL    string
	token     string
	channelID string
	channels  map[string]string
	queue     chan sendRequest
	ticker    *time.Ticker
}

func NewClient(ctx context.Context, cfg config.Mattermost) *Mattermost {
	return &Mattermost{
		ctx:       ctx,
		apiURL:    cfg.ServerURL + "/api/v4/posts",
		token:     cfg.Token,
		channelID: cfg.ChannelID,
		channels:  cfg.Channels,
		queue:     make(chan sendRequest, 1000),
		ticker:    time.NewTicker(time.Minute / 20),
	}
}

func (m *Mattermost) StartSendWorker(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()

		for req := range m.queue {
			<-m.ticker.C
			m.sendToAPI(req.payload)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		<-m.ctx.Done()
		if m.ticker != nil {
			m.ticker.Stop()
		}
		close(m.queue)
	}()
}

func (m *Mattermost) Send(msg string, container string) {
	var text string
	if isJSON(msg) {
		text = fmt.Sprintf("```json\n%s\n```", prettyPrintJSON(msg))
	} else {
		text = fmt.Sprintf("```\n%s\n```", msg)
	}

	channelID := m.channelID
	if ch, ok := m.channels[container]; ok {
		channelID = ch
	}

	select {
	case <-m.ctx.Done():
		return
	case m.queue <- sendRequest{payload: postPayload{ChannelID: channelID, Message: text}}:
	}
}

func (m *Mattermost) sendToAPI(payload postPayload) {
	data, err := json.Marshal(payload)
	if err != nil {
		fmt.Println("mattermost: error marshalling JSON:", err)
		return
	}

	req, err := http.NewRequest(http.MethodPost, m.apiURL, bytes.NewBuffer(data))
	if err != nil {
		fmt.Println("mattermost: error creating request:", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+m.token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("mattermost: error sending request:", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		fmt.Println("mattermost: unexpected response:", resp.Status)
	}
}

func isJSON(str string) bool {
	var js json.RawMessage
	return json.Unmarshal([]byte(str), &js) == nil
}

func prettyPrintJSON(str string) string {
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

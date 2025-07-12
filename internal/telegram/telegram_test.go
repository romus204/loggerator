package telegram

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/romus204/loggerator/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestNewBot(t *testing.T) {
	ctx := context.Background()
	cfg := config.Telegram{
		Token: "123456",
		Chat:  78910,
		Topics: map[string]int{
			"test": 1,
		},
	}

	bot := NewBot(ctx, cfg)

	assert.Equal(t, "123456", bot.Token)
	assert.Equal(t, 78910, bot.Chat)
	assert.Equal(t, 1, bot.Topics["test"])
	assert.NotNil(t, bot.queue)
	assert.NotNil(t, bot.ticker)
	assert.Equal(t, ctx, bot.ctx)
}

func TestIsJSON(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"{}", true},
		{"[]", true},
		{"{\"key\": \"value\"}", true},
		{"[1, 2, 3]", true},
		{"{invalid json", false},
		{"random string", false},
	}

	tb := &Telegram{}

	for _, test := range tests {
		result := tb.isJSON(test.input)
		if result != test.expected {
			t.Errorf("isJSON(%q) = %v; want %v", test.input, result, test.expected)
		}
	}
}

func TestPrettyPrintJSON(t *testing.T) {
	input := `{"name":"John", "age":30}`
	expectedOutput := "{\n  \"age\": 30,\n  \"name\": \"John\"\n}"

	tb := &Telegram{}
	actualOutput := tb.prettyPrintJSON(input)

	if !reflect.DeepEqual(actualOutput, expectedOutput) {
		t.Errorf("prettyPrintJSON(%q) = %q; want %q", input, actualOutput, expectedOutput)
	}
}

func TestSend(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	b := &Telegram{
		ctx:  ctx,
		Chat: 123,
		Topics: map[string]int{
			"container1": 999,
		},
		queue: make(chan SendMessageRequest, 1),
	}

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		msg := <-b.queue
		assert.Equal(t, 123, msg.ChatID)
		assert.Equal(t, 999, msg.MessageThreadID)
		assert.Contains(t, msg.Text, "```")
	}()

	b.Send("test message", "container1")
	
	wg.Wait()
}

func TestSendToApi(t *testing.T) {
	var received []byte
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		received = body
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	b := &Telegram{
		UrlSend: ts.URL,
	}

	msg := SendMessageRequest{
		ChatID:    123,
		Text:      "hello",
		ParseMode: "MarkdownV2",
	}

	b.sendToApi(msg)

	var decoded SendMessageRequest
	err := json.Unmarshal(received, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, "hello", decoded.Text)
}

func TestStartSendWorker(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var mu sync.Mutex
	var called bool

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		called = true
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	b := &Telegram{
		ctx:     ctx,
		UrlSend: ts.URL,
		queue:   make(chan SendMessageRequest, 1),
		ticker:  time.NewTicker(time.Millisecond), // ускорим тикер
	}

	var wg sync.WaitGroup
	b.StartSendWorker(&wg)

	b.queue <- SendMessageRequest{ChatID: 1, Text: "test"}

	// подождём немного
	time.Sleep(10 * time.Millisecond)
	cancel()

	wg.Wait()

	mu.Lock()
	assert.True(t, called)
	mu.Unlock()
}

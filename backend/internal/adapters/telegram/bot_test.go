package telegram

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/xhamps/bragdocument/backend/internal/app"
)

type fakeReplier struct {
	mu    sync.Mutex
	from  int64
	texts []string
}

func (f *fakeReplier) Reply(_ context.Context, telegramID int64, text string) app.BotReply {
	f.mu.Lock()
	f.from, f.texts = telegramID, append(f.texts, text)
	f.mu.Unlock()
	if text == "boom" {
		panic("replier exploded")
	}
	return app.BotReply{Text: "ok: " + text, Buttons: []app.BotButton{{Label: "Add", Data: "impact:add:l1"}}}
}

func (f *fakeReplier) Callback(_ context.Context, telegramID int64, data string) app.BotReply {
	f.mu.Lock()
	f.from, f.texts = telegramID, append(f.texts, "tap:"+data)
	f.mu.Unlock()
	return app.BotReply{Text: "tapped " + data}
}

func TestBotRepliesToPrivateMessagesAndSurvivesPanics(t *testing.T) {
	var mu sync.Mutex
	served := false
	type sentMsg struct{ text, parseMode, markup string }
	sent := make(chan sentMsg, 4)
	answered := make(chan string, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/getUpdates"):
			mu.Lock()
			first := !served
			served = true
			mu.Unlock()
			if first {
				_, _ = io.WriteString(w, `{"ok":true,"result":[
					{"update_id":1,"message":{"message_id":1,"date":0,"chat":{"id":99,"type":"group"},"from":{"id":7,"is_bot":false,"first_name":"G"},"text":"ignored"}},
					{"update_id":2,"message":{"message_id":2,"date":0,"chat":{"id":42,"type":"private"},"from":{"id":42,"is_bot":false,"first_name":"A"},"text":"boom"}},
					{"update_id":3,"message":{"message_id":3,"date":0,"chat":{"id":42,"type":"private"},"from":{"id":42,"is_bot":false,"first_name":"A"},"text":"Shipped X"}},
					{"update_id":4,"callback_query":{"id":"cb1","from":{"id":42,"is_bot":false,"first_name":"A"},"chat_instance":"x","data":"impact:add:l1"}}]}`)
				return
			}
			_, _ = io.WriteString(w, `{"ok":true,"result":[]}`)
		case strings.HasSuffix(r.URL.Path, "/sendMessage"):
			_ = r.ParseMultipartForm(1 << 20) // the library sends multipart; fall back to JSON if not
			m := sentMsg{text: r.FormValue("text"), parseMode: r.FormValue("parse_mode"), markup: r.FormValue("reply_markup")}
			if m.text == "" {
				var body struct {
					Text        string          `json:"text"`
					ParseMode   string          `json:"parse_mode"`
					ReplyMarkup json.RawMessage `json:"reply_markup"`
				}
				_ = json.NewDecoder(r.Body).Decode(&body)
				m = sentMsg{text: body.Text, parseMode: body.ParseMode, markup: string(body.ReplyMarkup)}
			}
			sent <- m
			_, _ = io.WriteString(w, `{"ok":true,"result":{"message_id":3,"date":0,"chat":{"id":42,"type":"private"}}}`)
		case strings.HasSuffix(r.URL.Path, "/answerCallbackQuery"):
			_ = r.ParseMultipartForm(1 << 20)
			answered <- r.FormValue("callback_query_id")
			_, _ = io.WriteString(w, `{"ok":true,"result":true}`)
		default:
			_, _ = io.WriteString(w, `{"ok":true,"result":true}`)
		}
	}))
	defer srv.Close()

	rep := &fakeReplier{}
	b, err := New(context.Background(), "TOKEN", rep, WithServerURL(srv.URL))
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	go b.Run(ctx)

	// Handlers run in order, so the replies arrive in update order.
	for _, want := range []string{failedReply, "ok: Shipped X", "tapped impact:add:l1"} {
		select {
		case got := <-sent:
			require.Equal(t, want, got.text)
			require.Empty(t, got.parseMode, "replies carry user text; send them as plain text")
			if want == "ok: Shipped X" {
				require.JSONEq(t, `{"inline_keyboard":[[{"text":"Add","callback_data":"impact:add:l1"}]]}`, got.markup)
			} else {
				require.Empty(t, got.markup)
			}
		case <-ctx.Done():
			t.Fatalf("no reply %q sent", want)
		}
	}
	require.Equal(t, "cb1", <-answered, "the button spinner is stopped")
	rep.mu.Lock()
	defer rep.mu.Unlock()
	require.Equal(t, int64(42), rep.from)
	require.Equal(t, []string{"boom", "Shipped X", "tap:impact:add:l1"}, rep.texts, "the group message never reaches the replier")
}

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
)

type fakeReplier struct {
	mu   sync.Mutex
	from int64
	text string
}

func (f *fakeReplier) Reply(_ context.Context, telegramID int64, text string) string {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.from, f.text = telegramID, text
	return "ok: " + text
}

func TestBotRepliesToPrivateMessages(t *testing.T) {
	var mu sync.Mutex
	served := false
	type sentMsg struct{ text, parseMode string }
	sent := make(chan sentMsg, 4)
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
					{"update_id":2,"message":{"message_id":2,"date":0,"chat":{"id":42,"type":"private"},"from":{"id":42,"is_bot":false,"first_name":"A"},"text":"Shipped X"}}]}`)
				return
			}
			_, _ = io.WriteString(w, `{"ok":true,"result":[]}`)
		case strings.HasSuffix(r.URL.Path, "/sendMessage"):
			_ = r.ParseMultipartForm(1 << 20) // the library sends multipart; fall back to JSON if not
			m := sentMsg{text: r.FormValue("text"), parseMode: r.FormValue("parse_mode")}
			if m.text == "" {
				var body struct {
					Text      string `json:"text"`
					ParseMode string `json:"parse_mode"`
				}
				_ = json.NewDecoder(r.Body).Decode(&body)
				m = sentMsg{text: body.Text, parseMode: body.ParseMode}
			}
			sent <- m
			_, _ = io.WriteString(w, `{"ok":true,"result":{"message_id":3,"date":0,"chat":{"id":42,"type":"private"}}}`)
		default:
			_, _ = io.WriteString(w, `{"ok":true,"result":true}`)
		}
	}))
	defer srv.Close()

	rep := &fakeReplier{}
	b, err := New("TOKEN", rep, WithServerURL(srv.URL))
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	go b.Run(ctx)

	select {
	case got := <-sent:
		require.Equal(t, "ok: Shipped X", got.text)
		require.Empty(t, got.parseMode, "replies carry user text; send them as plain text")
	case <-ctx.Done():
		t.Fatal("no reply sent")
	}
	rep.mu.Lock()
	defer rep.mu.Unlock()
	require.Equal(t, int64(42), rep.from)
}

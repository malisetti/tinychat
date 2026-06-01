package main

import (
	"encoding/json"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestHubUserListUpdatesOnJoinLeave(t *testing.T) {
	h := newHub()
	c1 := &client{name: "alice"}
	c2 := &client{name: "bob"}

	h.mu.Lock()
	h.clients[c1] = struct{}{}
	h.mu.Unlock()

	got := h.userList()
	sort.Strings(got)
	want := []string{"alice"}
	if len(got) != len(want) || got[0] != want[0] {
		t.Fatalf("after first join: got %v want %v", got, want)
	}

	h.mu.Lock()
	h.clients[c2] = struct{}{}
	h.mu.Unlock()

	got = h.userList()
	sort.Strings(got)
	want = []string{"alice", "bob"}
	if len(got) != len(want) {
		t.Fatalf("after second join: got %v want %v", got, want)
	}

	h.mu.Lock()
	delete(h.clients, c1)
	h.mu.Unlock()

	got = h.userList()
	want = []string{"bob"}
	if len(got) != 1 || got[0] != "bob" {
		t.Fatalf("after leave: got %v want %v", got, want)
	}
}

func TestNormalizeNameTrimsDefaultsAndCapsLength(t *testing.T) {
	if got := normalizeName("  alice  "); got != "alice" {
		t.Fatalf("trimmed name: got %q want %q", got, "alice")
	}
	if got := normalizeName("   "); got != "anon" {
		t.Fatalf("blank name: got %q want %q", got, "anon")
	}

	got := normalizeName(strings.Repeat("a", maxNameLength+1))
	if len([]rune(got)) != maxNameLength {
		t.Fatalf("capped name length: got %d want %d", len([]rune(got)), maxNameLength)
	}
}

func TestNormalizeTextTrimsRejectsBlankAndCapsLength(t *testing.T) {
	got, ok := normalizeText("  hello  ")
	if !ok || got != "hello" {
		t.Fatalf("trimmed text: got %q ok %t want %q true", got, ok, "hello")
	}

	if got, ok := normalizeText(" \t\n "); ok || got != "" {
		t.Fatalf("blank text: got %q ok %t want empty false", got, ok)
	}

	got, ok = normalizeText(strings.Repeat("x", maxTextLength+1))
	if !ok {
		t.Fatal("long text should be accepted after truncation")
	}
	if len([]rune(got)) != maxTextLength {
		t.Fatalf("capped text length: got %d want %d", len([]rune(got)), maxTextLength)
	}
}

func TestWebSocketNormalizesNameAndChatText(t *testing.T) {
	server := httptest.NewServer(handleWS(newHub()))
	defer server.Close()

	url := "ws" + strings.TrimPrefix(server.URL, "http") + "?name=%20alice%20"
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	defer conn.Close()

	if err := conn.WriteJSON(map[string]string{
		"type": "chat",
		"text": "  hello  ",
	}); err != nil {
		t.Fatalf("write chat: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if err := conn.SetReadDeadline(deadline); err != nil {
			t.Fatalf("set read deadline: %v", err)
		}
		_, data, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("read message: %v", err)
		}

		var msg map[string]string
		if err := json.Unmarshal(data, &msg); err != nil {
			continue
		}
		if msg["type"] != "chat" {
			continue
		}
		if msg["from"] != "alice" || msg["text"] != "hello" {
			t.Fatalf("chat message: got from=%q text=%q want alice/hello", msg["from"], msg["text"])
		}
		return
	}
	t.Fatal("timed out waiting for chat message")
}

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

const (
	maxNameLength = 32
	maxTextLength = 500
)

type client struct {
	conn *websocket.Conn
	name string
}

type hub struct {
	mu      sync.Mutex
	clients map[*client]struct{}
}

func newHub() *hub {
	return &hub{clients: make(map[*client]struct{})}
}

func truncateRunes(s string, limit int) string {
	runes := []rune(s)
	if len(runes) <= limit {
		return s
	}
	return string(runes[:limit])
}

func normalizeName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "anon"
	}
	return truncateRunes(name, maxNameLength)
}

func normalizeText(text string) (string, bool) {
	text = strings.TrimSpace(text)
	if text == "" {
		return "", false
	}
	return truncateRunes(text, maxTextLength), true
}

func (h *hub) userList() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	users := make([]string, 0, len(h.clients))
	for c := range h.clients {
		users = append(users, c.name)
	}
	return users
}

func (h *hub) broadcastPresence() {
	payload, err := json.Marshal(map[string]any{
		"type":  "presence",
		"users": h.userList(),
	})
	if err != nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	for c := range h.clients {
		_ = c.conn.WriteMessage(websocket.TextMessage, payload)
	}
}

func (h *hub) broadcast(msg []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for c := range h.clients {
		_ = c.conn.WriteMessage(websocket.TextMessage, msg)
	}
}

func (h *hub) add(c *client) {
	h.mu.Lock()
	h.clients[c] = struct{}{}
	h.mu.Unlock()
	h.broadcastPresence()
	h.broadcastSystem(fmt.Sprintf("%s joined", c.name))
}

func (h *hub) remove(c *client) {
	h.mu.Lock()
	delete(h.clients, c)
	h.mu.Unlock()
	h.broadcastPresence()
	h.broadcastSystem(fmt.Sprintf("%s left", c.name))
}

func (h *hub) broadcastSystem(text string) {
	payload, _ := json.Marshal(map[string]string{
		"type": "system",
		"text": text,
	})
	h.broadcast(payload)
}

func (h *hub) rename(c *client, newName string) {
	h.mu.Lock()
	oldName := c.name
	c.name = newName
	h.mu.Unlock()
	h.broadcastPresence()
	h.broadcastSystem(fmt.Sprintf("%s is now %s", oldName, newName))
}

func handleWS(h *hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := normalizeName(r.URL.Query().Get("name"))
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		c := &client{conn: conn, name: name}
		h.add(c)
		defer func() {
			h.remove(c)
			conn.Close()
		}()

		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				return
			}
			var incoming struct {
				Type string `json:"type"`
				Text string `json:"text"`
				Name string `json:"name"`
			}
			if json.Unmarshal(data, &incoming) != nil {
				continue
			}
			switch incoming.Type {
			case "rename":
				name := normalizeName(incoming.Name)
				if name != c.name {
					h.rename(c, name)
				}
			case "chat":
				text, ok := normalizeText(incoming.Text)
				if !ok {
					continue
				}
				out, _ := json.Marshal(map[string]string{
					"type": "chat",
					"from": c.name,
					"text": text,
				})
				h.broadcast(out)
			}
		}
	}
}

func main() {
	addr := flag.String("addr", ":8080", "listen address")
	flag.Parse()

	h := newHub()
	http.Handle("/", http.FileServer(http.Dir("web")))
	http.HandleFunc("/ws", handleWS(h))

	log.Printf("tinychat listening on %s", *addr)
	log.Fatal(http.ListenAndServe(*addr, nil))
}

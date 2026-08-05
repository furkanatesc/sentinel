package ws

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"

	cws "github.com/coder/websocket"
)

// message, /ws üzerinden gönderilen zarf: {topic, payload}.
type message struct {
	Topic   string `json:"topic"`
	Payload any    `json:"payload"`
}

type client struct {
	send chan message // bounded; dolu ise mesaj düşürülür (yavaş client)
}

// Hub, bağlı frontend WS client'larına topic bazlı yayın yapar (SRP: yalnız fan-out).
type Hub struct {
	register   chan *client
	unregister chan *client
	broadcast  chan message
	mu         sync.RWMutex
	clients    map[*client]struct{}
}

func NewHub() *Hub {
	return &Hub{
		register:   make(chan *client),
		unregister: make(chan *client),
		broadcast:  make(chan message, 256),
		clients:    map[*client]struct{}{},
	}
}

func (h *Hub) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case c := <-h.register:
			h.mu.Lock()
			h.clients[c] = struct{}{}
			h.mu.Unlock()
		case c := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[c]; ok {
				delete(h.clients, c)
				close(c.send)
			}
			h.mu.Unlock()
		case msg := <-h.broadcast:
			h.mu.RLock()
			for c := range h.clients {
				select {
				case c.send <- msg:
				default: // yavaş client → mesajı düşür (bounded)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Broadcast, ingest.Broadcaster arayüzünü karşılar.
func (h *Hub) Broadcast(topic string, payload any) {
	select {
	case h.broadcast <- message{Topic: topic, Payload: payload}:
	default: // hub kuyruğu dolu → düşür (backpressure)
	}
}

func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// ServeWS, /ws bağlantısını kabul eder; client'ı kaydeder ve yazma-pump çalıştırır.
func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request) {
	conn, err := cws.Accept(w, r, &cws.AcceptOptions{InsecureSkipVerify: true}) // CORS: origin kontrolü ayrı
	if err != nil {
		return
	}
	c := &client{send: make(chan message, 64)}
	h.register <- c
	defer func() { h.unregister <- c }()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			conn.Close(cws.StatusNormalClosure, "")
			return
		case msg, ok := <-c.send:
			if !ok {
				conn.Close(cws.StatusNormalClosure, "")
				return
			}
			b, _ := json.Marshal(msg)
			if err := conn.Write(ctx, cws.MessageText, b); err != nil {
				return
			}
		}
	}
}

// --- test yardımcıları (aynı pakette; ağsız register/unregister) ---
func (h *Hub) registerForTest() *client {
	c := &client{send: make(chan message, 8)}
	h.register <- c
	return c
}
func (h *Hub) unregisterForTest(c *client) { h.unregister <- c }

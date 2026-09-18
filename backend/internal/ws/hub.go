package ws

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Configurado para permitir conexões do frontend/mobile
	},
}

type MessageType string

const (
	MsgJoinTable   MessageType = "JOIN_TABLE"
	MsgLeaveTable  MessageType = "LEAVE_TABLE"
	MsgPlayerAct   MessageType = "PLAYER_ACTION"
	MsgTableState  MessageType = "TABLE_STATE"
	MsgChatMessage MessageType = "CHAT_MESSAGE"
	MsgPing        MessageType = "PING"
	MsgPong        MessageType = "PONG"
)

type WSMessage struct {
	Type      MessageType     `json:"type"`
	TableID   *uuid.UUID      `json:"table_id,omitempty"`
	UserID    *uuid.UUID      `json:"user_id,omitempty"`
	Payload   json.RawMessage `json:"payload,omitempty"`
	Timestamp int64           `json:"timestamp"`
}

type Client struct {
	Hub      *Hub
	Conn     *websocket.Conn
	Send     chan []byte
	UserID   uuid.UUID
	Nome     string
	TableID  *uuid.UUID
}

type Hub struct {
	clients    map[*Client]bool
	tables     map[uuid.UUID]map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		tables:     make(map[uuid.UUID]map[*Client]bool),
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("Cliente WS conectado: %s (User: %s)", client.Conn.RemoteAddr(), client.Nome)

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.Send)
				if client.TableID != nil {
					if tbl, exists := h.tables[*client.TableID]; exists {
						delete(tbl, client)
						if len(tbl) == 0 {
							delete(h.tables, *client.TableID)
						}
					}
				}
			}
			h.mu.Unlock()
			log.Printf("Cliente WS desconectado: %s", client.Nome)

		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (h *Hub) BroadcastAll(message []byte) {
	h.broadcast <- message
}

func (h *Hub) BroadcastToTable(tableID uuid.UUID, message []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if clients, ok := h.tables[tableID]; ok {
		for client := range clients {
			select {
			case client.Send <- message:
			default:
				close(client.Send)
				delete(clients, client)
			}
		}
	}
}

func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request, userID uuid.UUID, userName string) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Erro no upgrade WS: %v", err)
		return
	}

	client := &Client{
		Hub:    h,
		Conn:   conn,
		Send:   make(chan []byte, 256),
		UserID: userID,
		Nome:   userName,
	}

	h.register <- client

	go client.writePump()
	go client.readPump()
}

func (c *Client) readPump() {
	defer func() {
		c.Hub.unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(4096)
	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Erro de leitura WS: %v", err)
			}
			break
		}

		var msg WSMessage
		if err := json.Unmarshal(message, &msg); err == nil {
			msg.Timestamp = time.Now().Unix()
			msg.UserID = &c.UserID

			if msg.Type == MsgPing {
				pongMsg, _ := json.Marshal(WSMessage{
					Type:      MsgPong,
					Timestamp: time.Now().Unix(),
				})
				c.Send <- pongMsg
			}
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			n := len(c.Send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.Send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

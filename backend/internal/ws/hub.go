package ws

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/walissonpaulo/poker-dos-amigos-backend/internal/engine"
	"github.com/walissonpaulo/poker-dos-amigos-backend/internal/models"
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
	MsgJoinTable    MessageType = "JOIN_TABLE"
	MsgLeaveTable   MessageType = "LEAVE_TABLE"
	MsgPlayerAct    MessageType = "PLAYER_ACTION"
	MsgTableState   MessageType = "TABLE_STATE"
	MsgPrivateCards MessageType = "PRIVATE_CARDS"
	MsgChatMessage  MessageType = "CHAT_MESSAGE"
	MsgPing         MessageType = "PING"
	MsgPong         MessageType = "PONG"
)

type WSMessage struct {
	Type      MessageType     `json:"type"`
	TableID   *uuid.UUID      `json:"table_id,omitempty"`
	UserID    *uuid.UUID      `json:"user_id,omitempty"`
	Payload   json.RawMessage `json:"payload,omitempty"`
	Timestamp int64           `json:"timestamp"`
}

type JoinTablePayload struct {
	TableID    string `json:"table_id"`
	SeatNumber int    `json:"seat_number"`
	BuyIn      int64  `json:"buy_in"`
}

type PlayerActionPayload struct {
	Action string `json:"action"`
	Amount int64  `json:"amount,omitempty"`
}

type PrivateCardsPayload struct {
	Cards []models.Card `json:"cards"`
}

type Client struct {
	Hub     *Hub
	Conn    *websocket.Conn
	Send    chan []byte
	UserID  uuid.UUID
	Nome    string
	TableID *uuid.UUID
}

type Hub struct {
	clients     map[*Client]bool
	tables      map[uuid.UUID]map[*Client]bool
	broadcast   chan []byte
	register    chan *Client
	unregister  chan *Client
	gameService *engine.GameService
	mu          sync.RWMutex
}

func NewHub(gameService *engine.GameService) *Hub {
	return &Hub{
		clients:     make(map[*Client]bool),
		tables:      make(map[uuid.UUID]map[*Client]bool),
		broadcast:   make(chan []byte),
		register:    make(chan *Client),
		unregister:  make(chan *Client),
		gameService: gameService,
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
					tid := *client.TableID
					if tbl, exists := h.tables[tid]; exists {
						delete(tbl, client)
						if len(tbl) == 0 {
							delete(h.tables, tid)
						}
					}
					// Notifica GameService da saída
					if h.gameService != nil {
						if table, ok := h.gameService.GetTable(tid); ok {
							table.LeavePlayer(client.UserID)
							go h.broadcastCurrentTableState(tid, table)
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
	h.mu.Lock()
	defer h.mu.Unlock()

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

func (h *Hub) BroadcastTableState(tableID uuid.UUID, state engine.TableStatePayload) {
	payload, err := json.Marshal(state)
	if err != nil {
		return
	}
	msg, err := json.Marshal(WSMessage{
		Type:      MsgTableState,
		TableID:   &tableID,
		Payload:   payload,
		Timestamp: time.Now().Unix(),
	})
	if err == nil {
		h.BroadcastToTable(tableID, msg)
	}
}

func (h *Hub) SendPrivateCardsToUser(tableID uuid.UUID, userID uuid.UUID, cards []models.Card) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if clients, ok := h.tables[tableID]; ok {
		for client := range clients {
			if client.UserID == userID {
				payload, _ := json.Marshal(PrivateCardsPayload{Cards: cards})
				msg, _ := json.Marshal(WSMessage{
					Type:      MsgPrivateCards,
					TableID:   &tableID,
					UserID:    &userID,
					Payload:   payload,
					Timestamp: time.Now().Unix(),
				})
				select {
				case client.Send <- msg:
				default:
				}
				break
			}
		}
	}
}

func (h *Hub) broadcastCurrentTableState(tableID uuid.UUID, table *engine.TableGame) {
	state := table.GetPublicState()
	h.BroadcastTableState(tableID, state)

	// Envia cartas privadas para cada jogador individualmente
	for _, p := range state.Players {
		if p.UserID != nil {
			cards := table.GetPrivateCards(*p.UserID)
			if cards != nil {
				h.SendPrivateCardsToUser(tableID, *p.UserID, cards)
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

			switch msg.Type {
			case MsgPing:
				pongMsg, _ := json.Marshal(WSMessage{
					Type:      MsgPong,
					Timestamp: time.Now().Unix(),
				})
				c.Send <- pongMsg

			case MsgJoinTable:
				var joinPayload JoinTablePayload
				if err := json.Unmarshal(msg.Payload, &joinPayload); err == nil {
					tid, err := uuid.Parse(joinPayload.TableID)
					if err == nil {
						c.handleJoinTable(tid, joinPayload.SeatNumber, joinPayload.BuyIn)
					}
				}

			case MsgLeaveTable:
				c.handleLeaveTable()

			case MsgPlayerAct:
				var actPayload PlayerActionPayload
				if err := json.Unmarshal(msg.Payload, &actPayload); err == nil {
					c.handlePlayerAction(actPayload.Action, actPayload.Amount)
				}
			}
		}
	}
}

func (c *Client) handleJoinTable(tableID uuid.UUID, seatNumber int, buyIn int64) {
	c.Hub.mu.Lock()
	// Se já estava em outra mesa, remove dela
	if c.TableID != nil && *c.TableID != tableID {
		oldTid := *c.TableID
		if tbl, exists := c.Hub.tables[oldTid]; exists {
			delete(tbl, c)
			if len(tbl) == 0 {
				delete(c.Hub.tables, oldTid)
			}
		}
	}

	c.TableID = &tableID
	if _, exists := c.Hub.tables[tableID]; !exists {
		c.Hub.tables[tableID] = make(map[*Client]bool)
	}
	c.Hub.tables[tableID][c] = true
	c.Hub.mu.Unlock()

	log.Printf("Jogador %s (%s) juntou-se à mesa %s no assento %d", c.Nome, c.UserID, tableID, seatNumber)

	if c.Hub.gameService != nil {
		table := c.Hub.gameService.GetOrCreateTable(tableID, 25, 50)
		if seatNumber > 0 {
			if buyIn <= 0 {
				buyIn = 2500
			}
			_ = table.JoinPlayer(c.UserID, c.Nome, seatNumber, buyIn)
		}
		c.Hub.broadcastCurrentTableState(tableID, table)
	}
}

func (c *Client) handleLeaveTable() {
	if c.TableID == nil {
		return
	}
	tableID := *c.TableID

	c.Hub.mu.Lock()
	if tbl, exists := c.Hub.tables[tableID]; exists {
		delete(tbl, c)
		if len(tbl) == 0 {
			delete(c.Hub.tables, tableID)
		}
	}
	c.TableID = nil
	c.Hub.mu.Unlock()

	log.Printf("Jogador %s (%s) saiu da mesa %s", c.Nome, c.UserID, tableID)

	if c.Hub.gameService != nil {
		if table, ok := c.Hub.gameService.GetTable(tableID); ok {
			table.LeavePlayer(c.UserID)
			c.Hub.broadcastCurrentTableState(tableID, table)
		}
	}
}

func (c *Client) handlePlayerAction(action string, amount int64) {
	if c.TableID == nil || c.Hub.gameService == nil {
		return
	}
	tableID := *c.TableID

	table, ok := c.Hub.gameService.GetTable(tableID)
	if !ok {
		return
	}

	actType := engine.ActionType(action)
	err := table.ProcessAction(c.UserID, actType, amount)
	if err != nil {
		log.Printf("Ação inválida do jogador %s: %v", c.Nome, err)
		return
	}

	c.Hub.broadcastCurrentTableState(tableID, table)
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

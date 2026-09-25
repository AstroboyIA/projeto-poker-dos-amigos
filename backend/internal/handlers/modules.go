package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/walissonpaulo/poker-dos-amigos-backend/internal/auth"
	"github.com/walissonpaulo/poker-dos-amigos-backend/internal/models"
	"github.com/walissonpaulo/poker-dos-amigos-backend/internal/ws"
	"github.com/walissonpaulo/poker-dos-amigos-backend/pkg/response"
)

type ModulesHandler struct {
	hub         *ws.Hub
	authHandler *AuthHandler
	tables      []models.PokerTable
	mu          sync.RWMutex
}

func NewModulesHandler(hub *ws.Hub, authHandler *AuthHandler) *ModulesHandler {
	seedTables := []models.PokerTable{
		{
			ID:            uuid.MustParse("a1111111-1111-1111-1111-111111111111"),
			Nome:          "Mesa VIP Ouro #01 (Texas Hold'em)",
			Tipo:          models.TableTypeCashGame,
			SmallBlind:    25,
			BigBlind:      50,
			BuyInMin:      1000,
			BuyInMax:      5000,
			MaxSeats:      9,
			Status:        models.TableStatusRunning,
			CurrentPot:    1450,
			BotSeats:      []int{3, 5, 7},
			OccupiedSeats: []int{1, 2, 4, 6},
			CreatedBy:     "Clube",
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		},
		{
			ID:            uuid.MustParse("a2222222-2222-2222-2222-222222222222"),
			Nome:          "Mesa dos Amigos Fechada",
			Tipo:          models.TableTypeCashGame,
			SmallBlind:    10,
			BigBlind:      20,
			BuyInMin:      400,
			BuyInMax:      2000,
			MaxSeats:      9,
			Status:        models.TableStatusRunning,
			CurrentPot:    320,
			BotSeats:      []int{2, 8},
			OccupiedSeats: []int{1, 4, 6},
			Password:      "1234",
			CreatedBy:     "Felipe",
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		},
		{
			ID:            uuid.MustParse("a3333333-3333-3333-3333-333333333333"),
			Nome:          "Mesa Torneio Amigos Final Table",
			Tipo:          models.TableTypeTournament,
			SmallBlind:    500,
			BigBlind:      1000,
			BuyInMin:      0,
			BuyInMax:      0,
			MaxSeats:      9,
			Status:        models.TableStatusRunning,
			CurrentPot:    28000,
			BotSeats:      []int{},
			OccupiedSeats: []int{1, 2, 3, 4, 5, 6, 7, 8, 9},
			CreatedBy:     "Diretoria",
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		},
	}

	return &ModulesHandler{
		hub:         hub,
		authHandler: authHandler,
		tables:      seedTables,
	}
}

func (h *ModulesHandler) broadcastTablesUpdate() {
	if h.hub == nil {
		return
	}
	h.mu.RLock()
	tablesCopy := make([]models.PokerTable, len(h.tables))
	copy(tablesCopy, h.tables)
	h.mu.RUnlock()

	payload, err := json.Marshal(tablesCopy)
	if err != nil {
		return
	}

	msg, err := json.Marshal(ws.WSMessage{
		Type:      "TABLES_UPDATED",
		Payload:   payload,
		Timestamp: time.Now().Unix(),
	})
	if err == nil {
		h.hub.BroadcastAll(msg)
	}
}

// Lista de Torneios
func (h *ModulesHandler) GetTournaments(w http.ResponseWriter, r *http.Request) {
	tournaments := []models.Tournament{
		{
			ID:            uuid.New(),
			Nome:          "🏆 Super High Roller Semanal",
			BuyIn:         500,
			Garantido:     25000,
			Inscritos:     32,
			MaxInscritos:  50,
			DataInicio:    time.Now().Add(2 * time.Hour),
			Status:        "aberto",
			BlindInterval: 15,
		},
		{
			ID:            uuid.New(),
			Nome:          "🔥 Torneio dos Amigos DeepStack",
			BuyIn:         100,
			Garantido:     10000,
			Inscritos:     45,
			MaxInscritos:  100,
			DataInicio:    time.Now().Add(6 * time.Hour),
			Status:        "aberto",
			BlindInterval: 12,
		},
		{
			ID:            uuid.New(),
			Nome:          "⚡ Turbo Knockout 6-Max",
			BuyIn:         150,
			Garantido:     8000,
			Inscritos:     18,
			MaxInscritos:  36,
			DataInicio:    time.Now().Add(24 * time.Hour),
			Status:        "aberto",
			BlindInterval: 8,
		},
	}
	response.JSON(w, http.StatusOK, tournaments)
}

// Ranking Geral do Clube
func (h *ModulesHandler) GetRankings(w http.ResponseWriter, r *http.Request) {
	rankings := []models.RankingEntry{
		{Posicao: 1, UserID: uuid.New(), Nome: "Jonatas (Sócio)", Pontos: 4850, TorneiosGan: 7, LucroTotal: 34200},
		{Posicao: 2, UserID: uuid.New(), Nome: "Felipe (Sócio)", Pontos: 4620, TorneiosGan: 6, LucroTotal: 31500},
		{Posicao: 3, UserID: uuid.New(), Nome: "Alex 'PokerKing'", Pontos: 3980, TorneiosGan: 4, LucroTotal: 22400},
		{Posicao: 4, UserID: uuid.New(), Nome: "Bruno 'AllIn'", Pontos: 3450, TorneiosGan: 3, LucroTotal: 18900},
		{Posicao: 5, UserID: uuid.New(), Nome: "Carlos Shark", Pontos: 3120, TorneiosGan: 3, LucroTotal: 15400},
	}
	response.JSON(w, http.StatusOK, rankings)
}

// Lista de Mesas / Cash Games
func (h *ModulesHandler) GetTables(w http.ResponseWriter, r *http.Request) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	response.JSON(w, http.StatusOK, h.tables)
}

type CreateTableRequest struct {
	Nome          string `json:"nome"`
	SmallBlind    int64  `json:"small_blind"`
	BigBlind      int64  `json:"big_blind"`
	BuyInMin      int64  `json:"buy_in_min"`
	BuyInMax      int64  `json:"buy_in_max"`
	MaxSeats      int    `json:"max_seats"`
	BotSeats      []int  `json:"bot_seats"`
	OccupiedSeats []int  `json:"occupied_seats"`
	Password      string `json:"password,omitempty"`
}

func (h *ModulesHandler) CreateTable(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(auth.UserContextKey).(*auth.Claims)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "Não autenticado")
		return
	}

	var req CreateTableRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Dados inválidos")
		return
	}

	if req.Nome == "" {
		req.Nome = "Nova Mesa Cash Game"
	}
	if req.MaxSeats <= 0 {
		req.MaxSeats = 9
	}
	if req.BotSeats == nil {
		req.BotSeats = []int{}
	}
	if req.OccupiedSeats == nil {
		req.OccupiedSeats = []int{}
	}

	newTable := models.PokerTable{
		ID:            uuid.New(),
		Nome:          req.Nome,
		Tipo:          models.TableTypeCashGame,
		SmallBlind:    req.SmallBlind,
		BigBlind:      req.BigBlind,
		BuyInMin:      req.BuyInMin,
		BuyInMax:      req.BuyInMax,
		MaxSeats:      req.MaxSeats,
		Status:        models.TableStatusWaiting,
		CurrentPot:    0,
		BotSeats:      req.BotSeats,
		OccupiedSeats: req.OccupiedSeats,
		Password:      req.Password,
		CreatedBy:     claims.Nome,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	h.mu.Lock()
	h.tables = append([]models.PokerTable{newTable}, h.tables...)
	h.mu.Unlock()

	h.broadcastTablesUpdate()

	response.JSON(w, http.StatusCreated, newTable)
}

type SeatActionRequest struct {
	TableID    string `json:"table_id"`
	SeatNumber int    `json:"seat_number"`
}

func (h *ModulesHandler) OccupySeat(w http.ResponseWriter, r *http.Request) {
	var req SeatActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Dados inválidos")
		return
	}

	h.mu.Lock()

	for i, t := range h.tables {
		if t.ID.String() == req.TableID {
			if req.SeatNumber < 1 || req.SeatNumber > t.MaxSeats {
				h.mu.Unlock()
				response.Error(w, http.StatusBadRequest, fmt.Sprintf("Assento deve estar entre 1 e %d", t.MaxSeats))
				return
			}

			for _, s := range t.BotSeats {
				if s == req.SeatNumber {
					h.mu.Unlock()
					response.Error(w, http.StatusConflict, "Assento reservado para bot")
					return
				}
			}

			for _, s := range t.OccupiedSeats {
				if s == req.SeatNumber {
					h.mu.Unlock()
					response.Error(w, http.StatusConflict, "Assento já está ocupado")
					return
				}
			}

			h.tables[i].OccupiedSeats = append(h.tables[i].OccupiedSeats, req.SeatNumber)
			h.tables[i].UpdatedAt = time.Now()
			updatedTable := h.tables[i]
			h.mu.Unlock()
			h.broadcastTablesUpdate()
			response.JSON(w, http.StatusOK, updatedTable)
			return
		}
	}

	h.mu.Unlock()
	response.Error(w, http.StatusNotFound, "Mesa não encontrada")
}

func (h *ModulesHandler) LeaveSeat(w http.ResponseWriter, r *http.Request) {
	var req SeatActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Dados inválidos")
		return
	}

	if !h.releaseSeat(req.TableID, req.SeatNumber) {
		response.Error(w, http.StatusNotFound, "Mesa não encontrada")
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ReleaseSeat is also used by the websocket hub when a client disconnects.
// It is intentionally idempotent so an explicit leave followed by disconnect
// cannot leave stale occupancy or fail the cleanup.
func (h *ModulesHandler) ReleaseSeat(tableID uuid.UUID, seatNumber int) {
	h.releaseSeat(tableID.String(), seatNumber)
}

func (h *ModulesHandler) releaseSeat(tableID string, seatNumber int) bool {
	h.mu.Lock()

	for i, t := range h.tables {
		if t.ID.String() != tableID {
			continue
		}

		newOccupied := make([]int, 0, len(t.OccupiedSeats))
		for _, s := range t.OccupiedSeats {
			if s != seatNumber {
				newOccupied = append(newOccupied, s)
			}
		}

		changed := len(newOccupied) != len(t.OccupiedSeats)
		if changed {
			h.tables[i].OccupiedSeats = newOccupied
			h.tables[i].UpdatedAt = time.Now()
		}

		if len(newOccupied) == 0 && t.CreatedBy != "Clube" && t.CreatedBy != "Diretoria" {
			h.tables = append(h.tables[:i], h.tables[i+1:]...)
			changed = true
		}

		h.mu.Unlock()
		if changed {
			h.broadcastTablesUpdate()
		}
		return true
	}

	h.mu.Unlock()
	return false
}

type ChipTransactionRequest struct {
	Amount int64 `json:"amount"`
}

func (h *ModulesHandler) BuyIn(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(auth.UserContextKey).(*auth.Claims)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "Não autenticado")
		return
	}

	var req ChipTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Amount <= 0 {
		response.Error(w, http.StatusBadRequest, "Valor de buy-in inválido")
		return
	}

	user, exists := h.authHandler.GetUserByEmail(claims.Email)
	if !exists {
		response.Error(w, http.StatusNotFound, "Usuário não encontrado")
		return
	}

	if user.SaldoFichas < req.Amount {
		response.Error(w, http.StatusBadRequest, "Saldo insuficiente para o buy-in")
		return
	}

	updatedUser, _ := h.authHandler.UpdateChips(claims.Email, -req.Amount)
	response.JSON(w, http.StatusOK, updatedUser)
}

func (h *ModulesHandler) CashOut(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(auth.UserContextKey).(*auth.Claims)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "Não autenticado")
		return
	}

	var req ChipTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Amount < 0 {
		response.Error(w, http.StatusBadRequest, "Valor de cash-out inválido")
		return
	}

	updatedUser, _ := h.authHandler.UpdateChips(claims.Email, req.Amount)
	response.JSON(w, http.StatusOK, updatedUser)
}

// Comunicados Oficiais do Clube
func (h *ModulesHandler) GetAnnouncements(w http.ResponseWriter, r *http.Request) {
	announcements := []models.Announcement{
		{
			ID:        uuid.New(),
			Titulo:    "🎉 Inauguração Oficial do Club Poker dos Amigos",
			Conteudo:  "Sejam bem-vindos ao sistema oficial! Aproveitem os novos torneios com garantia semanal e sistema exclusivo de embaralhamento contínuo.",
			Autor:     "Diretoria (Jonatas e Felipe)",
			Publicado: true,
			CreatedAt: time.Now().Add(-48 * time.Hour),
		},
		{
			ID:        uuid.New(),
			Titulo:    "🏆 Regulamento da Etapa de Inverno",
			Conteudo:  "As inscrições para a Etapa de Inverno estão abertas a todos os membros ativos no ranking.",
			Autor:     "Gerência",
			Publicado: true,
			CreatedAt: time.Now().Add(-24 * time.Hour),
		},
	}
	response.JSON(w, http.StatusOK, announcements)
}

// Financeiro (Gerencial)
func (h *ModulesHandler) GetFinancialSummary(w http.ResponseWriter, r *http.Request) {
	summary := map[string]interface{}{
		"saldo_caixa_total":     145890.00,
		"entradas_mes":          58200.00,
		"saidas_premios_mes":    39500.00,
		"rake_arrecadado":       8450.00,
		"jogadores_com_saldo":   128,
		"total_transacoes_hoje": 34,
	}
	response.JSON(w, http.StatusOK, summary)
}

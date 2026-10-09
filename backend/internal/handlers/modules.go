package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/walissonpaulo/poker-dos-amigos-backend/internal/auth"
	"github.com/walissonpaulo/poker-dos-amigos-backend/internal/database"
	"github.com/walissonpaulo/poker-dos-amigos-backend/internal/engine"
	"github.com/walissonpaulo/poker-dos-amigos-backend/internal/finance"
	"github.com/walissonpaulo/poker-dos-amigos-backend/internal/models"
	"github.com/walissonpaulo/poker-dos-amigos-backend/internal/ws"
	"github.com/walissonpaulo/poker-dos-amigos-backend/pkg/response"
)

type ModulesHandler struct {
	hub         *ws.Hub
	authHandler *AuthHandler
	tables      []models.PokerTable
	mu          sync.RWMutex
	wallets     *finance.Service
	store       *database.Store
}

func NewModulesHandler(hub *ws.Hub, authHandler *AuthHandler) *ModulesHandler {
	return &ModulesHandler{
		hub:         hub,
		authHandler: authHandler,
		tables:      make([]models.PokerTable, 0),
		wallets:     finance.NewService(),
	}
}

func NewModulesHandlerWithStore(hub *ws.Hub, authHandler *AuthHandler, store *database.Store) *ModulesHandler {
	handler := NewModulesHandler(hub, authHandler)
	handler.store = store
	return handler
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
	if h.store != nil {
		tables, err := h.store.ListTables(r.Context())
		if err != nil {
			log.Printf("Falha ao consultar mesas compartilhadas: %v", err)
			response.Error(w, http.StatusInternalServerError, "Falha ao carregar mesas")
			return
		}
		response.JSON(w, http.StatusOK, tables)
		return
	}
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
		OccupiedSeats: req.OccupiedSeats,
		Password:      req.Password,
		CreatedBy:     claims.Nome,
		CreatorUserID: claims.UserID,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if h.store != nil {
		if err := h.store.CreateTable(r.Context(), newTable); err != nil {
			log.Printf("Falha ao salvar mesa no PostgreSQL: %v", err)
			response.Error(w, http.StatusInternalServerError, "Falha ao criar mesa")
			return
		}
		response.JSON(w, http.StatusCreated, newTable)
		return
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
	if h.authHandler != nil {
		if _, ok := r.Context().Value(auth.UserContextKey).(*auth.Claims); !ok {
			response.Error(w, http.StatusUnauthorized, "Não autenticado")
			return
		}
	}
	var req SeatActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Dados inválidos")
		return
	}

	if h.store != nil {
		tableID, err := uuid.Parse(req.TableID)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "Mesa inválida")
			return
		}
		table, err := h.store.ReserveSeat(r.Context(), tableID, req.SeatNumber)
		switch {
		case errors.Is(err, database.ErrTableNotFound):
			response.Error(w, http.StatusNotFound, "Mesa não encontrada")
		case errors.Is(err, database.ErrSeatOccupied):
			response.Error(w, http.StatusConflict, "Assento já está ocupado")
		case errors.Is(err, database.ErrInvalidSeat):
			response.Error(w, http.StatusBadRequest, err.Error())
		case err != nil:
			log.Printf("Falha ao reservar assento no PostgreSQL: %v", err)
			response.Error(w, http.StatusInternalServerError, "Falha ao ocupar assento")
		default:
			response.JSON(w, http.StatusOK, table)
		}
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
	claims, ok := r.Context().Value(auth.UserContextKey).(*auth.Claims)
	if h.authHandler != nil {
		if !ok {
			response.Error(w, http.StatusUnauthorized, "Não autenticado")
			return
		}
	}
	var req SeatActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Dados inválidos")
		return
	}

	if h.store != nil {
		tableID, err := uuid.Parse(req.TableID)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "Mesa inválida")
			return
		}
		var stack int64
		if h.hub != nil && h.hub.GameService() != nil {
			game, updateErr := h.hub.UpdateGame(tableID, func(game *engine.TableGame) error {
				stack, _ = game.LeavePlayerWithStack(claims.UserID)
				return nil
			})
			if updateErr != nil {
				log.Printf("Falha ao atualizar partida compartilhada ao sair: %v", updateErr)
				if errors.Is(updateErr, database.ErrTableNotFound) {
					response.Error(w, http.StatusNotFound, "Mesa não encontrada")
				} else {
					response.Error(w, http.StatusInternalServerError, "Falha ao sair da mesa")
				}
				return
			}
			h.hub.GameService().SetTable(game)
		}
		if !h.releaseSeat(req.TableID, req.SeatNumber) {
			response.Error(w, http.StatusNotFound, "Mesa não encontrada")
			return
		}
		if stack > 0 && h.authHandler != nil {
			if user, exists := h.authHandler.GetUserByIdentity(claims.Username, claims.Email); exists {
				h.wallets.EnsureWallet(user.ID, user.SaldoFichas)
				wallet, returnErr := h.wallets.ReturnFromTable(user.ID, finance.ChipsToMoney(stack), req.TableID, claims.UserID.String()+":"+req.TableID+":return")
				if returnErr != nil {
					log.Printf("Falha ao devolver saldo da mesa para o usuário %s: %v", user.ID, returnErr)
					response.Error(w, http.StatusInternalServerError, "Assento liberado; retorno do saldo precisa de suporte")
					return
				}
				user.SaldoFichas = finance.MoneyToChips(wallet.AvailableCents)
				user.Wallet = &models.WalletSummary{BalanceCents: wallet.BalanceCents, AvailableCents: wallet.AvailableCents, ReservedCents: wallet.ReservedCents}
			}
		}
		response.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
		return
	}

	if !h.releaseSeat(req.TableID, req.SeatNumber) {
		response.Error(w, http.StatusNotFound, "Mesa não encontrada")
		return
	}
	if h.hub != nil {
		if tableID, err := uuid.Parse(req.TableID); err == nil {
			if table, exists := h.hub.GameService().GetTable(tableID); exists {
				if stack, removed := table.LeavePlayerWithStack(claims.UserID); removed {
					if user, exists := h.authHandler.GetUserByIdentity(claims.Username, claims.Email); exists {
						h.wallets.EnsureWallet(user.ID, user.SaldoFichas)
						wallet, err := h.wallets.ReturnFromTable(user.ID, finance.ChipsToMoney(stack), req.TableID, claims.UserID.String()+":"+req.TableID+":return")
						if err == nil {
							user.SaldoFichas = finance.MoneyToChips(wallet.AvailableCents)
							user.Wallet = &models.WalletSummary{BalanceCents: wallet.BalanceCents, AvailableCents: wallet.AvailableCents, ReservedCents: wallet.ReservedCents}
						}
					}
				}
			}
		}
	}
	response.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *ModulesHandler) GetWallet(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(auth.UserContextKey).(*auth.Claims)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "Não autenticado")
		return
	}
	user, exists := h.authHandler.GetUserByIdentity(claims.Username, claims.Email)
	if !exists {
		response.Error(w, http.StatusNotFound, "Usuário não encontrado")
		return
	}
	wallet := h.wallets.EnsureWallet(user.ID, user.SaldoFichas)
	response.JSON(w, http.StatusOK, map[string]interface{}{
		"wallet": wallet,
		"ledger": h.wallets.Entries(user.ID),
	})
}

// DevDeposit is deliberately named and routed as a development-only mock.
// It must be replaced by a verified payment webhook before real money is enabled.
func (h *ModulesHandler) DevDeposit(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(auth.UserContextKey).(*auth.Claims)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "Não autenticado")
		return
	}
	var req struct {
		AmountCents int64 `json:"amount_cents"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.AmountCents <= 0 {
		response.Error(w, http.StatusBadRequest, "Valor de depósito inválido")
		return
	}
	user, exists := h.authHandler.GetUserByIdentity(claims.Username, claims.Email)
	if !exists {
		response.Error(w, http.StatusNotFound, "Usuário não encontrado")
		return
	}
	h.wallets.EnsureWallet(user.ID, user.SaldoFichas)
	wallet, err := h.wallets.Deposit(user.ID, req.AmountCents, r.Header.Get("Idempotency-Key"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	user.SaldoFichas = finance.MoneyToChips(wallet.AvailableCents)
	user.Wallet = &models.WalletSummary{BalanceCents: wallet.BalanceCents, AvailableCents: wallet.AvailableCents, ReservedCents: wallet.ReservedCents}
	response.JSON(w, http.StatusOK, user)
}

// ReleaseSeat is also used by the websocket hub when a client disconnects.
// It is intentionally idempotent so an explicit leave followed by disconnect
// cannot leave stale occupancy or fail the cleanup.
func (h *ModulesHandler) ReleaseSeat(tableID uuid.UUID, seatNumber int) {
	h.releaseSeat(tableID.String(), seatNumber)
}

func (h *ModulesHandler) releaseSeat(tableID string, seatNumber int) bool {
	if h.store != nil {
		id, err := uuid.Parse(tableID)
		if err != nil {
			return false
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		released, err := h.store.ReleaseSeat(ctx, id, seatNumber)
		if err != nil {
			log.Printf("Falha ao liberar assento no PostgreSQL: %v", err)
			return false
		}
		return released
	}
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

func (h *ModulesHandler) BuyIn(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(auth.UserContextKey).(*auth.Claims)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "Não autenticado")
		return
	}

	var req struct {
		Amount      int64  `json:"amount"`
		AmountCents int64  `json:"amount_cents"`
		TableID     string `json:"table_id"`
		SeatNumber  int    `json:"seat_number"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Valor de buy-in inválido")
		return
	}

	user, exists := h.authHandler.GetUserByIdentity(claims.Username, claims.Email)
	if !exists {
		response.Error(w, http.StatusNotFound, "Usuário não encontrado")
		return
	}

	amountCents := req.AmountCents
	if amountCents == 0 {
		amountCents = finance.ChipsToMoney(req.Amount)
	}
	if amountCents <= 0 {
		response.Error(w, http.StatusBadRequest, "Valor de buy-in inválido")
		return
	}

	h.wallets.EnsureWallet(user.ID, user.SaldoFichas)
	key := r.Header.Get("Idempotency-Key")
	if key == "" && req.TableID != "" {
		key = claims.UserID.String() + ":" + req.TableID + ":" + fmt.Sprint(req.SeatNumber)
	}
	if req.TableID != "" {
		if h.hub == nil || h.hub.GameService() == nil {
			response.Error(w, http.StatusServiceUnavailable, "Serviço da mesa indisponível")
			return
		}

		if h.store != nil {
			tableID, err := uuid.Parse(req.TableID)
			if err != nil {
				response.Error(w, http.StatusBadRequest, "Mesa inválida")
				return
			}
			tableConfig, err := h.store.GetTable(r.Context(), tableID)
			if errors.Is(err, database.ErrTableNotFound) {
				response.Error(w, http.StatusNotFound, "Mesa não encontrada")
				return
			}
			if err != nil {
				log.Printf("Falha ao carregar configuração da mesa: %v", err)
				response.Error(w, http.StatusInternalServerError, "Falha ao carregar mesa")
				return
			}
			if req.SeatNumber < 1 || req.SeatNumber > tableConfig.MaxSeats || req.Amount < tableConfig.BuyInMin || req.Amount > tableConfig.BuyInMax {
				response.Error(w, http.StatusBadRequest, "Buy-in ou assento inválido para esta mesa")
				return
			}

			wallet, err := h.wallets.BuyIn(user.ID, amountCents, req.TableID, key)
			if err != nil {
				if errors.Is(err, finance.ErrInsufficientFunds) {
					response.Error(w, http.StatusBadRequest, "Saldo insuficiente para o buy-in")
					return
				}
				response.Error(w, http.StatusBadRequest, err.Error())
				return
			}

			playerName := user.NomeCompleto
			if playerName == "" {
				playerName = user.Username
			}
			game, _, updateErr := h.store.JoinPlayer(r.Context(), tableID, user.ID, playerName, req.SeatNumber, req.Amount)
			if updateErr != nil {
				if _, refundErr := h.wallets.RefundBuyIn(user.ID, amountCents, req.TableID, key+":rollback"); refundErr != nil {
					log.Printf("Falha ao estornar buy-in após erro de persistência do jogo para %s: %v", user.ID, refundErr)
					response.Error(w, http.StatusInternalServerError, "Entrada recusada e estorno pendente; contate o suporte")
					return
				}
				switch {
				case errors.Is(updateErr, database.ErrSeatOccupied):
					response.Error(w, http.StatusConflict, "Assento já está ocupado")
				case errors.Is(updateErr, database.ErrTableNotFound):
					response.Error(w, http.StatusNotFound, "Mesa não encontrada")
				default:
					response.Error(w, http.StatusConflict, updateErr.Error())
				}
				return
			}

			h.hub.GameService().SetTable(game)
			h.broadcastTablesUpdate()
			h.hub.TableChanged(tableID, game)
			user.SaldoFichas = finance.MoneyToChips(wallet.AvailableCents)
			user.Wallet = &models.WalletSummary{BalanceCents: wallet.BalanceCents, AvailableCents: wallet.AvailableCents, ReservedCents: wallet.ReservedCents}
			response.JSON(w, http.StatusOK, user)
			return
		}

		h.mu.Lock()
		tableIndex := -1
		for i, table := range h.tables {
			if table.ID.String() == req.TableID {
				tableIndex = i
				if req.SeatNumber < 1 || req.SeatNumber > table.MaxSeats || req.Amount <= 0 || req.Amount < table.BuyInMin || req.Amount > table.BuyInMax {
					h.mu.Unlock()
					response.Error(w, http.StatusBadRequest, "Buy-in ou assento inválido para esta mesa")
					return
				}
				for _, seat := range table.OccupiedSeats {
					if seat == req.SeatNumber {
						h.mu.Unlock()
						response.Error(w, http.StatusConflict, "Assento já está ocupado")
						return
					}
				}
				break
			}
		}
		if tableIndex < 0 {
			h.mu.Unlock()
			response.Error(w, http.StatusNotFound, "Mesa não encontrada")
			return
		}
		tableConfig := h.tables[tableIndex]
		wallet, err := h.wallets.BuyIn(user.ID, amountCents, req.TableID, key)
		if err != nil {
			h.mu.Unlock()
			if err == finance.ErrInsufficientFunds {
				response.Error(w, http.StatusBadRequest, "Saldo insuficiente para o buy-in")
				return
			}
			response.Error(w, http.StatusBadRequest, err.Error())
			return
		}

		playerName := user.NomeCompleto
		if playerName == "" {
			playerName = user.Username
		}
		tableGame := h.hub.GameService().GetOrCreateTable(tableConfig.ID, tableConfig.SmallBlind, tableConfig.BigBlind)
		if err := tableGame.JoinPlayer(user.ID, playerName, req.SeatNumber, req.Amount); err != nil {
			h.mu.Unlock()
			if _, refundErr := h.wallets.RefundBuyIn(user.ID, amountCents, req.TableID, key+":rollback"); refundErr != nil {
				log.Printf("Falha ao estornar buy-in recusado do usuário %s: %v", user.ID, refundErr)
				response.Error(w, http.StatusInternalServerError, "Entrada recusada e estorno pendente; contate o suporte")
				return
			}
			response.Error(w, http.StatusConflict, err.Error())
			return
		}
		h.tables[tableIndex].OccupiedSeats = append(h.tables[tableIndex].OccupiedSeats, req.SeatNumber)
		h.tables[tableIndex].UpdatedAt = time.Now()
		h.mu.Unlock()
		h.broadcastTablesUpdate()
		h.hub.TableChanged(tableConfig.ID, tableGame)
		user.SaldoFichas = finance.MoneyToChips(wallet.AvailableCents)
		user.Wallet = &models.WalletSummary{BalanceCents: wallet.BalanceCents, AvailableCents: wallet.AvailableCents, ReservedCents: wallet.ReservedCents}
		response.JSON(w, http.StatusOK, user)
		return
	}
	wallet, err := h.wallets.BuyIn(user.ID, amountCents, req.TableID, key)
	if err != nil {
		if err == finance.ErrInsufficientFunds {
			response.Error(w, http.StatusBadRequest, "Saldo insuficiente para o buy-in")
			return
		}
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	user.SaldoFichas = finance.MoneyToChips(wallet.AvailableCents)
	user.Wallet = &models.WalletSummary{BalanceCents: wallet.BalanceCents, AvailableCents: wallet.AvailableCents, ReservedCents: wallet.ReservedCents}
	response.JSON(w, http.StatusOK, user)
}

func (h *ModulesHandler) Rebuy(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(auth.UserContextKey).(*auth.Claims)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "Não autenticado")
		return
	}
	var req struct {
		Amount  int64  `json:"amount"`
		TableID string `json:"table_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Amount <= 0 || req.TableID == "" {
		response.Error(w, http.StatusBadRequest, "Dados de recarga inválidos")
		return
	}
	user, exists := h.authHandler.GetUserByIdentity(claims.Username, claims.Email)
	if !exists {
		response.Error(w, http.StatusNotFound, "Usuário não encontrado")
		return
	}
	tableID, err := uuid.Parse(req.TableID)
	if err != nil || h.hub == nil || h.hub.GameService() == nil {
		response.Error(w, http.StatusBadRequest, "Mesa inválida")
		return
	}
	var tableConfig models.PokerTable
	if h.store != nil {
		tableConfig, err = h.store.GetTable(r.Context(), tableID)
		if errors.Is(err, database.ErrTableNotFound) {
			response.Error(w, http.StatusNotFound, "Mesa não encontrada")
			return
		}
		if err != nil {
			log.Printf("Falha ao carregar mesa para recarga: %v", err)
			response.Error(w, http.StatusInternalServerError, "Falha ao carregar mesa")
			return
		}
	} else {
		h.mu.RLock()
		found := false
		for i := range h.tables {
			if h.tables[i].ID == tableID {
				tableConfig = h.tables[i]
				found = true
				break
			}
		}
		h.mu.RUnlock()
		if !found {
			response.Error(w, http.StatusNotFound, "Mesa não encontrada")
			return
		}
	}
	if req.Amount < tableConfig.BuyInMin || req.Amount > tableConfig.BuyInMax {
		response.Error(w, http.StatusBadRequest, "Recarga fora dos limites de buy-in da mesa")
		return
	}

	h.wallets.EnsureWallet(user.ID, user.SaldoFichas)
	key := r.Header.Get("Idempotency-Key")
	if key == "" {
		key = uuid.NewString()
	}
	wallet, err := h.wallets.BuyIn(user.ID, finance.ChipsToMoney(req.Amount), req.TableID, key)
	if err != nil {
		if err == finance.ErrInsufficientFunds {
			response.Error(w, http.StatusBadRequest, "Saldo fora da mesa insuficiente para a recarga")
			return
		}
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	var table *engine.TableGame
	if h.store != nil {
		table, err = h.hub.UpdateGame(tableID, func(game *engine.TableGame) error {
			return game.AddStack(user.ID, req.Amount, key)
		})
	} else if table, exists = h.hub.GameService().GetTable(tableID); !exists {
		err = database.ErrTableNotFound
	} else {
		err = table.AddStack(user.ID, req.Amount, key)
	}
	if err != nil {
		if _, refundErr := h.wallets.RefundBuyIn(user.ID, finance.ChipsToMoney(req.Amount), req.TableID, key+":rollback"); refundErr != nil {
			log.Printf("Falha ao estornar recarga recusada do usuário %s: %v", user.ID, refundErr)
			response.Error(w, http.StatusInternalServerError, "Recarga recusada e estorno pendente; contate o suporte")
			return
		}
		if errors.Is(err, database.ErrTableNotFound) {
			response.Error(w, http.StatusNotFound, "Mesa não encontrada")
			return
		}
		response.Error(w, http.StatusConflict, err.Error())
		return
	}
	user.SaldoFichas = finance.MoneyToChips(wallet.AvailableCents)
	user.Wallet = &models.WalletSummary{BalanceCents: wallet.BalanceCents, AvailableCents: wallet.AvailableCents, ReservedCents: wallet.ReservedCents}
	h.hub.TableChanged(tableID, table)
	response.JSON(w, http.StatusOK, user)
}

func (h *ModulesHandler) CashOut(w http.ResponseWriter, r *http.Request) {
	response.Error(w, http.StatusGone, "Cash-out é calculado pelo servidor ao sair da mesa")
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

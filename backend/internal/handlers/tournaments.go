package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/walissonpaulo/poker-dos-amigos-backend/internal/auth"
	"github.com/walissonpaulo/poker-dos-amigos-backend/internal/database"
	"github.com/walissonpaulo/poker-dos-amigos-backend/internal/engine"
	"github.com/walissonpaulo/poker-dos-amigos-backend/internal/finance"
	"github.com/walissonpaulo/poker-dos-amigos-backend/internal/models"
	"github.com/walissonpaulo/poker-dos-amigos-backend/internal/ws"
	"github.com/walissonpaulo/poker-dos-amigos-backend/pkg/response"
)

type TournamentHandler struct {
	store       *database.Store
	hub         *ws.Hub
	authHandler *AuthHandler
}

func NewTournamentHandler(store *database.Store, hub *ws.Hub, authHandler *AuthHandler) *TournamentHandler {
	return &TournamentHandler{store: store, hub: hub, authHandler: authHandler}
}

func (h *TournamentHandler) GetTournaments(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(auth.UserContextKey).(*auth.Claims)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "Não autenticado")
		return
	}
	tournaments, err := h.store.ListTournaments(r.Context(), claims.UserID)
	if err != nil {
		log.Printf("Falha ao carregar torneios: %v", err)
		response.Error(w, http.StatusInternalServerError, "Falha ao carregar torneios")
		return
	}
	response.JSON(w, http.StatusOK, tournaments)
}

func (h *TournamentHandler) CreateTournament(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(auth.UserContextKey).(*auth.Claims)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "Não autenticado")
		return
	}
	var request struct {
		Name             string `json:"nome"`
		BuyIn            int64  `json:"buy_in"`
		MaxEntries       int    `json:"max_inscritos"`
		StartingStack    int64  `json:"starting_stack"`
		BlindIntervalMin int    `json:"blind_interval_min"`
		SmallBlind       int64  `json:"small_blind"`
		BigBlind         int64  `json:"big_blind"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		response.Error(w, http.StatusBadRequest, "Dados inválidos")
		return
	}
	request.Name = strings.TrimSpace(request.Name)
	const maxTournamentChips int64 = 1_000_000_000
	if request.Name == "" || len(request.Name) > 150 || request.BuyIn <= 0 ||
		request.BuyIn > maxTournamentChips ||
		request.MaxEntries < 2 || request.MaxEntries > 9 ||
		request.StartingStack <= 0 || request.StartingStack > maxTournamentChips ||
		request.BlindIntervalMin <= 0 || request.BlindIntervalMin > 1440 ||
		request.SmallBlind <= 0 || request.SmallBlind > maxTournamentChips ||
		request.BigBlind < request.SmallBlind || request.BigBlind > maxTournamentChips {
		response.Error(w, http.StatusBadRequest, "Configuração de torneio inválida")
		return
	}
	tournament := models.Tournament{
		Nome:          request.Name,
		BuyIn:         request.BuyIn,
		MaxInscritos:  request.MaxEntries,
		StartingStack: request.StartingStack,
		BlindInterval: request.BlindIntervalMin,
		SmallBlind:    request.SmallBlind,
		BigBlind:      request.BigBlind,
	}
	creatorName := claims.Nome
	if creatorName == "" {
		creatorName = claims.Username
	}
	created, err := h.store.CreateTournament(r.Context(), tournament, claims.UserID, creatorName)
	if err != nil {
		log.Printf("Falha ao criar torneio: %v", err)
		response.Error(w, http.StatusInternalServerError, "Falha ao criar torneio")
		return
	}
	response.JSON(w, http.StatusCreated, created)
}

func (h *TournamentHandler) Register(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(auth.UserContextKey).(*auth.Claims)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "Não autenticado")
		return
	}
	tournamentID, err := uuid.Parse(chi.URLParam(r, "tournamentID"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "Torneio inválido")
		return
	}
	if h.authHandler == nil {
		response.Error(w, http.StatusServiceUnavailable, "Serviço de usuários indisponível")
		return
	}
	user, err := h.authHandler.FindUserByIdentity(r.Context(), claims.Username, claims.Email)
	if err != nil {
		log.Printf("Falha ao carregar jogador da inscrição: %v", err)
		response.Error(w, http.StatusInternalServerError, "Falha ao carregar jogador")
		return
	}
	playerName := user.NomeCompleto
	if playerName == "" {
		playerName = user.Username
	}
	tournament, seat, err := h.store.RegisterTournament(r.Context(), tournamentID, user.ID, playerName)
	switch {
	case errors.Is(err, database.ErrTournamentNotFound):
		response.Error(w, http.StatusNotFound, "Torneio não encontrado")
	case errors.Is(err, database.ErrTournamentFull):
		response.Error(w, http.StatusConflict, "Torneio sem vagas")
	case errors.Is(err, database.ErrTournamentClosed):
		response.Error(w, http.StatusConflict, "Inscrições encerradas")
	case errors.Is(err, finance.ErrInsufficientFunds):
		response.Error(w, http.StatusBadRequest, "Saldo insuficiente para o buy-in")
	case err != nil:
		log.Printf("Falha ao inscrever jogador no torneio %s: %v", tournamentID, err)
		response.Error(w, http.StatusInternalServerError, "Falha ao realizar inscrição")
	default:
		response.JSON(w, http.StatusCreated, map[string]interface{}{
			"tournament":  tournament,
			"seat_number": seat,
			"table_id":    tournament.TableID,
		})
	}
}

func (h *TournamentHandler) Leave(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(auth.UserContextKey).(*auth.Claims)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "Não autenticado")
		return
	}
	tournamentID, err := uuid.Parse(chi.URLParam(r, "tournamentID"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "Torneio inválido")
		return
	}
	err = h.store.LeaveTournament(r.Context(), tournamentID, claims.UserID)
	switch {
	case errors.Is(err, database.ErrTournamentNotFound):
		response.Error(w, http.StatusNotFound, "Torneio não encontrado")
	case errors.Is(err, database.ErrTournamentClosed):
		response.Error(w, http.StatusConflict, "Não é possível sair de um torneio já iniciado")
	case errors.Is(err, database.ErrTournamentEntry):
		response.Error(w, http.StatusConflict, "Você não está inscrito nem é o criador deste torneio")
	case errors.Is(err, database.ErrTournamentSuccessor):
		response.Error(w, http.StatusConflict, "Outro jogador precisa se inscrever antes de transferir o controle do torneio")
	case err != nil:
		log.Printf("Falha ao sair do torneio %s: %v", tournamentID, err)
		response.Error(w, http.StatusInternalServerError, "Falha ao sair do torneio")
	default:
		response.JSON(w, http.StatusOK, map[string]string{"status": "left"})
	}
}

func (h *TournamentHandler) Start(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(auth.UserContextKey).(*auth.Claims)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "Não autenticado")
		return
	}
	tournamentID, err := uuid.Parse(chi.URLParam(r, "tournamentID"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "Torneio inválido")
		return
	}
	game, err := h.store.StartTournament(r.Context(), tournamentID, claims.UserID)
	switch {
	case errors.Is(err, database.ErrTournamentNotFound):
		response.Error(w, http.StatusNotFound, "Torneio não encontrado")
	case errors.Is(err, database.ErrTournamentForbidden):
		response.Error(w, http.StatusForbidden, "Somente o dono atual do torneio pode iniciá-lo")
	case errors.Is(err, database.ErrTournamentPlayers):
		response.Error(w, http.StatusConflict, "São necessários ao menos dois jogadores inscritos")
	case errors.Is(err, database.ErrTournamentStart), errors.Is(err, database.ErrTournamentClosed):
		response.Error(w, http.StatusConflict, err.Error())
	case err != nil:
		log.Printf("Falha ao iniciar torneio %s: %v", tournamentID, err)
		response.Error(w, http.StatusInternalServerError, "Falha ao iniciar torneio")
	default:
		h.hub.GameService().SetTable(game)
		h.hub.TableChanged(game.TableID, game)
		response.JSON(w, http.StatusOK, map[string]string{"status": "em_andamento"})
	}
}

func (h *TournamentHandler) Run(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			batchCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
			h.processRunningTournaments(batchCtx, now)
			cancel()
		}
	}
}

func (h *TournamentHandler) processRunningTournaments(ctx context.Context, now time.Time) {
	tournaments, err := h.store.ListRunningTournaments(ctx)
	if err != nil {
		log.Printf("Falha ao consultar torneios em andamento: %v", err)
		return
	}
	for _, tournament := range tournaments {
		if tournament.TableID == nil {
			continue
		}
		game, err := h.store.LoadGame(ctx, *tournament.TableID)
		if err != nil {
			log.Printf("Falha ao carregar partida do torneio %s: %v", tournament.ID, err)
			continue
		}
		h.hub.GameService().SetTable(game)
		state := game.GetPublicState()
		if state.Stage == engine.StageShowdown || state.Stage == engine.StageHandOver {
			var winnerID *uuid.UUID
			for _, player := range state.Players {
				if player.Stack > 0 && player.UserID != nil {
					if winnerID != nil {
						winnerID = nil
						break
					}
					id := *player.UserID
					winnerID = &id
				}
			}
			if winnerID != nil {
				if err := h.store.SettleTournament(ctx, tournament.ID, *winnerID); err != nil {
					log.Printf("Falha ao liquidar torneio %s: %v", tournament.ID, err)
				} else {
					continue
				}
			}
		}
		updatedGame, changed, err := h.store.RemoveEliminatedTournamentPlayers(ctx, tournament.ID)
		if err != nil {
			log.Printf("Falha ao registrar eliminações do torneio %s: %v", tournament.ID, err)
		} else if changed {
			game = updatedGame
			h.hub.GameService().SetTable(game)
		}
		blindedGame, blindChanged, err := h.store.AdvanceTournamentBlinds(ctx, tournament.ID, now)
		if err != nil {
			log.Printf("Falha ao atualizar blinds do torneio %s: %v", tournament.ID, err)
		} else if blindChanged {
			h.hub.GameService().SetTable(blindedGame)
		}
	}
}

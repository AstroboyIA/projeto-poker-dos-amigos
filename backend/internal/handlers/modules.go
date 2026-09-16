package handlers

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/walissonpaulo/poker-dos-amigos-backend/internal/models"
	"github.com/walissonpaulo/poker-dos-amigos-backend/pkg/response"
)

type ModulesHandler struct{}

func NewModulesHandler() *ModulesHandler {
	return &ModulesHandler{}
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
	tables := []models.PokerTable{
		{
			ID:         uuid.New(),
			Nome:       "Mesa VIP Ouro #01 (Texas Hold'em)",
			Tipo:       models.TableTypeCashGame,
			SmallBlind: 25,
			BigBlind:   50,
			BuyInMin:   1000,
			BuyInMax:   5000,
			MaxSeats:   9,
			Status:     models.TableStatusRunning,
			CurrentPot: 1450,
			CreatedAt:  time.Now(),
		},
		{
			ID:         uuid.New(),
			Nome:       "Mesa Bronze #02 (Micro Stakes)",
			Tipo:       models.TableTypeCashGame,
			SmallBlind: 5,
			BigBlind:   10,
			BuyInMin:   200,
			BuyInMax:   1000,
			MaxSeats:   6,
			Status:     models.TableStatusRunning,
			CurrentPot: 320,
			CreatedAt:  time.Now(),
		},
		{
			ID:         uuid.New(),
			Nome:       "Mesa Torneio Amigos Final Table",
			Tipo:       models.TableTypeTournament,
			SmallBlind: 500,
			BigBlind:   1000,
			BuyInMin:   0,
			BuyInMax:   0,
			MaxSeats:   9,
			Status:     models.TableStatusRunning,
			CurrentPot: 28000,
			CreatedAt:  time.Now(),
		},
	}
	response.JSON(w, http.StatusOK, tables)
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

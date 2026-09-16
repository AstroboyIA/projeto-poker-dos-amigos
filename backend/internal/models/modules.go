package models

import (
	"time"

	"github.com/google/uuid"
)

// Torneio
type Tournament struct {
	ID            uuid.UUID `json:"id"`
	Nome          string    `json:"nome"`
	BuyIn         int64     `json:"buy_in"`
	Garantido     int64     `json:"garantido"`
	Inscritos     int       `json:"inscritos"`
	MaxInscritos  int       `json:"max_inscritos"`
	DataInicio    time.Time `json:"data_inicio"`
	Status        string    `json:"status"` // "aberto", "em_andamento", "concluido"
	BlindInterval int       `json:"blind_interval_min"`
}

// Ranking
type RankingEntry struct {
	Posicao     int       `json:"posicao"`
	UserID      uuid.UUID `json:"user_id"`
	Nome        string    `json:"nome"`
	Pontos      int       `json:"pontos"`
	TorneiosGan int       `json:"torneios_ganhos"`
	LucroTotal  int64     `json:"lucro_total"`
}

// Financeiro
type TransactionType string

const (
	TxDeposito TransactionType = "deposito"
	TxSaque    TransactionType = "saque"
	TxBuyIn    TransactionType = "buyin"
	TxPremio   TransactionType = "premio"
	TxRake     TransactionType = "rake"
)

type FinancialTransaction struct {
	ID        uuid.UUID       `json:"id"`
	UserID    uuid.UUID       `json:"user_id"`
	Tipo      TransactionType `json:"tipo"`
	Valor     int64           `json:"valor"`
	Descricao string          `json:"descricao"`
	Status    string          `json:"status"` // "concluido", "pendente"
	CreatedAt time.Time       `json:"created_at"`
}

// Comunicados
type Announcement struct {
	ID        uuid.UUID `json:"id"`
	Titulo    string    `json:"titulo"`
	Conteudo  string    `json:"conteudo"`
	Autor     string    `json:"autor"`
	Publicado bool      `json:"publicado"`
	CreatedAt time.Time `json:"created_at"`
}

// Estatísticas
type UserStats struct {
	UserID         uuid.UUID `json:"user_id"`
	MaosJogadas    int       `json:"maos_jogadas"`
	MaosGanhas     int       `json:"maos_ganhas"`
	VPIP           float64   `json:"vpip_percent"` // Voluntarily Put In Pot %
	PFR            float64   `json:"pfr_percent"`  // Pre-Flop Raise %
	TotalGanhos    int64     `json:"total_ganhos"`
	TotalPerdas    int64     `json:"total_perdas"`
	MelhorMao      string    `json:"melhor_mao"`
}

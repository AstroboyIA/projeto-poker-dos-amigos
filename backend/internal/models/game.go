package models

import (
	"time"

	"github.com/google/uuid"
)

type TableType string

const (
	TableTypeCashGame   TableType = "cash_game"
	TableTypeTournament TableType = "torneio"
)

type TableStatus string

const (
	TableStatusWaiting  TableStatus = "aguardando"
	TableStatusRunning  TableStatus = "em_jogo"
	TableStatusFinished TableStatus = "finalizada"
)

type CardSuit string

const (
	SuitSpades   CardSuit = "S" // Espadas ♠
	SuitHearts   CardSuit = "H" // Copas ♥
	SuitDiamonds CardSuit = "D" // Ouros ♦
	SuitClubs    CardSuit = "C" // Paus ♣
)

type Card struct {
	Value int      `json:"value"` // 2-14 (14 = Ás)
	Suit  CardSuit `json:"suit"`  // S, H, D, C
	Code  string   `json:"code"`  // ex: "AS", "KH", "10D", "2C"
}

type PokerTable struct {
	ID             uuid.UUID   `json:"id"`
	Nome           string      `json:"nome"`
	Tipo           TableType   `json:"tipo"`
	SmallBlind     int64       `json:"small_blind"`
	BigBlind       int64       `json:"big_blind"`
	BuyInMin       int64       `json:"buy_in_min"`
	BuyInMax       int64       `json:"buy_in_max"`
	MaxSeats       int         `json:"max_seats"`
	Status         TableStatus `json:"status"`
	CurrentPot     int64       `json:"current_pot"`
	CommunityCards []Card      `json:"community_cards"`
	CurrentDealer  int         `json:"current_dealer"`
	CurrentTurn    int         `json:"current_turn"`
	BotSeats       []int       `json:"bot_seats"`
	OccupiedSeats  []int       `json:"occupied_seats"`
	Password       string      `json:"password,omitempty"`
	CreatedBy      string      `json:"created_by"`
	CreatedAt      time.Time   `json:"created_at"`
	UpdatedAt      time.Time   `json:"updated_at"`
}

type PlayerSeat struct {
	SeatIndex   int       `json:"seat_index"`
	UserID      uuid.UUID `json:"user_id"`
	Nome        string    `json:"nome"`
	Stack       int64     `json:"stack"`
	CurrentBet  int64     `json:"current_bet"`
	Cards       []Card    `json:"cards,omitempty"` // Oculto se não for o próprio jogador ou showdown
	HasFolded   bool      `json:"has_folded"`
	IsAllIn     bool      `json:"is_all_in"`
	IsConnected bool      `json:"is_connected"`
}

package models

import (
	"time"

	"github.com/google/uuid"
)

type GameHistoryEntry struct {
	ID                  uuid.UUID  `json:"id"`
	TableName           string     `json:"table_name"`
	GameType            string     `json:"game_type"`
	AmountInvestedChips int64      `json:"amount_invested_chips"`
	PayoutChips         int64      `json:"payout_chips"`
	Won                 *bool      `json:"won,omitempty"`
	StartedAt           time.Time  `json:"started_at"`
	FinishedAt          *time.Time `json:"finished_at,omitempty"`
}

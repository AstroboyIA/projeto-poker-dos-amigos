package ws

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/walissonpaulo/poker-dos-amigos-backend/internal/engine"
)

func TestTurnTimeoutAutomaticallyFoldsCurrentPlayer(t *testing.T) {
	gameService := engine.NewGameService()
	tableID := uuid.New()
	table := gameService.GetOrCreateTable(tableID, 25, 50)
	if err := table.JoinPlayer(uuid.New(), "Jogador 1", 1, 1000); err != nil {
		t.Fatalf("primeiro jogador não entrou: %v", err)
	}
	if err := table.JoinPlayer(uuid.New(), "Jogador 2", 2, 1000); err != nil {
		t.Fatalf("segundo jogador não entrou: %v", err)
	}
	if err := table.Start(); err != nil {
		t.Fatalf("mesa não iniciou: %v", err)
	}

	hub := NewHub(gameService)
	hub.turnTimeout = 10 * time.Millisecond
	hub.scheduleTurnTimeout(tableID, table)
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		state := table.GetPublicState()
		if state.Stage == engine.StageShowdown && state.WinnerMessage != "" {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("o jogador sem ação não recebeu fold automático: %+v", table.GetPublicState())
}

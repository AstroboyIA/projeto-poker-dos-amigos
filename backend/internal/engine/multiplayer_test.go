package engine_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/walissonpaulo/poker-dos-amigos-backend/internal/engine"
)

func TestGameServiceTwoPlayersSync(t *testing.T) {
	gs := engine.NewGameService()
	tableID := uuid.New()

	table := gs.GetOrCreateTable(tableID, 25, 50)

	userA := uuid.New()
	userB := uuid.New()

	// 1. Jogador A entra
	err := table.JoinPlayer(userA, "Jogador A", 1, 2000)
	if err != nil {
		t.Fatalf("Erro ao entrar Jogador A: %v", err)
	}

	state1 := table.GetPublicState()
	if len(state1.Players) != 1 {
		t.Fatalf("Esperava 1 jogador, obteve %d", len(state1.Players))
	}
	if state1.Stage != engine.StageWaiting {
		t.Fatalf("Esperava StageWaiting com 1 jogador, obteve %s", state1.Stage)
	}

	// 2. Jogador B entra -> Mesa deve iniciar partida automaticamente
	err = table.JoinPlayer(userB, "Jogador B", 2, 2000)
	if err != nil {
		t.Fatalf("Erro ao entrar Jogador B: %v", err)
	}

	state2 := table.GetPublicState()
	if len(state2.Players) != 2 {
		t.Fatalf("Esperava 2 jogadores, obteve %d", len(state2.Players))
	}
	if state2.Stage != engine.StagePreFlop {
		t.Fatalf("Esperava StagePreFlop com 2 jogadores, obteve %s", state2.Stage)
	}
	if state2.Pot != 75 {
		t.Fatalf("Esperava Pote inicial de 75 (SB 25 + BB 50), obteve %d", state2.Pot)
	}

	// 3. Cartas privadas não devem vazar no estado público
	for _, p := range state2.Players {
		if len(p.Cards) != 0 {
			t.Fatalf("Vazamento de cartas privadas no estado público para jogador %s!", p.Name)
		}
		if p.CardCount != 2 {
			t.Fatalf("Esperava contagem de 2 cartas para o jogador %s, obteve %d", p.Name, p.CardCount)
		}
	}

	// 4. Cartas privadas acessíveis apenas pelo respectivo usuário
	cardsA := table.GetPrivateCards(userA)
	cardsB := table.GetPrivateCards(userB)
	if len(cardsA) != 2 || len(cardsB) != 2 {
		t.Fatalf("Esperava 2 cartas privadas por jogador")
	}
	if cardsA[0].Code == cardsB[0].Code || cardsA[1].Code == cardsB[1].Code {
		t.Fatalf("Cartas idênticas sorteadas para jogadores diferentes!")
	}

	// 5. Teste de ação válida e fora de turno
	currentTurnPlayerID := state2.Players[state2.CurrentTurnIdx].UserID
	if currentTurnPlayerID == nil {
		t.Fatalf("UserID do turno atual não pode ser nulo")
	}

	var otherUser uuid.UUID
	if *currentTurnPlayerID == userA {
		otherUser = userB
	} else {
		otherUser = userA
	}

	// Jogador fora da vez tenta agir -> deve falhar
	err = table.ProcessAction(otherUser, engine.ActionCall, 0)
	if err == nil {
		t.Fatalf("Esperava erro ao agir fora de turno")
	}

	// Jogador da vez age (Call) -> deve suceder
	err = table.ProcessAction(*currentTurnPlayerID, engine.ActionCall, 0)
	if err != nil {
		t.Fatalf("Ação válida de Call falhou: %v", err)
	}

	state3 := table.GetPublicState()
	t.Logf("Estado após Call: Pote = %d, Turno = %d", state3.Pot, state3.CurrentTurnIdx)
}

func TestJoinPlayerRejectsOccupiedSeatWithoutChangingTable(t *testing.T) {
	table := engine.NewGameService().GetOrCreateTable(uuid.New(), 25, 50)
	firstUser := uuid.New()
	secondUser := uuid.New()

	if err := table.JoinPlayer(firstUser, "Primeiro", 1, 2000); err != nil {
		t.Fatalf("primeiro jogador não entrou: %v", err)
	}
	if err := table.JoinPlayer(secondUser, "Segundo", 1, 2000); err == nil {
		t.Fatal("esperava erro ao tentar ocupar assento já utilizado")
	}

	if state := table.GetPublicState(); len(state.Players) != 1 {
		t.Fatalf("a mesa foi alterada após entrada inválida: %d jogadores", len(state.Players))
	}
}

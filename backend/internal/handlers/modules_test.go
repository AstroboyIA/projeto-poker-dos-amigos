package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/walissonpaulo/poker-dos-amigos-backend/internal/auth"
	"github.com/walissonpaulo/poker-dos-amigos-backend/internal/engine"
	"github.com/walissonpaulo/poker-dos-amigos-backend/internal/models"
	"github.com/walissonpaulo/poker-dos-amigos-backend/internal/ws"
)

func TestOccupySeatValidatesPlayerSeat(t *testing.T) {
	handler := NewModulesHandler(nil, nil)
	tableID := handler.tables[0].ID.String()

	tests := []struct {
		name       string
		seatNumber int
		wantStatus int
	}{
		{name: "seat outside table", seatNumber: 10, wantStatus: http.StatusBadRequest},
		{name: "occupied seat", seatNumber: 1, wantStatus: http.StatusConflict},
		{name: "available seat 3", seatNumber: 3, wantStatus: http.StatusOK},
		{name: "available seat 8", seatNumber: 8, wantStatus: http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(
				http.MethodPost,
				"/api/tables/occupy",
				strings.NewReader(`{"table_id":"`+tableID+`","seat_number":`+strconv.Itoa(tt.seatNumber)+`}`),
			)
			rec := httptest.NewRecorder()

			handler.OccupySeat(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}

func TestReleaseSeatDeletesEmptyUserTable(t *testing.T) {
	handler := NewModulesHandler(nil, nil)
	tableID := uuid.New()
	handler.tables = append(handler.tables, models.PokerTable{
		ID:            tableID,
		MaxSeats:      9,
		OccupiedSeats: []int{1},
		CreatedBy:     "Jogador",
	})

	if !handler.releaseSeat(tableID.String(), 1) {
		t.Fatal("releaseSeat() = false, want true")
	}

	for _, table := range handler.tables {
		if table.ID == tableID {
			t.Fatal("empty user-created table was not deleted")
		}
	}
}

func TestBuyInJoinsPlayerToGameTable(t *testing.T) {
	authHandler := NewAuthHandler(auth.NewTokenManager("test-secret"))
	user := authHandler.users["jogador"]
	gameService := engine.NewGameService()
	hub := ws.NewHub(gameService)
	go hub.Run()
	handler := NewModulesHandler(hub, authHandler)
	tableID := uuid.New()
	handler.tables = append(handler.tables, models.PokerTable{
		ID:            tableID,
		SmallBlind:    25,
		BigBlind:      50,
		BuyInMin:      1000,
		BuyInMax:      5000,
		MaxSeats:      9,
		OccupiedSeats: []int{},
		CreatedBy:     "Gerente",
	})
	claims := &auth.Claims{
		UserID:   user.ID,
		Username: user.Username,
		Email:    user.Email,
		Nome:     user.NomeCompleto,
	}
	req := httptest.NewRequest(http.MethodPost, "/api/chips/buy-in", strings.NewReader(
		`{"amount":1500,"table_id":"`+tableID.String()+`","seat_number":3}`,
	))
	req = req.WithContext(context.WithValue(req.Context(), auth.UserContextKey, claims))
	req.Header.Set("Idempotency-Key", uuid.NewString())
	rec := httptest.NewRecorder()

	handler.BuyIn(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("buy-in status = %d, want %d: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	state, exists := gameService.GetTable(tableID)
	if !exists {
		t.Fatal("buy-in succeeded without creating the game table")
	}
	publicState := state.GetPublicState()
	if len(publicState.Players) != 1 ||
		publicState.Players[0].UserID == nil ||
		*publicState.Players[0].UserID != user.ID ||
		publicState.Players[0].SeatNumber != 3 ||
		publicState.Players[0].Stack != 1500 {
		t.Fatalf("player was not seated in the game after buy-in: %+v", publicState.Players)
	}
}

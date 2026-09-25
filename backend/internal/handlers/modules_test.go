package handlers

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/walissonpaulo/poker-dos-amigos-backend/internal/models"
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
		{name: "bot seat", seatNumber: 3, wantStatus: http.StatusConflict},
		{name: "occupied seat", seatNumber: 1, wantStatus: http.StatusConflict},
		{name: "available seat", seatNumber: 8, wantStatus: http.StatusOK},
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

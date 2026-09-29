package finance

import (
	"testing"

	"github.com/google/uuid"
)

func TestReturnFromTableSettlesOnlyThatTableReservation(t *testing.T) {
	service := NewService()
	userID := uuid.New()
	service.EnsureWallet(userID, 1000)
	tableOne := uuid.NewString()
	tableTwo := uuid.NewString()

	if _, err := service.BuyIn(userID, 300, tableOne, "table-one-buy-in"); err != nil {
		t.Fatalf("buy-in da primeira mesa falhou: %v", err)
	}
	if _, err := service.BuyIn(userID, 200, tableTwo, "table-two-buy-in"); err != nil {
		t.Fatalf("buy-in da segunda mesa falhou: %v", err)
	}
	wallet, err := service.ReturnFromTable(userID, 0, tableOne, "table-one-exit")
	if err != nil {
		t.Fatalf("saída sem fichas falhou: %v", err)
	}
	if wallet.AvailableCents != 500 || wallet.ReservedCents != 200 || wallet.BalanceCents != 700 {
		t.Fatalf("carteira após perder o stack: %+v", wallet)
	}

	wallet, err = service.ReturnFromTable(userID, 200, tableTwo, "table-two-exit")
	if err != nil {
		t.Fatalf("saída da segunda mesa falhou: %v", err)
	}
	if wallet.AvailableCents != 700 || wallet.ReservedCents != 0 || wallet.BalanceCents != 700 {
		t.Fatalf("carteira após encerrar ambas as mesas: %+v", wallet)
	}
}

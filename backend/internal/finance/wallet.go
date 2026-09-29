package finance

import (
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInsufficientFunds = errors.New("saldo insuficiente")
	ErrInvalidAmount     = errors.New("valor deve ser positivo")
)

type TransactionType string

const (
	Deposit     TransactionType = "DEPOSIT"
	BuyIn       TransactionType = "BUY_IN"
	TableReturn TransactionType = "TABLE_RETURN"
	Adjustment  TransactionType = "ADJUSTMENT"
)

type LedgerEntry struct {
	ID             uuid.UUID       `json:"id"`
	UserID         uuid.UUID       `json:"user_id"`
	Type           TransactionType `json:"type"`
	AmountCents    int64           `json:"amount_cents"`
	BalanceBefore  int64           `json:"balance_before"`
	BalanceAfter   int64           `json:"balance_after"`
	ReferenceType  string          `json:"reference_type,omitempty"`
	ReferenceID    string          `json:"reference_id,omitempty"`
	IdempotencyKey string          `json:"idempotency_key,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
}

type Wallet struct {
	ID             uuid.UUID `json:"id"`
	UserID         uuid.UUID `json:"user_id"`
	BalanceCents   int64     `json:"balance_cents"`
	AvailableCents int64     `json:"available_cents"`
	ReservedCents  int64     `json:"reserved_cents"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type Service struct {
	mu                sync.Mutex
	wallets           map[uuid.UUID]*Wallet
	ledger            map[uuid.UUID][]LedgerEntry
	processed         map[string]LedgerEntry
	tableReservations map[uuid.UUID]map[string]int64
}

func NewService() *Service {
	return &Service{
		wallets:           make(map[uuid.UUID]*Wallet),
		ledger:            make(map[uuid.UUID][]LedgerEntry),
		processed:         make(map[string]LedgerEntry),
		tableReservations: make(map[uuid.UUID]map[string]int64),
	}
}

func (s *Service) EnsureWallet(userID uuid.UUID, initialCents int64) Wallet {
	s.mu.Lock()
	defer s.mu.Unlock()
	if wallet, ok := s.wallets[userID]; ok {
		return *wallet
	}
	now := time.Now()
	wallet := &Wallet{ID: uuid.New(), UserID: userID, BalanceCents: initialCents, AvailableCents: initialCents, UpdatedAt: now}
	s.wallets[userID] = wallet
	return *wallet
}

func (s *Service) Get(userID uuid.UUID) (Wallet, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	wallet, ok := s.wallets[userID]
	if !ok {
		return Wallet{}, false
	}
	return *wallet, true
}

func (s *Service) Entries(userID uuid.UUID) []LedgerEntry {
	s.mu.Lock()
	defer s.mu.Unlock()
	entries := append([]LedgerEntry(nil), s.ledger[userID]...)
	return entries
}

func processedKey(userID uuid.UUID, key string) string {
	return userID.String() + ":" + key
}

func (s *Service) applyLocked(userID uuid.UUID, amount int64, kind TransactionType, referenceType, referenceID, key string) (Wallet, error) {
	if amount <= 0 {
		return Wallet{}, ErrInvalidAmount
	}
	if key != "" {
		if _, ok := s.processed[processedKey(userID, key)]; ok {
			return *s.wallets[userID], nil
		}
	}
	wallet, ok := s.wallets[userID]
	if !ok {
		wallet = &Wallet{ID: uuid.New(), UserID: userID}
		s.wallets[userID] = wallet
	}
	before := wallet.BalanceCents
	if kind == BuyIn {
		if wallet.AvailableCents < amount {
			return Wallet{}, ErrInsufficientFunds
		}
		wallet.AvailableCents -= amount
		wallet.ReservedCents += amount
		if referenceType == "table" && referenceID != "" {
			if s.tableReservations[userID] == nil {
				s.tableReservations[userID] = make(map[string]int64)
			}
			s.tableReservations[userID][referenceID] += amount
		}
	} else {
		wallet.AvailableCents += amount
	}
	wallet.BalanceCents = wallet.AvailableCents + wallet.ReservedCents
	wallet.UpdatedAt = time.Now()
	entry := LedgerEntry{ID: uuid.New(), UserID: userID, Type: kind, AmountCents: amount, BalanceBefore: before, BalanceAfter: wallet.BalanceCents, ReferenceType: referenceType, ReferenceID: referenceID, IdempotencyKey: key, CreatedAt: time.Now()}
	s.ledger[userID] = append(s.ledger[userID], entry)
	if key != "" {
		s.processed[processedKey(userID, key)] = entry
	}
	return *wallet, nil
}

func (s *Service) Deposit(userID uuid.UUID, cents int64, key string) (Wallet, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.applyLocked(userID, cents, Deposit, "dev_mock", "", key)
}

func (s *Service) BuyIn(userID uuid.UUID, cents int64, tableID, key string) (Wallet, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.applyLocked(userID, cents, BuyIn, "table", tableID, key)
}

func (s *Service) ReturnFromTable(userID uuid.UUID, cents int64, tableID, key string) (Wallet, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	wallet, ok := s.wallets[userID]
	if !ok {
		return Wallet{}, errors.New("carteira não encontrada")
	}
	if cents < 0 {
		return Wallet{}, ErrInvalidAmount
	}
	if key != "" {
		if _, exists := s.processed[processedKey(userID, key)]; exists {
			return *wallet, nil
		}
	}
	reservedForTable := s.tableReservations[userID][tableID]
	if reservedForTable > wallet.ReservedCents {
		reservedForTable = wallet.ReservedCents
	}
	before := wallet.BalanceCents
	wallet.ReservedCents -= reservedForTable
	wallet.AvailableCents += cents
	wallet.BalanceCents = wallet.AvailableCents + wallet.ReservedCents
	wallet.UpdatedAt = time.Now()
	entry := LedgerEntry{ID: uuid.New(), UserID: userID, Type: TableReturn, AmountCents: cents, BalanceBefore: before, BalanceAfter: wallet.BalanceCents, ReferenceType: "table", ReferenceID: tableID, IdempotencyKey: key, CreatedAt: time.Now()}
	s.ledger[userID] = append(s.ledger[userID], entry)
	delete(s.tableReservations[userID], tableID)
	if key != "" {
		s.processed[processedKey(userID, key)] = entry
	}
	return *wallet, nil
}

func (s *Service) RefundBuyIn(userID uuid.UUID, cents int64, tableID, key string) (Wallet, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if cents <= 0 {
		return Wallet{}, ErrInvalidAmount
	}
	wallet, ok := s.wallets[userID]
	if !ok {
		return Wallet{}, errors.New("carteira não encontrada")
	}
	if key != "" {
		if _, exists := s.processed[processedKey(userID, key)]; exists {
			return *wallet, nil
		}
	}
	reserved := s.tableReservations[userID][tableID]
	if reserved < cents || wallet.ReservedCents < cents {
		return Wallet{}, errors.New("buy-in reservado não encontrado para estorno")
	}
	before := wallet.BalanceCents
	s.tableReservations[userID][tableID] -= cents
	if s.tableReservations[userID][tableID] == 0 {
		delete(s.tableReservations[userID], tableID)
	}
	wallet.ReservedCents -= cents
	wallet.AvailableCents += cents
	wallet.BalanceCents = wallet.AvailableCents + wallet.ReservedCents
	wallet.UpdatedAt = time.Now()
	entry := LedgerEntry{ID: uuid.New(), UserID: userID, Type: TableReturn, AmountCents: cents, BalanceBefore: before, BalanceAfter: wallet.BalanceCents, ReferenceType: "table", ReferenceID: tableID, IdempotencyKey: key, CreatedAt: time.Now()}
	s.ledger[userID] = append(s.ledger[userID], entry)
	if key != "" {
		s.processed[processedKey(userID, key)] = entry
	}
	return *wallet, nil
}

func (s *Service) GetUnlocked(userID uuid.UUID) (Wallet, error) {
	wallet, ok := s.wallets[userID]
	if !ok {
		return Wallet{}, errors.New("carteira não encontrada")
	}
	return *wallet, nil
}

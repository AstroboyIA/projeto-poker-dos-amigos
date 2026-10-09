package database

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/walissonpaulo/poker-dos-amigos-backend/internal/engine"
	"github.com/walissonpaulo/poker-dos-amigos-backend/internal/finance"
	"github.com/walissonpaulo/poker-dos-amigos-backend/internal/models"
)

const notificationChannel = "poker_table_updates"

var (
	ErrTableNotFound = errors.New("table not found")
	ErrSeatOccupied  = errors.New("seat is already occupied")
	ErrInvalidSeat   = errors.New("invalid seat number")
	ErrUserNotFound  = errors.New("user not found")
	ErrUsernameTaken = errors.New("username is already registered")
	ErrEmailTaken    = errors.New("email is already registered")
)

//go:embed migrations/004_shared_table_state.sql
var sharedTableMigration string

//go:embed migrations/005_user_email_case_insensitive.sql
var userIdentityMigration string

//go:embed migrations/006_persistent_wallets.sql
var persistentWalletMigration string

//go:embed migrations/007_single_table_tournaments.sql
var singleTableTournamentMigration string

//go:embed migrations/008_player_game_history.sql
var playerGameHistoryMigration string

type Store struct {
	pool *pgxpool.Pool
	dsn  string
}

func Open(ctx context.Context, dsn string) (*Store, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("parse PostgreSQL URL: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("connect to PostgreSQL: %w", err)
	}
	return &Store{pool: pool, dsn: dsn}, nil
}

func (s *Store) Close() {
	s.pool.Close()
}

func (s *Store) ApplySharedTableMigration(ctx context.Context) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin database migrations: %w", err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('poker-schema-migrations', 0))`); err != nil {
		return fmt.Errorf("lock database migrations: %w", err)
	}
	if _, err := tx.Exec(ctx, sharedTableMigration); err != nil {
		return fmt.Errorf("apply shared table migration: %w", err)
	}
	if _, err := tx.Exec(ctx, userIdentityMigration); err != nil {
		return fmt.Errorf("apply user identity migration: %w", err)
	}
	if _, err := tx.Exec(ctx, persistentWalletMigration); err != nil {
		return fmt.Errorf("apply persistent wallet migration: %w", err)
	}
	if _, err := tx.Exec(ctx, singleTableTournamentMigration); err != nil {
		return fmt.Errorf("apply single-table tournament migration: %w", err)
	}
	if _, err := tx.Exec(ctx, playerGameHistoryMigration); err != nil {
		return fmt.Errorf("apply player game history migration: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit database migrations: %w", err)
	}
	return nil
}

func (s *Store) EnsureWallet(ctx context.Context, userID uuid.UUID, initialCents int64) (finance.Wallet, error) {
	if initialCents < 0 {
		return finance.Wallet{}, finance.ErrInvalidAmount
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return finance.Wallet{}, fmt.Errorf("begin wallet initialization: %w", err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `
		INSERT INTO wallets (user_id, balance_cents, available_cents)
		VALUES ($1, $2, $2)
		ON CONFLICT (user_id) DO NOTHING
	`, userID, initialCents); err != nil {
		return finance.Wallet{}, fmt.Errorf("create wallet: %w", err)
	}
	wallet, err := scanWallet(tx.QueryRow(ctx, `
		SELECT id, user_id, balance_cents, available_cents, reserved_cents, updated_at
		FROM wallets WHERE user_id = $1 FOR UPDATE
	`, userID))
	if err != nil {
		return finance.Wallet{}, fmt.Errorf("load initialized wallet: %w", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE users SET wallet_id = $1, saldo_fichas = $2 WHERE id = $3`, wallet.ID, wallet.AvailableCents, userID); err != nil {
		return finance.Wallet{}, fmt.Errorf("link wallet to user: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return finance.Wallet{}, fmt.Errorf("commit wallet initialization: %w", err)
	}
	return wallet, nil
}

func (s *Store) GetWallet(ctx context.Context, userID uuid.UUID) (finance.Wallet, bool, error) {
	wallet, err := scanWallet(s.pool.QueryRow(ctx, `
		SELECT id, user_id, balance_cents, available_cents, reserved_cents, updated_at
		FROM wallets WHERE user_id = $1
	`, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return finance.Wallet{}, false, nil
	}
	if err != nil {
		return finance.Wallet{}, false, fmt.Errorf("query wallet: %w", err)
	}
	return wallet, true, nil
}

func (s *Store) ListWalletEntries(ctx context.Context, userID uuid.UUID) ([]finance.LedgerEntry, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, user_id, type, amount_cents, balance_before, balance_after,
			COALESCE(reference_type, ''), COALESCE(reference_id, ''),
			COALESCE(idempotency_key, ''), created_at
		FROM wallet_ledger WHERE user_id = $1 ORDER BY created_at DESC, id DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("query wallet ledger: %w", err)
	}
	defer rows.Close()
	entries := make([]finance.LedgerEntry, 0)
	for rows.Next() {
		var entry finance.LedgerEntry
		if err := rows.Scan(
			&entry.ID, &entry.UserID, &entry.Type, &entry.AmountCents,
			&entry.BalanceBefore, &entry.BalanceAfter, &entry.ReferenceType,
			&entry.ReferenceID, &entry.IdempotencyKey, &entry.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan wallet ledger entry: %w", err)
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate wallet ledger: %w", err)
	}
	return entries, nil
}

func (s *Store) Deposit(ctx context.Context, userID uuid.UUID, cents int64, key string) (finance.Wallet, error) {
	if cents <= 0 {
		return finance.Wallet{}, finance.ErrInvalidAmount
	}
	return s.applyWalletTransaction(ctx, userID, cents, finance.Deposit, "dev_mock", "", key)
}

func (s *Store) BuyIn(ctx context.Context, userID uuid.UUID, cents int64, tableID, key string) (finance.Wallet, error) {
	if cents <= 0 {
		return finance.Wallet{}, finance.ErrInvalidAmount
	}
	return s.applyWalletTransaction(ctx, userID, cents, finance.BuyIn, "table", tableID, key)
}

func (s *Store) ReturnFromTable(ctx context.Context, userID uuid.UUID, cents int64, tableID, key string) (finance.Wallet, error) {
	if cents < 0 {
		return finance.Wallet{}, finance.ErrInvalidAmount
	}
	return s.applyWalletTransaction(ctx, userID, cents, finance.TableReturn, "table", tableID, key)
}

func (s *Store) RefundBuyIn(ctx context.Context, userID uuid.UUID, cents int64, tableID, key string) (finance.Wallet, error) {
	if cents <= 0 {
		return finance.Wallet{}, finance.ErrInvalidAmount
	}
	return s.applyWalletTransaction(ctx, userID, cents, finance.Adjustment, "table", tableID, key)
}

func (s *Store) applyWalletTransaction(ctx context.Context, userID uuid.UUID, cents int64, kind finance.TransactionType, referenceType, referenceID, key string) (finance.Wallet, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return finance.Wallet{}, fmt.Errorf("begin wallet transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `INSERT INTO wallets (user_id) VALUES ($1) ON CONFLICT (user_id) DO NOTHING`, userID); err != nil {
		return finance.Wallet{}, fmt.Errorf("ensure wallet exists: %w", err)
	}
	wallet, err := scanWallet(tx.QueryRow(ctx, `
		SELECT id, user_id, balance_cents, available_cents, reserved_cents, updated_at
		FROM wallets WHERE user_id = $1 FOR UPDATE
	`, userID))
	if err != nil {
		return finance.Wallet{}, fmt.Errorf("lock wallet: %w", err)
	}

	storedKey := walletIdempotencyKey(userID, key)
	if len(storedKey) > 255 {
		return finance.Wallet{}, errors.New("chave de idempotência excede 255 bytes")
	}
	if storedKey != "" {
		var exists bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM wallet_ledger WHERE idempotency_key = $1)`, storedKey).Scan(&exists); err != nil {
			return finance.Wallet{}, fmt.Errorf("check wallet idempotency key: %w", err)
		}
		if exists {
			if err := tx.Commit(ctx); err != nil {
				return finance.Wallet{}, fmt.Errorf("commit idempotent wallet request: %w", err)
			}
			return wallet, nil
		}
	}

	before := wallet.BalanceCents
	switch kind {
	case finance.BuyIn:
		if wallet.AvailableCents < cents {
			return finance.Wallet{}, finance.ErrInsufficientFunds
		}
		wallet.AvailableCents -= cents
		wallet.ReservedCents += cents
		if _, err := tx.Exec(ctx, `
			INSERT INTO wallet_table_reservations (user_id, table_id, amount_cents)
			VALUES ($1, $2, $3)
			ON CONFLICT (user_id, table_id) DO UPDATE
			SET amount_cents = wallet_table_reservations.amount_cents + EXCLUDED.amount_cents,
				updated_at = CURRENT_TIMESTAMP
		`, userID, referenceID, cents); err != nil {
			return finance.Wallet{}, fmt.Errorf("reserve table buy-in: %w", err)
		}
	case finance.TableReturn:
		var reservedForTable int64
		err := tx.QueryRow(ctx, `
			SELECT amount_cents FROM wallet_table_reservations
			WHERE user_id = $1 AND table_id = $2 FOR UPDATE
		`, userID, referenceID).Scan(&reservedForTable)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return finance.Wallet{}, fmt.Errorf("load table reservation: %w", err)
		}
		if reservedForTable > wallet.ReservedCents {
			reservedForTable = wallet.ReservedCents
		}
		wallet.ReservedCents -= reservedForTable
		wallet.AvailableCents += cents
		if _, err := tx.Exec(ctx, `DELETE FROM wallet_table_reservations WHERE user_id = $1 AND table_id = $2`, userID, referenceID); err != nil {
			return finance.Wallet{}, fmt.Errorf("settle table reservation: %w", err)
		}
	case finance.Deposit:
		wallet.AvailableCents += cents
	case finance.Adjustment:
		var reservedForTable int64
		err := tx.QueryRow(ctx, `
			SELECT amount_cents FROM wallet_table_reservations
			WHERE user_id = $1 AND table_id = $2 FOR UPDATE
		`, userID, referenceID).Scan(&reservedForTable)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return finance.Wallet{}, errors.New("buy-in reservado não encontrado para estorno")
			}
			return finance.Wallet{}, fmt.Errorf("load table reservation for refund: %w", err)
		}
		if reservedForTable < cents || wallet.ReservedCents < cents {
			return finance.Wallet{}, errors.New("buy-in reservado não encontrado para estorno")
		}
		if reservedForTable == cents {
			_, err = tx.Exec(ctx, `DELETE FROM wallet_table_reservations WHERE user_id = $1 AND table_id = $2`, userID, referenceID)
		} else {
			_, err = tx.Exec(ctx, `
				UPDATE wallet_table_reservations SET amount_cents = amount_cents - $3, updated_at = CURRENT_TIMESTAMP
				WHERE user_id = $1 AND table_id = $2
			`, userID, referenceID, cents)
		}
		if err != nil {
			return finance.Wallet{}, fmt.Errorf("release refunded reservation: %w", err)
		}
		wallet.ReservedCents -= cents
		wallet.AvailableCents += cents
	default:
		return finance.Wallet{}, fmt.Errorf("unsupported wallet transaction type %q", kind)
	}
	wallet.BalanceCents = wallet.AvailableCents + wallet.ReservedCents
	now := time.Now()
	wallet.UpdatedAt = now
	if _, err := tx.Exec(ctx, `
		UPDATE wallets
		SET balance_cents = $1, available_cents = $2, reserved_cents = $3, updated_at = $4
		WHERE id = $5
	`, wallet.BalanceCents, wallet.AvailableCents, wallet.ReservedCents, now, wallet.ID); err != nil {
		return finance.Wallet{}, fmt.Errorf("update wallet balance: %w", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE users SET saldo_fichas = $1 WHERE id = $2`, wallet.AvailableCents, userID); err != nil {
		return finance.Wallet{}, fmt.Errorf("sync legacy user balance: %w", err)
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO wallet_ledger (
			user_id, wallet_id, type, amount_cents, balance_before, balance_after,
			reference_type, reference_id, idempotency_key, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, NULLIF($7, ''), NULLIF($8, ''), NULLIF($9, ''), $10)
	`, userID, wallet.ID, kind, cents, before, wallet.BalanceCents, referenceType, referenceID, storedKey, now)
	if err != nil {
		return finance.Wallet{}, fmt.Errorf("write wallet ledger entry: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return finance.Wallet{}, fmt.Errorf("commit wallet transaction: %w", err)
	}
	return wallet, nil
}

func walletIdempotencyKey(userID uuid.UUID, key string) string {
	if key == "" {
		return ""
	}
	return userID.String() + ":" + key
}

func scanWallet(row pgx.Row) (finance.Wallet, error) {
	var wallet finance.Wallet
	err := row.Scan(&wallet.ID, &wallet.UserID, &wallet.BalanceCents, &wallet.AvailableCents, &wallet.ReservedCents, &wallet.UpdatedAt)
	return wallet, err
}

func (s *Store) CreateUser(ctx context.Context, user *models.User) error {
	identities := make([]string, 0, 2)
	if user.Username != "" {
		identities = append(identities, strings.ToLower(strings.TrimSpace(user.Username)))
	}
	if user.Email != "" {
		identities = append(identities, strings.ToLower(strings.TrimSpace(user.Email)))
	}
	sort.Strings(identities)
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin user creation: %w", err)
	}
	defer tx.Rollback(ctx)
	for _, identity := range identities {
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, identity); err != nil {
			return fmt.Errorf("lock user identity: %w", err)
		}
	}
	for _, identity := range identities {
		var existingUsername, existingEmail bool
		if err := tx.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM users
				WHERE LOWER(COALESCE(username, '')) = $1
			), EXISTS (
				SELECT 1 FROM users
				WHERE LOWER(COALESCE(email, '')) = $1
			)
		`, identity).Scan(&existingUsername, &existingEmail); err != nil {
			return fmt.Errorf("check existing user identity: %w", err)
		}
		if existingUsername {
			return ErrUsernameTaken
		}
		if existingEmail {
			return ErrEmailTaken
		}
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO users (
			id, username, nome_completo, telefone, email, password_hash,
			data_nascimento, cidade_estado, aceitou_termos, role, status,
			saldo_fichas, created_at, updated_at
		) VALUES (
			$1, NULLIF($2, ''), NULLIF($3, ''), NULLIF($4, ''), NULLIF($5, ''),
			$6, NULLIF($7, ''), NULLIF($8, ''), $9, $10, $11, $12, $13, $14
		)
	`, user.ID, user.Username, user.NomeCompleto, user.Telefone, user.Email,
		user.PasswordHash, user.DataNasc, user.CidadeEstado, user.AceitouTermo,
		user.Role, user.Status, user.SaldoFichas, user.CreatedAt, user.UpdatedAt)
	if err == nil {
		var walletID uuid.UUID
		if err := tx.QueryRow(ctx, `
			INSERT INTO wallets (user_id, balance_cents, available_cents, reserved_cents)
			VALUES ($1, $2, $2, 0)
			RETURNING id
		`, user.ID, user.SaldoFichas).Scan(&walletID); err != nil {
			return fmt.Errorf("create user wallet: %w", err)
		}
		if _, err := tx.Exec(ctx, `UPDATE users SET wallet_id = $1 WHERE id = $2`, walletID, user.ID); err != nil {
			return fmt.Errorf("link wallet to user: %w", err)
		}
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("commit user creation: %w", err)
		}
		user.Wallet = &models.WalletSummary{
			BalanceCents:   user.SaldoFichas,
			AvailableCents: user.SaldoFichas,
		}
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		switch pgErr.ConstraintName {
		case "users_username_lower_idx", "users_username_key":
			return ErrUsernameTaken
		case "users_email_key", "users_email_lower_idx":
			return ErrEmailTaken
		}
	}
	return fmt.Errorf("insert user: %w", err)
}

func (s *Store) GetUserByIdentity(ctx context.Context, username, email string) (*models.User, error) {
	user := new(models.User)
	err := scanUser(s.pool.QueryRow(ctx, `
		SELECT id, COALESCE(username, ''), COALESCE(nome_completo, ''),
			COALESCE(telefone, ''), COALESCE(email, ''), password_hash,
			COALESCE(data_nascimento, ''), COALESCE(cidade_estado, ''),
			COALESCE(aceitou_termos, FALSE), COALESCE(role, 'jogador'),
			COALESCE(status, 'ativo'), COALESCE(saldo_fichas, 0), created_at, updated_at
		FROM users
		WHERE ($1 <> '' AND LOWER(COALESCE(username, '')) = LOWER($1))
		   OR ($2 <> '' AND LOWER(COALESCE(email, '')) = LOWER($2))
		ORDER BY CASE WHEN $1 <> '' AND LOWER(COALESCE(username, '')) = LOWER($1) THEN 0 ELSE 1 END
		LIMIT 1
	`, strings.TrimSpace(username), strings.TrimSpace(email)), user)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("query user by identity: %w", err)
	}
	return user, nil
}

func (s *Store) GetUserByID(ctx context.Context, userID uuid.UUID) (*models.User, error) {
	user := new(models.User)
	err := scanUser(s.pool.QueryRow(ctx, `
		SELECT id, COALESCE(username, ''), COALESCE(nome_completo, ''),
			COALESCE(telefone, ''), COALESCE(email, ''), password_hash,
			COALESCE(data_nascimento, ''), COALESCE(cidade_estado, ''),
			COALESCE(aceitou_termos, FALSE), COALESCE(role, 'jogador'),
			COALESCE(status, 'ativo'), COALESCE(saldo_fichas, 0), created_at, updated_at
		FROM users WHERE id = $1
	`, userID), user)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("query user by ID: %w", err)
	}
	return user, nil
}

func scanUser(row pgx.Row, user *models.User) error {
	return row.Scan(
		&user.ID, &user.Username, &user.NomeCompleto, &user.Telefone, &user.Email,
		&user.PasswordHash, &user.DataNasc, &user.CidadeEstado, &user.AceitouTermo,
		&user.Role, &user.Status, &user.SaldoFichas, &user.CreatedAt, &user.UpdatedAt,
	)
}

func (s *Store) ListTables(ctx context.Context) ([]models.PokerTable, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, nome, tipo, small_blind, big_blind, buy_in_min, buy_in_max,
			max_seats, status, current_pot, community_cards, current_dealer,
			current_turn, occupied_seats, password, created_by, creator_user_id,
			created_at, updated_at
		FROM poker_tables
		ORDER BY created_at DESC, id
	`)
	if err != nil {
		return nil, fmt.Errorf("query poker tables: %w", err)
	}
	defer rows.Close()

	tables := make([]models.PokerTable, 0)
	for rows.Next() {
		var table models.PokerTable
		var occupiedJSON, communityJSON []byte
		var creatorID *uuid.UUID
		if err := rows.Scan(
			&table.ID, &table.Nome, &table.Tipo, &table.SmallBlind, &table.BigBlind,
			&table.BuyInMin, &table.BuyInMax, &table.MaxSeats, &table.Status,
			&table.CurrentPot, &communityJSON, &table.CurrentDealer, &table.CurrentTurn,
			&occupiedJSON, &table.Password, &table.CreatedBy, &creatorID,
			&table.CreatedAt, &table.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan poker table: %w", err)
		}
		if err := json.Unmarshal(occupiedJSON, &table.OccupiedSeats); err != nil {
			return nil, fmt.Errorf("decode occupied seats for table %s: %w", table.ID, err)
		}
		if len(communityJSON) > 0 && string(communityJSON) != "null" {
			if err := json.Unmarshal(communityJSON, &table.CommunityCards); err != nil {
				return nil, fmt.Errorf("decode community cards for table %s: %w", table.ID, err)
			}
		}
		if creatorID != nil {
			table.CreatorUserID = *creatorID
		}
		tables = append(tables, table)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate poker tables: %w", err)
	}
	return tables, nil
}

func (s *Store) GetTable(ctx context.Context, tableID uuid.UUID) (models.PokerTable, error) {
	var table models.PokerTable
	var occupiedJSON, communityJSON []byte
	var creatorID *uuid.UUID
	err := s.pool.QueryRow(ctx, `
		SELECT id, nome, tipo, small_blind, big_blind, buy_in_min, buy_in_max,
			max_seats, status, current_pot, community_cards, current_dealer,
			current_turn, occupied_seats, password, created_by, creator_user_id,
			created_at, updated_at
		FROM poker_tables WHERE id = $1
	`, tableID).Scan(
		&table.ID, &table.Nome, &table.Tipo, &table.SmallBlind, &table.BigBlind,
		&table.BuyInMin, &table.BuyInMax, &table.MaxSeats, &table.Status,
		&table.CurrentPot, &communityJSON, &table.CurrentDealer, &table.CurrentTurn,
		&occupiedJSON, &table.Password, &table.CreatedBy, &creatorID,
		&table.CreatedAt, &table.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.PokerTable{}, ErrTableNotFound
		}
		return models.PokerTable{}, err
	}
	if err := json.Unmarshal(occupiedJSON, &table.OccupiedSeats); err != nil {
		return models.PokerTable{}, fmt.Errorf("decode occupied seats for table %s: %w", table.ID, err)
	}
	if len(communityJSON) > 0 && string(communityJSON) != "null" {
		if err := json.Unmarshal(communityJSON, &table.CommunityCards); err != nil {
			return models.PokerTable{}, fmt.Errorf("decode community cards for table %s: %w", table.ID, err)
		}
	}
	if creatorID != nil {
		table.CreatorUserID = *creatorID
	}
	return table, nil
}

func (s *Store) CreateTable(ctx context.Context, table models.PokerTable) error {
	occupied, err := json.Marshal(table.OccupiedSeats)
	if err != nil {
		return fmt.Errorf("encode occupied seats: %w", err)
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO poker_tables (
			id, nome, tipo, small_blind, big_blind, buy_in_min, buy_in_max,
			max_seats, status, occupied_seats, password, created_by,
			creator_user_id, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10::jsonb, $11, $12, $13, $14, $15)
	`, table.ID, table.Nome, table.Tipo, table.SmallBlind, table.BigBlind,
		table.BuyInMin, table.BuyInMax, table.MaxSeats, table.Status, occupied,
		table.Password, table.CreatedBy, table.CreatorUserID, table.CreatedAt, table.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert poker table: %w", err)
	}
	return s.Notify(ctx, "tables")
}

func (s *Store) ReserveSeat(ctx context.Context, tableID uuid.UUID, seatNumber int) (models.PokerTable, error) {
	tx, err := s.beginTableTransaction(ctx, tableID)
	if err != nil {
		return models.PokerTable{}, err
	}
	defer tx.Rollback(ctx)

	table, err := getTableTx(ctx, tx, tableID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.PokerTable{}, ErrTableNotFound
		}
		return models.PokerTable{}, err
	}
	if seatNumber < 1 || seatNumber > table.MaxSeats {
		return models.PokerTable{}, fmt.Errorf("%w: must be between 1 and %d", ErrInvalidSeat, table.MaxSeats)
	}
	for _, occupied := range table.OccupiedSeats {
		if occupied == seatNumber {
			return models.PokerTable{}, ErrSeatOccupied
		}
	}
	table.OccupiedSeats = append(table.OccupiedSeats, seatNumber)
	table.UpdatedAt = time.Now()
	if err := updateSeatsTx(ctx, tx, table); err != nil {
		return models.PokerTable{}, err
	}
	if err := notifyTx(ctx, tx, "tables"); err != nil {
		return models.PokerTable{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return models.PokerTable{}, fmt.Errorf("commit seat reservation: %w", err)
	}
	return table, nil
}

func (s *Store) ReleaseSeat(ctx context.Context, tableID uuid.UUID, seatNumber int) (bool, error) {
	tx, err := s.beginTableTransaction(ctx, tableID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	defer tx.Rollback(ctx)

	table, err := getTableTx(ctx, tx, tableID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	seats := make([]int, 0, len(table.OccupiedSeats))
	for _, occupied := range table.OccupiedSeats {
		if occupied != seatNumber {
			seats = append(seats, occupied)
		}
	}
	changed := len(seats) != len(table.OccupiedSeats)
	if changed {
		table.OccupiedSeats = seats
		table.UpdatedAt = time.Now()
		if len(seats) == 0 && table.CreatedBy != "Clube" && table.CreatedBy != "Diretoria" {
			if _, err := tx.Exec(ctx, `DELETE FROM poker_tables WHERE id = $1`, tableID); err != nil {
				return false, fmt.Errorf("delete empty poker table: %w", err)
			}
		} else if err := updateSeatsTx(ctx, tx, table); err != nil {
			return false, err
		}
		if err := notifyTx(ctx, tx, "tables"); err != nil {
			return false, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("commit seat release: %w", err)
	}
	return true, nil
}

func (s *Store) JoinPlayer(
	ctx context.Context,
	tableID uuid.UUID,
	userID uuid.UUID,
	name string,
	seatNumber int,
	buyIn int64,
	idempotencyKey string,
) (*engine.TableGame, models.PokerTable, error) {
	tx, err := s.beginTableTransaction(ctx, tableID)
	if err != nil {
		return nil, models.PokerTable{}, err
	}
	defer tx.Rollback(ctx)

	table, err := getTableTx(ctx, tx, tableID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, models.PokerTable{}, ErrTableNotFound
		}
		return nil, models.PokerTable{}, err
	}
	if seatNumber < 1 || seatNumber > table.MaxSeats {
		return nil, models.PokerTable{}, fmt.Errorf("%w: must be between 1 and %d", ErrInvalidSeat, table.MaxSeats)
	}
	for _, occupied := range table.OccupiedSeats {
		if occupied == seatNumber {
			return nil, models.PokerTable{}, ErrSeatOccupied
		}
	}

	game, err := loadGameTx(ctx, tx, tableID, table.SmallBlind, table.BigBlind)
	if err != nil {
		return nil, models.PokerTable{}, err
	}
	if err := game.JoinPlayer(userID, name, seatNumber, buyIn); err != nil {
		return nil, models.PokerTable{}, err
	}
	table.OccupiedSeats = append(table.OccupiedSeats, seatNumber)
	table.UpdatedAt = time.Now()
	if err := updateSeatsTx(ctx, tx, table); err != nil {
		return nil, models.PokerTable{}, err
	}
	if err := saveGameTx(ctx, tx, game); err != nil {
		return nil, models.PokerTable{}, err
	}
	if table.Tipo == models.TableTypeCashGame {
		var sessionID uuid.UUID
		if err := tx.QueryRow(ctx, `
			INSERT INTO cash_game_sessions (user_id, table_id, table_name)
			VALUES ($1, $2, $3)
			RETURNING id
		`, userID, tableID, table.Nome).Scan(&sessionID); err != nil {
			return nil, models.PokerTable{}, fmt.Errorf("create cash game history: %w", err)
		}
		if err := insertCashGameSessionBuyIn(ctx, tx, sessionID, userID, tableID, buyIn, idempotencyKey); err != nil {
			return nil, models.PokerTable{}, err
		}
	}
	if err := notifyTx(ctx, tx, "tables"); err != nil {
		return nil, models.PokerTable{}, err
	}
	if err := notifyTx(ctx, tx, "game:"+tableID.String()); err != nil {
		return nil, models.PokerTable{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, models.PokerTable{}, fmt.Errorf("commit player table join: %w", err)
	}
	return game, table, nil
}

func (s *Store) AddCashGameStack(ctx context.Context, tableID, userID uuid.UUID, amount int64, idempotencyKey string) (*engine.TableGame, error) {
	tx, err := s.beginTableTransaction(ctx, tableID)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	table, err := getTableTx(ctx, tx, tableID)
	if err != nil {
		return nil, err
	}
	if table.Tipo != models.TableTypeCashGame {
		return nil, errors.New("rebuys are only supported for cash games")
	}
	game, err := loadGameTx(ctx, tx, tableID, table.SmallBlind, table.BigBlind)
	if err != nil {
		return nil, err
	}
	if err := game.AddStack(userID, amount, idempotencyKey); err != nil {
		return nil, err
	}
	var sessionID uuid.UUID
	if err := tx.QueryRow(ctx, `
		SELECT id FROM cash_game_sessions
		WHERE user_id = $1 AND table_id = $2 AND finished_at IS NULL
		FOR UPDATE
	`, userID, tableID).Scan(&sessionID); err != nil {
		return nil, fmt.Errorf("load active cash game history: %w", err)
	}
	if err := insertCashGameSessionBuyIn(ctx, tx, sessionID, userID, tableID, amount, idempotencyKey); err != nil {
		return nil, err
	}
	if err := saveGameTx(ctx, tx, game); err != nil {
		return nil, err
	}
	if err := notifyTx(ctx, tx, "game:"+tableID.String()); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit cash game rebuy and history: %w", err)
	}
	return game, nil
}

func insertCashGameSessionBuyIn(ctx context.Context, tx pgx.Tx, sessionID, userID, tableID uuid.UUID, chips int64, idempotencyKey string) error {
	if idempotencyKey == "" {
		return errors.New("cash game buy-in requires an idempotency key")
	}
	storedKey := walletIdempotencyKey(userID, idempotencyKey)
	if len(storedKey) > 255 {
		return errors.New("cash game buy-in idempotency key exceeds 255 bytes")
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO cash_game_session_buyins (idempotency_key, session_id, amount_cents)
		VALUES ($1, $2, $3)
		ON CONFLICT (idempotency_key) DO NOTHING
	`, storedKey, sessionID, finance.ChipsToMoney(chips)); err != nil {
		return fmt.Errorf("record cash game buy-in history for user %s at table %s: %w", userID, tableID, err)
	}
	return nil
}

func (s *Store) FinishCashGameSession(ctx context.Context, tableID, userID uuid.UUID, payoutChips int64) error {
	result, err := s.pool.Exec(ctx, `
		UPDATE cash_game_sessions
		SET payout_cents = $1, finished_at = CURRENT_TIMESTAMP
		WHERE table_id = $2 AND user_id = $3 AND finished_at IS NULL
	`, finance.ChipsToMoney(payoutChips), tableID, userID)
	if err != nil {
		return fmt.Errorf("finish cash game history: %w", err)
	}
	if result.RowsAffected() > 1 {
		return fmt.Errorf("multiple active cash game sessions for user %s at table %s", userID, tableID)
	}
	return nil
}

func (s *Store) ListPlayerGameHistory(ctx context.Context, userID uuid.UUID) ([]models.GameHistoryEntry, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, table_name, game_type, amount_invested, payout, won, started_at, finished_at
		FROM (
			SELECT
				cgs.id,
				cgs.table_name,
				'cash_game'::TEXT AS game_type,
				COALESCE(SUM(cgb.amount_cents), 0)::BIGINT AS amount_invested,
				cgs.payout_cents::BIGINT AS payout,
				NULL::BOOLEAN AS won,
				cgs.started_at,
				cgs.finished_at
			FROM cash_game_sessions cgs
			LEFT JOIN cash_game_session_buyins cgb ON cgb.session_id = cgs.id
			WHERE cgs.user_id = $1
			GROUP BY cgs.id

			UNION ALL

			SELECT
				t.id,
				t.nome,
				'torneio'::TEXT AS game_type,
				t.buy_in::BIGINT AS amount_invested,
				te.prize_cents::BIGINT AS payout,
				CASE WHEN t.status = 'concluido' THEN te.status = 'winner' ELSE NULL END AS won,
				COALESCE(t.started_at, t.data_inicio) AS started_at,
				t.finished_at
			FROM tournament_entries te
			JOIN tournaments t ON t.id = te.tournament_id
			WHERE te.user_id = $1
		) history
		ORDER BY started_at DESC, id DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("query player game history: %w", err)
	}
	defer rows.Close()

	history := make([]models.GameHistoryEntry, 0)
	for rows.Next() {
		var entry models.GameHistoryEntry
		var amountInvested, payout int64
		if err := rows.Scan(
			&entry.ID, &entry.TableName, &entry.GameType, &amountInvested, &payout,
			&entry.Won, &entry.StartedAt, &entry.FinishedAt,
		); err != nil {
			return nil, fmt.Errorf("scan player game history: %w", err)
		}
		entry.AmountInvestedChips = finance.MoneyToChips(amountInvested)
		entry.PayoutChips = finance.MoneyToChips(payout)
		history = append(history, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate player game history: %w", err)
	}
	return history, nil
}

func (s *Store) UpdateGame(
	ctx context.Context,
	tableID uuid.UUID,
	smallBlind, bigBlind int64,
	update func(*engine.TableGame) error,
) (*engine.TableGame, error) {
	tx, err := s.beginTableTransaction(ctx, tableID)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	game, err := loadGameTx(ctx, tx, tableID, smallBlind, bigBlind)
	if err != nil {
		return nil, err
	}
	if err := update(game); err != nil {
		return nil, err
	}
	if err := saveGameTx(ctx, tx, game); err != nil {
		return nil, err
	}
	if err := notifyTx(ctx, tx, "game:"+tableID.String()); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit table game update: %w", err)
	}
	return game, nil
}

func (s *Store) LoadGame(ctx context.Context, tableID uuid.UUID) (*engine.TableGame, error) {
	var raw []byte
	err := s.pool.QueryRow(ctx, `SELECT state FROM poker_table_states WHERE table_id = $1`, tableID).Scan(&raw)
	if err != nil {
		return nil, err
	}
	var snapshot engine.TableSnapshot
	if err := json.Unmarshal(raw, &snapshot); err != nil {
		return nil, fmt.Errorf("decode game state for table %s: %w", tableID, err)
	}
	return engine.NewTableGameFromSnapshot(snapshot), nil
}

func (s *Store) Notify(ctx context.Context, event string) error {
	_, err := s.pool.Exec(ctx, `SELECT pg_notify($1, $2)`, notificationChannel, event)
	if err != nil {
		return fmt.Errorf("publish table event: %w", err)
	}
	return nil
}

func (s *Store) Listen(ctx context.Context, onEvent func(string)) error {
	conn, err := pgx.Connect(ctx, s.dsn)
	if err != nil {
		return fmt.Errorf("connect PostgreSQL listener: %w", err)
	}
	defer conn.Close(context.Background())
	if _, err := conn.Exec(ctx, `LISTEN `+notificationChannel); err != nil {
		return fmt.Errorf("listen for table events: %w", err)
	}
	for {
		notification, err := conn.WaitForNotification(ctx)
		if err != nil {
			return fmt.Errorf("wait for table event: %w", err)
		}
		onEvent(notification.Payload)
	}
}

func (s *Store) beginTableTransaction(ctx context.Context, tableID uuid.UUID) (pgx.Tx, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin table transaction: %w", err)
	}
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, tableID.String()); err != nil {
		tx.Rollback(ctx)
		return nil, fmt.Errorf("lock table transaction: %w", err)
	}
	return tx, nil
}

func getTableTx(ctx context.Context, tx pgx.Tx, tableID uuid.UUID) (models.PokerTable, error) {
	var table models.PokerTable
	var occupiedJSON, communityJSON []byte
	var creatorID *uuid.UUID
	err := tx.QueryRow(ctx, `
		SELECT id, nome, tipo, small_blind, big_blind, buy_in_min, buy_in_max,
			max_seats, status, current_pot, community_cards, current_dealer,
			current_turn, occupied_seats, password, created_by, creator_user_id,
			created_at, updated_at
		FROM poker_tables WHERE id = $1 FOR UPDATE
	`, tableID).Scan(
		&table.ID, &table.Nome, &table.Tipo, &table.SmallBlind, &table.BigBlind,
		&table.BuyInMin, &table.BuyInMax, &table.MaxSeats, &table.Status,
		&table.CurrentPot, &communityJSON, &table.CurrentDealer, &table.CurrentTurn,
		&occupiedJSON, &table.Password, &table.CreatedBy, &creatorID,
		&table.CreatedAt, &table.UpdatedAt,
	)
	if err != nil {
		return models.PokerTable{}, err
	}
	if err := json.Unmarshal(occupiedJSON, &table.OccupiedSeats); err != nil {
		return models.PokerTable{}, fmt.Errorf("decode occupied seats: %w", err)
	}
	if len(communityJSON) > 0 && string(communityJSON) != "null" {
		if err := json.Unmarshal(communityJSON, &table.CommunityCards); err != nil {
			return models.PokerTable{}, fmt.Errorf("decode community cards: %w", err)
		}
	}
	if creatorID != nil {
		table.CreatorUserID = *creatorID
	}
	return table, nil
}

func updateSeatsTx(ctx context.Context, tx pgx.Tx, table models.PokerTable) error {
	occupied, err := json.Marshal(table.OccupiedSeats)
	if err != nil {
		return fmt.Errorf("encode occupied seats: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE poker_tables SET occupied_seats = $2::jsonb, updated_at = $3 WHERE id = $1
	`, table.ID, occupied, table.UpdatedAt); err != nil {
		return fmt.Errorf("update occupied seats: %w", err)
	}
	return nil
}

func loadGameTx(ctx context.Context, tx pgx.Tx, tableID uuid.UUID, smallBlind, bigBlind int64) (*engine.TableGame, error) {
	var raw []byte
	err := tx.QueryRow(ctx, `SELECT state FROM poker_table_states WHERE table_id = $1 FOR UPDATE`, tableID).Scan(&raw)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return engine.NewWaitingTableGame(tableID, smallBlind, bigBlind), nil
		}
		return nil, fmt.Errorf("load table game: %w", err)
	}
	var snapshot engine.TableSnapshot
	if err := json.Unmarshal(raw, &snapshot); err != nil {
		return nil, fmt.Errorf("decode table game: %w", err)
	}
	return engine.NewTableGameFromSnapshot(snapshot), nil
}

func saveGameTx(ctx context.Context, tx pgx.Tx, game *engine.TableGame) error {
	raw, err := json.Marshal(game.Snapshot())
	if err != nil {
		return fmt.Errorf("encode table game: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO poker_table_states (table_id, state, updated_at)
		VALUES ($1, $2::jsonb, CURRENT_TIMESTAMP)
		ON CONFLICT (table_id) DO UPDATE
		SET state = EXCLUDED.state, updated_at = EXCLUDED.updated_at
	`, game.TableID, raw); err != nil {
		return fmt.Errorf("save table game: %w", err)
	}
	return nil
}

func notifyTx(ctx context.Context, tx pgx.Tx, event string) error {
	if _, err := tx.Exec(ctx, `SELECT pg_notify($1, $2)`, notificationChannel, event); err != nil {
		return fmt.Errorf("publish table event: %w", err)
	}
	return nil
}

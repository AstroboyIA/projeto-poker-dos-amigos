package database

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/walissonpaulo/poker-dos-amigos-backend/internal/engine"
	"github.com/walissonpaulo/poker-dos-amigos-backend/internal/models"
)

const notificationChannel = "poker_table_updates"

var (
	ErrTableNotFound = errors.New("table not found")
	ErrSeatOccupied  = errors.New("seat is already occupied")
	ErrInvalidSeat   = errors.New("invalid seat number")
)

//go:embed migrations/004_shared_table_state.sql
var sharedTableMigration string

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
	if _, err := s.pool.Exec(ctx, sharedTableMigration); err != nil {
		return fmt.Errorf("apply shared table migration: %w", err)
	}
	return nil
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

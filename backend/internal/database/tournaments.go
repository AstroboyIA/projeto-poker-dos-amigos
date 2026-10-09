package database

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/walissonpaulo/poker-dos-amigos-backend/internal/engine"
	"github.com/walissonpaulo/poker-dos-amigos-backend/internal/finance"
	"github.com/walissonpaulo/poker-dos-amigos-backend/internal/models"
)

var (
	ErrTournamentNotFound  = errors.New("tournament not found")
	ErrTournamentFull      = errors.New("tournament is full")
	ErrTournamentClosed    = errors.New("tournament is not open")
	ErrTournamentStart     = errors.New("tournament cannot start")
	ErrTournamentPlayers   = errors.New("at least two players are required")
	ErrTournamentForbidden = errors.New("only the tournament creator can start it")
	ErrTournamentEntry     = errors.New("player is not registered in tournament")
	ErrTournamentSuccessor = errors.New("another registered player is required to transfer tournament ownership")
)

func (s *Store) ListTournaments(ctx context.Context, userID uuid.UUID) ([]models.Tournament, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT t.id, t.nome, t.buy_in, t.prize_pool, t.max_inscritos, t.data_inicio,
			t.status, t.blind_interval_min, t.starting_stack,
			COALESCE(pt.small_blind, t.small_blind), COALESCE(pt.big_blind, t.big_blind),
			t.table_id, pt.creator_user_id, t.winner_user_id, COALESCE(w.nome_completo, w.username, ''),
			t.blind_level, t.started_at, t.finished_at,
			COUNT(e.user_id)::INT,
			COALESCE(MAX(e.status) FILTER (WHERE e.user_id = $1), ''),
			COALESCE(MAX(e.seat_number) FILTER (WHERE e.user_id = $1), 0),
			COALESCE(BOOL_OR(e.user_id = $1), FALSE)
		FROM tournaments t
		LEFT JOIN tournament_entries e ON e.tournament_id = t.id
		LEFT JOIN poker_tables pt ON pt.id = t.table_id
		LEFT JOIN users w ON w.id = t.winner_user_id
		GROUP BY t.id, pt.small_blind, pt.big_blind, pt.creator_user_id, w.nome_completo, w.username
		ORDER BY t.data_inicio DESC, t.created_at DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("query tournaments: %w", err)
	}
	defer rows.Close()

	tournaments := make([]models.Tournament, 0)
	for rows.Next() {
		var tournament models.Tournament
		var tableID, creatorID, winnerID *uuid.UUID
		var startedAt, finishedAt *time.Time
		var currentSeat int
		if err := rows.Scan(
			&tournament.ID, &tournament.Nome, &tournament.BuyIn, &tournament.Garantido,
			&tournament.MaxInscritos, &tournament.DataInicio, &tournament.Status,
			&tournament.BlindInterval, &tournament.StartingStack, &tournament.SmallBlind,
			&tournament.BigBlind, &tableID, &creatorID, &winnerID, &tournament.WinnerName, &tournament.BlindLevel,
			&startedAt, &finishedAt, &tournament.Inscritos, &tournament.CurrentUserStatus, &currentSeat,
			&tournament.CurrentUserEntry,
		); err != nil {
			return nil, fmt.Errorf("scan tournament: %w", err)
		}
		tournament.TableID = tableID
		tournament.CreatorUserID = creatorID
		tournament.WinnerUserID = winnerID
		tournament.StartedAt = startedAt
		tournament.FinishedAt = finishedAt
		if tournament.CurrentUserEntry {
			tournament.CurrentUserSeat = &currentSeat
		}
		tournaments = append(tournaments, tournament)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tournaments: %w", err)
	}
	return tournaments, nil
}

func (s *Store) CreateTournament(ctx context.Context, tournament models.Tournament, creatorID uuid.UUID, creatorName string) (models.Tournament, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return models.Tournament{}, fmt.Errorf("begin tournament creation: %w", err)
	}
	defer tx.Rollback(ctx)

	tournament.ID = uuid.New()
	tableID := uuid.New()
	tournament.CreatorUserID = &creatorID
	tournament.TableID = &tableID
	tournament.DataInicio = time.Now().UTC()
	tournament.Status = "aberto"
	if tournament.StartingStack <= 0 {
		tournament.StartingStack = 10000
	}
	if tournament.SmallBlind <= 0 {
		tournament.SmallBlind = 25
	}
	if tournament.BigBlind <= 0 {
		tournament.BigBlind = tournament.SmallBlind * 2
	}
	occupied := []byte("[]")
	_, err = tx.Exec(ctx, `
		INSERT INTO poker_tables (
			id, nome, tipo, small_blind, big_blind, buy_in_min, buy_in_max, max_seats,
			status, occupied_seats, created_by, creator_user_id, created_at, updated_at
		) VALUES ($1, $2, 'torneio', $3, $4, $5, $5, $6, 'aguardando', $7::jsonb, $8, $9, $10, $10)
	`, tableID, tournament.Nome, tournament.SmallBlind, tournament.BigBlind,
		tournament.BuyIn, tournament.MaxInscritos, occupied, creatorName, creatorID, tournament.DataInicio)
	if err != nil {
		return models.Tournament{}, fmt.Errorf("create tournament table: %w", err)
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO tournaments (
			id, nome, buy_in, garantido, max_inscritos, data_inicio, status,
			blind_interval_min, starting_stack, small_blind, big_blind, table_id,
			prize_pool, blind_level, created_at, updated_at
		) VALUES ($1, $2, $3, 0, $4, $5, 'aberto', $6, $7, $8, $9, $10, 0, 0, $5, $5)
	`, tournament.ID, tournament.Nome, tournament.BuyIn, tournament.MaxInscritos,
		tournament.DataInicio, tournament.BlindInterval, tournament.StartingStack,
		tournament.SmallBlind, tournament.BigBlind, tableID)
	if err != nil {
		return models.Tournament{}, fmt.Errorf("insert tournament: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return models.Tournament{}, fmt.Errorf("commit tournament creation: %w", err)
	}
	tournament.Garantido = 0
	tournament.Inscritos = 0
	return tournament, nil
}

func (s *Store) RegisterTournament(ctx context.Context, tournamentID, userID uuid.UUID, playerName string) (models.Tournament, int, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return models.Tournament{}, 0, fmt.Errorf("begin tournament registration: %w", err)
	}
	defer tx.Rollback(ctx)

	tournament, err := getTournamentTx(ctx, tx, tournamentID, true)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.Tournament{}, 0, ErrTournamentNotFound
	}
	if err != nil {
		return models.Tournament{}, 0, fmt.Errorf("load tournament for registration: %w", err)
	}
	var existingSeat int
	err = tx.QueryRow(ctx, `
		SELECT seat_number FROM tournament_entries
		WHERE tournament_id = $1 AND user_id = $2
	`, tournamentID, userID).Scan(&existingSeat)
	if err == nil {
		if err := tx.Commit(ctx); err != nil {
			return models.Tournament{}, 0, fmt.Errorf("commit existing tournament registration: %w", err)
		}
		return tournament, existingSeat, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return models.Tournament{}, 0, fmt.Errorf("check tournament registration: %w", err)
	}
	if tournament.Status != "aberto" {
		return models.Tournament{}, 0, ErrTournamentClosed
	}
	if tournament.Inscritos >= tournament.MaxInscritos {
		return models.Tournament{}, 0, ErrTournamentFull
	}

	var tableID uuid.UUID
	if tournament.TableID == nil {
		return models.Tournament{}, 0, errors.New("tournament table is not configured")
	}
	tableID = *tournament.TableID
	table, err := getTableTx(ctx, tx, tableID)
	if err != nil {
		return models.Tournament{}, 0, fmt.Errorf("load tournament table: %w", err)
	}
	seat := 0
	for candidate := 1; candidate <= tournament.MaxInscritos; candidate++ {
		occupied := false
		for _, seatNumber := range table.OccupiedSeats {
			if seatNumber == candidate {
				occupied = true
				break
			}
		}
		if !occupied {
			seat = candidate
			break
		}
	}
	if seat == 0 {
		return models.Tournament{}, 0, ErrTournamentFull
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO wallets (user_id, balance_cents, available_cents)
		SELECT id, COALESCE(saldo_fichas, 0), COALESCE(saldo_fichas, 0)
		FROM users WHERE id = $1
		ON CONFLICT (user_id) DO NOTHING
	`, userID); err != nil {
		return models.Tournament{}, 0, fmt.Errorf("ensure tournament wallet: %w", err)
	}
	wallet, err := scanWallet(tx.QueryRow(ctx, `
		SELECT id, user_id, balance_cents, available_cents, reserved_cents, updated_at
		FROM wallets WHERE user_id = $1 FOR UPDATE
	`, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return models.Tournament{}, 0, ErrUserNotFound
	}
	if err != nil {
		return models.Tournament{}, 0, fmt.Errorf("lock tournament wallet: %w", err)
	}
	if wallet.AvailableCents < tournament.BuyIn {
		return models.Tournament{}, 0, finance.ErrInsufficientFunds
	}

	before := wallet.BalanceCents
	wallet.AvailableCents -= tournament.BuyIn
	wallet.ReservedCents += tournament.BuyIn
	wallet.BalanceCents = wallet.AvailableCents + wallet.ReservedCents
	if _, err := tx.Exec(ctx, `
		UPDATE wallets SET balance_cents = $1, available_cents = $2,
			reserved_cents = $3, updated_at = CURRENT_TIMESTAMP WHERE id = $4
	`, wallet.BalanceCents, wallet.AvailableCents, wallet.ReservedCents, wallet.ID); err != nil {
		return models.Tournament{}, 0, fmt.Errorf("reserve tournament entry fee: %w", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE users SET saldo_fichas = $1, wallet_id = $2 WHERE id = $3`, wallet.AvailableCents, wallet.ID, userID); err != nil {
		return models.Tournament{}, 0, fmt.Errorf("sync tournament wallet balance: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO wallet_table_reservations (user_id, table_id, amount_cents)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, table_id) DO UPDATE
		SET amount_cents = wallet_table_reservations.amount_cents + EXCLUDED.amount_cents,
			updated_at = CURRENT_TIMESTAMP
	`, userID, tournamentID.String(), tournament.BuyIn); err != nil {
		return models.Tournament{}, 0, fmt.Errorf("record tournament wallet reservation: %w", err)
	}
	idempotencyKey := walletIdempotencyKey(userID, "tournament:"+tournamentID.String()+":entry:"+uuid.NewString())
	if _, err := tx.Exec(ctx, `
		INSERT INTO wallet_ledger (
			user_id, wallet_id, type, amount_cents, balance_before, balance_after,
			reference_type, reference_id, idempotency_key
		) VALUES ($1, $2, 'BUY_IN', $3, $4, $5, 'tournament', $6, $7)
	`, userID, wallet.ID, tournament.BuyIn, before, wallet.BalanceCents, tournamentID.String(), idempotencyKey); err != nil {
		return models.Tournament{}, 0, fmt.Errorf("record tournament entry fee: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO tournament_entries (tournament_id, user_id, seat_number)
		VALUES ($1, $2, $3)
	`, tournamentID, userID, seat); err != nil {
		return models.Tournament{}, 0, fmt.Errorf("insert tournament entry: %w", err)
	}
	tournament.Inscritos++
	tournament.Garantido += tournament.BuyIn
	if _, err := tx.Exec(ctx, `
		UPDATE tournaments SET prize_pool = $1, garantido = $1,
			updated_at = CURRENT_TIMESTAMP WHERE id = $2
	`, tournament.Garantido, tournamentID); err != nil {
		return models.Tournament{}, 0, fmt.Errorf("update tournament prize pool: %w", err)
	}

	game, err := loadGameTx(ctx, tx, tableID, tournament.SmallBlind, tournament.BigBlind)
	if err != nil {
		return models.Tournament{}, 0, err
	}
	if err := game.JoinPlayer(userID, playerName, seat, tournament.StartingStack); err != nil {
		return models.Tournament{}, 0, fmt.Errorf("join tournament table: %w", err)
	}
	table.OccupiedSeats = append(table.OccupiedSeats, seat)
	table.UpdatedAt = time.Now()
	if err := updateSeatsTx(ctx, tx, table); err != nil {
		return models.Tournament{}, 0, err
	}
	if err := saveGameTx(ctx, tx, game); err != nil {
		return models.Tournament{}, 0, err
	}
	if err := notifyTx(ctx, tx, "tables"); err != nil {
		return models.Tournament{}, 0, err
	}
	if err := notifyTx(ctx, tx, "game:"+tableID.String()); err != nil {
		return models.Tournament{}, 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return models.Tournament{}, 0, fmt.Errorf("commit tournament registration: %w", err)
	}
	return tournament, seat, nil
}

func (s *Store) LeaveTournament(ctx context.Context, tournamentID, userID uuid.UUID) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tournament leave: %w", err)
	}
	defer tx.Rollback(ctx)

	tournament, err := getTournamentTx(ctx, tx, tournamentID, true)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrTournamentNotFound
	}
	if err != nil {
		return fmt.Errorf("load tournament for leave: %w", err)
	}
	if tournament.Status != "aberto" {
		return ErrTournamentClosed
	}
	if tournament.TableID == nil {
		return errors.New("tournament table is not configured")
	}

	var seat int
	err = tx.QueryRow(ctx, `
		SELECT seat_number FROM tournament_entries
		WHERE tournament_id = $1 AND user_id = $2 AND status = 'registered'
		FOR UPDATE
	`, tournamentID, userID).Scan(&seat)
	hasEntry := err == nil
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("load tournament entry to leave: %w", err)
	}
	isCreator := tournament.CreatorUserID != nil && *tournament.CreatorUserID == userID
	if !hasEntry && !isCreator {
		return ErrTournamentEntry
	}

	if isCreator {
		var successorID uuid.UUID
		err := tx.QueryRow(ctx, `
			SELECT user_id FROM tournament_entries
			WHERE tournament_id = $1 AND user_id <> $2 AND status = 'registered'
			ORDER BY created_at, user_id
			LIMIT 1
		`, tournamentID, userID).Scan(&successorID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrTournamentSuccessor
		}
		if err != nil {
			return fmt.Errorf("select tournament successor: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			UPDATE poker_tables SET creator_user_id = $1, updated_at = CURRENT_TIMESTAMP
			WHERE id = $2
		`, successorID, *tournament.TableID); err != nil {
			return fmt.Errorf("transfer tournament ownership: %w", err)
		}
	}

	if hasEntry {
		wallet, err := scanWallet(tx.QueryRow(ctx, `
			SELECT id, user_id, balance_cents, available_cents, reserved_cents, updated_at
			FROM wallets WHERE user_id = $1 FOR UPDATE
		`, userID))
		if err != nil {
			return fmt.Errorf("lock wallet for tournament refund: %w", err)
		}
		var reserved int64
		if err := tx.QueryRow(ctx, `
			SELECT amount_cents FROM wallet_table_reservations
			WHERE user_id = $1 AND table_id = $2 FOR UPDATE
		`, userID, tournamentID.String()).Scan(&reserved); err != nil {
			return fmt.Errorf("load tournament reservation for refund: %w", err)
		}
		if reserved < tournament.BuyIn || wallet.ReservedCents < tournament.BuyIn {
			return fmt.Errorf("tournament entry reservation missing for user %s", userID)
		}

		now := time.Now().UTC()
		before := wallet.BalanceCents
		wallet.AvailableCents += tournament.BuyIn
		wallet.ReservedCents -= tournament.BuyIn
		wallet.BalanceCents = wallet.AvailableCents + wallet.ReservedCents
		wallet.UpdatedAt = now
		if _, err := tx.Exec(ctx, `
			UPDATE wallets SET balance_cents = $1, available_cents = $2,
				reserved_cents = $3, updated_at = $4 WHERE id = $5
		`, wallet.BalanceCents, wallet.AvailableCents, wallet.ReservedCents, now, wallet.ID); err != nil {
			return fmt.Errorf("refund tournament wallet: %w", err)
		}
		if _, err := tx.Exec(ctx, `UPDATE users SET saldo_fichas = $1 WHERE id = $2`, wallet.AvailableCents, userID); err != nil {
			return fmt.Errorf("sync refunded tournament balance: %w", err)
		}
		if reserved == tournament.BuyIn {
			if _, err := tx.Exec(ctx, `
				DELETE FROM wallet_table_reservations WHERE user_id = $1 AND table_id = $2
			`, userID, tournamentID.String()); err != nil {
				return fmt.Errorf("remove tournament reservation: %w", err)
			}
		} else if _, err := tx.Exec(ctx, `
			UPDATE wallet_table_reservations SET amount_cents = amount_cents - $3, updated_at = $4
			WHERE user_id = $1 AND table_id = $2
		`, userID, tournamentID.String(), tournament.BuyIn, now); err != nil {
			return fmt.Errorf("reduce tournament reservation: %w", err)
		}
		refundKey := walletIdempotencyKey(userID, "tournament:"+tournamentID.String()+":refund:"+uuid.NewString())
		if _, err := tx.Exec(ctx, `
			INSERT INTO wallet_ledger (
				user_id, wallet_id, type, amount_cents, balance_before, balance_after,
				reference_type, reference_id, idempotency_key, created_at
			) VALUES ($1, $2, 'TOURNAMENT_REFUND', $3, $4, $5, 'tournament', $6, $7, $8)
		`, userID, wallet.ID, tournament.BuyIn, before, wallet.BalanceCents, tournamentID.String(), refundKey, now); err != nil {
			return fmt.Errorf("record tournament refund: %w", err)
		}

		if _, err := tx.Exec(ctx, `
			DELETE FROM tournament_entries WHERE tournament_id = $1 AND user_id = $2
		`, tournamentID, userID); err != nil {
			return fmt.Errorf("remove tournament entry: %w", err)
		}
		poolUpdate, err := tx.Exec(ctx, `
			UPDATE tournaments
			SET prize_pool = prize_pool - $1, garantido = garantido - $1, updated_at = $2
			WHERE id = $3 AND prize_pool >= $1 AND garantido >= $1
		`, tournament.BuyIn, now, tournamentID)
		if err != nil {
			return fmt.Errorf("reduce tournament prize pool: %w", err)
		}
		if poolUpdate.RowsAffected() != 1 {
			return errors.New("tournament prize pool is inconsistent with the entry fee")
		}

		table, err := getTableTx(ctx, tx, *tournament.TableID)
		if err != nil {
			return fmt.Errorf("load tournament table for leave: %w", err)
		}
		occupied := table.OccupiedSeats[:0]
		for _, occupiedSeat := range table.OccupiedSeats {
			if occupiedSeat != seat {
				occupied = append(occupied, occupiedSeat)
			}
		}
		table.OccupiedSeats = occupied
		table.UpdatedAt = now
		if err := updateSeatsTx(ctx, tx, table); err != nil {
			return err
		}
		game, err := loadGameTx(ctx, tx, *tournament.TableID, tournament.SmallBlind, tournament.BigBlind)
		if err != nil {
			return err
		}
		if _, removed := game.LeavePlayerWithStack(userID); !removed {
			return fmt.Errorf("registered tournament player %s is missing from table state", userID)
		}
		if err := saveGameTx(ctx, tx, game); err != nil {
			return err
		}
		if err := notifyTx(ctx, tx, "game:"+tournament.TableID.String()); err != nil {
			return err
		}
	}

	if err := notifyTx(ctx, tx, "tables"); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tournament leave: %w", err)
	}
	return nil
}

func (s *Store) StartTournament(ctx context.Context, tournamentID, userID uuid.UUID) (*engine.TableGame, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tournament start: %w", err)
	}
	defer tx.Rollback(ctx)

	tournament, err := getTournamentTx(ctx, tx, tournamentID, true)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrTournamentNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("load tournament to start: %w", err)
	}
	if tournament.CreatorUserID == nil || *tournament.CreatorUserID != userID {
		return nil, ErrTournamentForbidden
	}
	if tournament.Status != "aberto" {
		return nil, ErrTournamentClosed
	}
	if tournament.Inscritos < 2 {
		return nil, ErrTournamentPlayers
	}
	if tournament.TableID == nil {
		return nil, errors.New("tournament table is not configured")
	}
	tableID := *tournament.TableID
	game, err := loadGameTx(ctx, tx, tableID, tournament.SmallBlind, tournament.BigBlind)
	if err != nil {
		return nil, err
	}
	if err := game.Start(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrTournamentStart, err)
	}
	now := time.Now().UTC()
	if _, err := tx.Exec(ctx, `
		UPDATE tournaments SET status = 'em_andamento', started_at = $1, updated_at = $1
		WHERE id = $2
	`, now, tournamentID); err != nil {
		return nil, fmt.Errorf("mark tournament as started: %w", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE tournament_entries SET status = 'playing', updated_at = $1 WHERE tournament_id = $2`, now, tournamentID); err != nil {
		return nil, fmt.Errorf("mark tournament entries as playing: %w", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE poker_tables SET status = 'em_jogo', updated_at = $1 WHERE id = $2`, now, tableID); err != nil {
		return nil, fmt.Errorf("mark tournament table as running: %w", err)
	}
	if err := saveGameTx(ctx, tx, game); err != nil {
		return nil, err
	}
	if err := notifyTx(ctx, tx, "tables"); err != nil {
		return nil, err
	}
	if err := notifyTx(ctx, tx, "game:"+tableID.String()); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tournament start: %w", err)
	}
	return game, nil
}

func (s *Store) ListRunningTournaments(ctx context.Context) ([]models.Tournament, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, nome, buy_in, prize_pool, max_inscritos, data_inicio, status,
			blind_interval_min, starting_stack, small_blind, big_blind, table_id,
			winner_user_id, blind_level, started_at, finished_at
		FROM tournaments WHERE status = 'em_andamento' AND table_id IS NOT NULL
		ORDER BY started_at, id
	`)
	if err != nil {
		return nil, fmt.Errorf("query running tournaments: %w", err)
	}
	defer rows.Close()
	tournaments := make([]models.Tournament, 0)
	for rows.Next() {
		var tournament models.Tournament
		var tableID, winnerID *uuid.UUID
		var startedAt, finishedAt *time.Time
		if err := rows.Scan(
			&tournament.ID, &tournament.Nome, &tournament.BuyIn, &tournament.Garantido,
			&tournament.MaxInscritos, &tournament.DataInicio, &tournament.Status,
			&tournament.BlindInterval, &tournament.StartingStack, &tournament.SmallBlind,
			&tournament.BigBlind, &tableID, &winnerID, &tournament.BlindLevel,
			&startedAt, &finishedAt,
		); err != nil {
			return nil, fmt.Errorf("scan running tournament: %w", err)
		}
		tournament.TableID = tableID
		tournament.WinnerUserID = winnerID
		tournament.StartedAt = startedAt
		tournament.FinishedAt = finishedAt
		tournaments = append(tournaments, tournament)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate running tournaments: %w", err)
	}
	return tournaments, nil
}

func (s *Store) AdvanceTournamentBlinds(ctx context.Context, tournamentID uuid.UUID, now time.Time) (*engine.TableGame, bool, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("begin blind level update: %w", err)
	}
	defer tx.Rollback(ctx)
	tournament, err := getTournamentTx(ctx, tx, tournamentID, true)
	if err != nil {
		return nil, false, fmt.Errorf("load tournament for blind update: %w", err)
	}
	if tournament.Status != "em_andamento" || tournament.StartedAt == nil || tournament.TableID == nil {
		return nil, false, nil
	}
	interval := time.Duration(tournament.BlindInterval) * time.Minute
	if interval <= 0 {
		return nil, false, errors.New("invalid tournament blind interval")
	}
	level := int(now.Sub(*tournament.StartedAt) / interval)
	if level > 30 {
		level = 30
	}
	if level <= tournament.BlindLevel {
		return nil, false, nil
	}
	tableID := *tournament.TableID
	game, err := loadGameTx(ctx, tx, tableID, tournament.SmallBlind, tournament.BigBlind)
	if err != nil {
		return nil, false, err
	}
	state := game.GetPublicState()
	if state.Stage != engine.StageWaiting && state.Stage != engine.StageShowdown && state.Stage != engine.StageHandOver {
		return nil, false, nil
	}
	factor := int64(1) << level
	if tournament.SmallBlind > math.MaxInt64/factor || tournament.BigBlind > math.MaxInt64/factor {
		return nil, false, errors.New("tournament blind level exceeds supported range")
	}
	smallBlind := tournament.SmallBlind * factor
	bigBlind := tournament.BigBlind * factor
	game.SmallBlind = smallBlind
	game.BigBlind = bigBlind
	if _, err := tx.Exec(ctx, `
		UPDATE tournaments SET blind_level = $1, updated_at = $2 WHERE id = $3
	`, level, now, tournamentID); err != nil {
		return nil, false, fmt.Errorf("update tournament blind level: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE poker_tables SET small_blind = $1, big_blind = $2, updated_at = $3 WHERE id = $4
	`, smallBlind, bigBlind, now, tableID); err != nil {
		return nil, false, fmt.Errorf("update tournament table blinds: %w", err)
	}
	if err := saveGameTx(ctx, tx, game); err != nil {
		return nil, false, err
	}
	if err := notifyTx(ctx, tx, "game:"+tableID.String()); err != nil {
		return nil, false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, false, fmt.Errorf("commit blind level update: %w", err)
	}
	return game, true, nil
}

func (s *Store) RemoveEliminatedTournamentPlayers(ctx context.Context, tournamentID uuid.UUID) (*engine.TableGame, bool, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("begin tournament elimination update: %w", err)
	}
	defer tx.Rollback(ctx)
	tournament, err := getTournamentTx(ctx, tx, tournamentID, true)
	if err != nil {
		return nil, false, fmt.Errorf("load tournament for elimination update: %w", err)
	}
	if tournament.Status != "em_andamento" || tournament.TableID == nil {
		return nil, false, nil
	}
	tableID := *tournament.TableID
	game, err := loadGameTx(ctx, tx, tableID, tournament.SmallBlind, tournament.BigBlind)
	if err != nil {
		return nil, false, err
	}
	state := game.GetPublicState()
	if state.Stage != engine.StageShowdown && state.Stage != engine.StageHandOver {
		return game, false, nil
	}
	eliminated := make([]uuid.UUID, 0)
	for _, player := range state.Players {
		if player.Stack <= 0 && player.UserID != nil {
			eliminated = append(eliminated, *player.UserID)
		}
	}
	if len(eliminated) == 0 {
		return game, false, nil
	}
	for _, userID := range eliminated {
		game.LeavePlayerWithStack(userID)
		if _, err := tx.Exec(ctx, `
			UPDATE tournament_entries SET status = 'eliminated', updated_at = CURRENT_TIMESTAMP
			WHERE tournament_id = $1 AND user_id = $2 AND status = 'playing'
		`, tournamentID, userID); err != nil {
			return nil, false, fmt.Errorf("mark tournament player %s eliminated: %w", userID, err)
		}
	}
	if err := saveGameTx(ctx, tx, game); err != nil {
		return nil, false, err
	}
	if err := notifyTx(ctx, tx, "game:"+tableID.String()); err != nil {
		return nil, false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, false, fmt.Errorf("commit tournament elimination update: %w", err)
	}
	return game, true, nil
}

func (s *Store) SettleTournament(ctx context.Context, tournamentID, winnerID uuid.UUID) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tournament settlement: %w", err)
	}
	defer tx.Rollback(ctx)
	tournament, err := getTournamentTx(ctx, tx, tournamentID, true)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrTournamentNotFound
	}
	if err != nil {
		return fmt.Errorf("load tournament to settle: %w", err)
	}
	if tournament.Status == "concluido" {
		return tx.Commit(ctx)
	}
	if tournament.Status != "em_andamento" {
		return ErrTournamentClosed
	}
	if tournament.TableID == nil {
		return errors.New("tournament table is not configured")
	}
	var winnerRegistered bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM tournament_entries WHERE tournament_id = $1 AND user_id = $2
		)
	`, tournamentID, winnerID).Scan(&winnerRegistered); err != nil {
		return fmt.Errorf("validate tournament winner: %w", err)
	}
	if !winnerRegistered {
		return errors.New("tournament winner is not registered")
	}
	rows, err := tx.Query(ctx, `
		SELECT user_id FROM tournament_entries
		WHERE tournament_id = $1 ORDER BY user_id
	`, tournamentID)
	if err != nil {
		return fmt.Errorf("load tournament entrants: %w", err)
	}
	userIDs := make([]uuid.UUID, 0, tournament.Inscritos)
	for rows.Next() {
		var userID uuid.UUID
		if err := rows.Scan(&userID); err != nil {
			rows.Close()
			return fmt.Errorf("scan tournament entrant: %w", err)
		}
		userIDs = append(userIDs, userID)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("iterate tournament entrants: %w", err)
	}
	rows.Close()
	if len(userIDs) < 2 {
		return ErrTournamentPlayers
	}
	now := time.Now().UTC()
	for _, userID := range userIDs {
		wallet, err := scanWallet(tx.QueryRow(ctx, `
			SELECT id, user_id, balance_cents, available_cents, reserved_cents, updated_at
			FROM wallets WHERE user_id = $1 FOR UPDATE
		`, userID))
		if err != nil {
			return fmt.Errorf("lock entrant wallet %s: %w", userID, err)
		}
		var reserved int64
		if err := tx.QueryRow(ctx, `
			SELECT amount_cents FROM wallet_table_reservations
			WHERE user_id = $1 AND table_id = $2 FOR UPDATE
		`, userID, tournamentID.String()).Scan(&reserved); err != nil {
			return fmt.Errorf("load tournament reservation for %s: %w", userID, err)
		}
		if reserved < tournament.BuyIn || wallet.ReservedCents < tournament.BuyIn {
			return fmt.Errorf("tournament entry reservation missing for user %s", userID)
		}
		if reserved == tournament.BuyIn {
			if _, err := tx.Exec(ctx, `DELETE FROM wallet_table_reservations WHERE user_id = $1 AND table_id = $2`, userID, tournamentID.String()); err != nil {
				return fmt.Errorf("release tournament reservation for %s: %w", userID, err)
			}
		} else if _, err := tx.Exec(ctx, `
			UPDATE wallet_table_reservations SET amount_cents = amount_cents - $3, updated_at = $4
			WHERE user_id = $1 AND table_id = $2
		`, userID, tournamentID.String(), tournament.BuyIn, now); err != nil {
			return fmt.Errorf("reduce tournament reservation for %s: %w", userID, err)
		}
		before := wallet.BalanceCents
		wallet.ReservedCents -= tournament.BuyIn
		transactionType := "TOURNAMENT_ENTRY"
		ledgerAmount := tournament.BuyIn
		prize := int64(0)
		if userID == winnerID {
			transactionType = "TOURNAMENT_PRIZE"
			ledgerAmount = tournament.Garantido
			prize = tournament.Garantido
			wallet.AvailableCents += prize
		}
		wallet.BalanceCents = wallet.AvailableCents + wallet.ReservedCents
		if _, err := tx.Exec(ctx, `
			UPDATE wallets SET balance_cents = $1, available_cents = $2,
				reserved_cents = $3, updated_at = $4 WHERE id = $5
		`, wallet.BalanceCents, wallet.AvailableCents, wallet.ReservedCents, now, wallet.ID); err != nil {
			return fmt.Errorf("settle wallet for %s: %w", userID, err)
		}
		if _, err := tx.Exec(ctx, `UPDATE users SET saldo_fichas = $1 WHERE id = $2`, wallet.AvailableCents, userID); err != nil {
			return fmt.Errorf("sync settled balance for %s: %w", userID, err)
		}
		idempotencyKey := walletIdempotencyKey(userID, "tournament:"+tournamentID.String()+":settlement")
		if _, err := tx.Exec(ctx, `
			INSERT INTO wallet_ledger (
				user_id, wallet_id, type, amount_cents, balance_before, balance_after,
				reference_type, reference_id, idempotency_key, created_at
			) VALUES ($1, $2, $3, $4, $5, $6, 'tournament', $7, $8, $9)
		`, userID, wallet.ID, transactionType, ledgerAmount, before, wallet.BalanceCents, tournamentID.String(), idempotencyKey, now); err != nil {
			return fmt.Errorf("record tournament settlement for %s: %w", userID, err)
		}
		if _, err := tx.Exec(ctx, `
			UPDATE tournament_entries SET status = $1, prize_cents = $2, updated_at = $3
			WHERE tournament_id = $4 AND user_id = $5
		`, map[bool]string{true: "winner", false: "eliminated"}[userID == winnerID], prize, now, tournamentID, userID); err != nil {
			return fmt.Errorf("update tournament result for %s: %w", userID, err)
		}
	}
	if _, err := tx.Exec(ctx, `
		UPDATE tournaments SET status = 'concluido', winner_user_id = $1,
			finished_at = $2, updated_at = $2 WHERE id = $3
	`, winnerID, now, tournamentID); err != nil {
		return fmt.Errorf("mark tournament as finished: %w", err)
	}
	tableID := *tournament.TableID
	if _, err := tx.Exec(ctx, `UPDATE poker_tables SET status = 'finalizada', updated_at = $1 WHERE id = $2`, now, tableID); err != nil {
		return fmt.Errorf("mark tournament table as finished: %w", err)
	}
	if err := notifyTx(ctx, tx, "tables"); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tournament settlement: %w", err)
	}
	return nil
}

func getTournamentTx(ctx context.Context, tx pgx.Tx, tournamentID uuid.UUID, lock bool) (models.Tournament, error) {
	query := `
		SELECT t.id, t.nome, t.buy_in, t.prize_pool, t.max_inscritos, t.data_inicio, t.status,
			t.blind_interval_min, t.starting_stack, t.small_blind, t.big_blind, t.table_id,
			pt.creator_user_id, t.winner_user_id, t.blind_level, t.started_at, t.finished_at
		FROM tournaments t
		LEFT JOIN poker_tables pt ON pt.id = t.table_id
		WHERE t.id = $1
	`
	if lock {
		query += " FOR UPDATE OF t"
	}
	var tournament models.Tournament
	var tableID, creatorID, winnerID *uuid.UUID
	var startedAt, finishedAt *time.Time
	err := tx.QueryRow(ctx, query, tournamentID).Scan(
		&tournament.ID, &tournament.Nome, &tournament.BuyIn, &tournament.Garantido,
		&tournament.MaxInscritos, &tournament.DataInicio, &tournament.Status,
		&tournament.BlindInterval, &tournament.StartingStack, &tournament.SmallBlind,
		&tournament.BigBlind, &tableID, &creatorID, &winnerID, &tournament.BlindLevel,
		&startedAt, &finishedAt,
	)
	if err != nil {
		return models.Tournament{}, err
	}
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*)::INT FROM tournament_entries WHERE tournament_id = $1
	`, tournamentID).Scan(&tournament.Inscritos); err != nil {
		return models.Tournament{}, fmt.Errorf("count tournament entries: %w", err)
	}
	tournament.TableID = tableID
	tournament.CreatorUserID = creatorID
	tournament.WinnerUserID = winnerID
	tournament.StartedAt = startedAt
	tournament.FinishedAt = finishedAt
	return tournament, nil
}

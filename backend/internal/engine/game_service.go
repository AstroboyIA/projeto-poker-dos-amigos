package engine

import (
	"fmt"
	"sort"
	"sync"

	"github.com/google/uuid"
	"github.com/walissonpaulo/poker-dos-amigos-backend/internal/models"
)

type GameStage string

const (
	StageWaiting  GameStage = "WAITING"
	StageDealing  GameStage = "DEALING"
	StagePreFlop  GameStage = "PRE_FLOP"
	StageFlop     GameStage = "FLOP"
	StageTurn     GameStage = "TURN"
	StageRiver    GameStage = "RIVER"
	StageShowdown GameStage = "SHOWDOWN"
	StageHandOver GameStage = "HAND_OVER"
)

type ActionType string

const (
	ActionFold  ActionType = "FOLD"
	ActionCheck ActionType = "CHECK"
	ActionCall  ActionType = "CALL"
	ActionRaise ActionType = "RAISE"
	ActionAllIn ActionType = "ALL_IN"
)

type HandEvaluation struct {
	Rank        int    `json:"rank"`
	RankName    string `json:"rank_name"`
	Score       int    `json:"score"`
	Description string `json:"description"`
}

type PlayerState struct {
	ID           int             `json:"id"`
	SeatNumber   int             `json:"seat_number"`
	UserID       *uuid.UUID      `json:"user_id,omitempty"`
	Name         string          `json:"name"`
	IsBot        bool            `json:"is_bot"`
	Stack        int64           `json:"stack"`
	CurrentBet   int64           `json:"current_bet"`
	Cards        []models.Card   `json:"-"` // Mantido privado!
	HasFolded    bool            `json:"has_folded"`
	IsAllIn      bool            `json:"is_all_in"`
	HasActed     bool            `json:"has_acted"`
	LastAction   string          `json:"last_action,omitempty"`
	IsDealer     bool            `json:"is_dealer"`
	IsSmallBlind bool            `json:"is_small_blind"`
	IsBigBlind   bool            `json:"is_big_blind"`
	HandEval     *HandEvaluation `json:"hand_eval,omitempty"`
}

type PublicPlayerInfo struct {
	ID           int             `json:"id"`
	SeatNumber   int             `json:"seat_number"`
	UserID       *uuid.UUID      `json:"user_id,omitempty"`
	Name         string          `json:"name"`
	IsBot        bool            `json:"is_bot"`
	Stack        int64           `json:"stack"`
	CurrentBet   int64           `json:"current_bet"`
	CardCount    int             `json:"card_count"`
	Cards        []models.Card   `json:"cards,omitempty"` // Só preenchido no Showdown se não deu fold
	HasFolded    bool            `json:"has_folded"`
	IsAllIn      bool            `json:"is_all_in"`
	HasActed     bool            `json:"has_acted"`
	LastAction   string          `json:"last_action,omitempty"`
	IsDealer     bool            `json:"is_dealer"`
	IsSmallBlind bool            `json:"is_small_blind"`
	IsBigBlind   bool            `json:"is_big_blind"`
	HandEval     *HandEvaluation `json:"hand_eval,omitempty"`
}

type TableStatePayload struct {
	TableID        uuid.UUID          `json:"table_id"`
	Stage          GameStage          `json:"stage"`
	HandNumber     int                `json:"hand_number"`
	Pot            int64              `json:"pot"`
	CurrentBet     int64              `json:"current_round_bet"`
	CurrentTurnIdx int                `json:"current_turn_idx"`
	DealerIdx      int                `json:"dealer_idx"`
	SmallBlind     int64              `json:"small_blind"`
	BigBlind       int64              `json:"big_blind"`
	CommunityCards []models.Card      `json:"community_cards"`
	Players        []PublicPlayerInfo `json:"players"`
	WinnerMessage  string             `json:"winner_message,omitempty"`
}

type TableGame struct {
	TableID        uuid.UUID
	SmallBlind     int64
	BigBlind       int64
	Players        []*PlayerState
	Stage          GameStage
	HandNumber     int
	Pot            int64
	CurrentBet     int64
	CurrentTurnIdx int
	DealerIdx      int
	CommunityCards []models.Card
	Deck           []models.Card
	WinnerMessage  string
	Shuffler       *PhysicalMemoryShuffler
	mu             sync.Mutex
}

type GameService struct {
	tables map[uuid.UUID]*TableGame
	mu     sync.RWMutex
}

func NewGameService() *GameService {
	return &GameService{
		tables: make(map[uuid.UUID]*TableGame),
	}
}

func (gs *GameService) GetOrCreateTable(tableID uuid.UUID, smallBlind, bigBlind int64) *TableGame {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	tg, exists := gs.tables[tableID]
	if !exists {
		if smallBlind <= 0 {
			smallBlind = 25
		}
		if bigBlind <= 0 {
			bigBlind = 50
		}
		tg = &TableGame{
			TableID:    tableID,
			SmallBlind: smallBlind,
			BigBlind:   bigBlind,
			Players:    make([]*PlayerState, 0),
			Stage:      StageWaiting,
			HandNumber: 1,
			Shuffler:   NewPhysicalMemoryShuffler(),
		}
		gs.tables[tableID] = tg
	}
	return tg
}

func (gs *GameService) GetTable(tableID uuid.UUID) (*TableGame, bool) {
	gs.mu.RLock()
	defer gs.mu.RUnlock()
	tg, exists := gs.tables[tableID]
	return tg, exists
}

// Adiciona ou reconecta jogador
func (tg *TableGame) JoinPlayer(userID uuid.UUID, name string, seatNumber int, buyIn int64) error {
	tg.mu.Lock()
	defer tg.mu.Unlock()

	// Verifica se jogador já está na mesa (reconectar)
	for _, p := range tg.Players {
		if p.UserID != nil && *p.UserID == userID {
			p.SeatNumber = seatNumber
			p.Name = name
			if p.Stack <= 0 && buyIn > 0 {
				p.Stack = buyIn
			}
			return nil
		}
	}

	// Verifica se assento já está ocupado
	for _, p := range tg.Players {
		if p.SeatNumber == seatNumber {
			// Se o assento era de um bot, podemos substituir pelo humano
			if p.IsBot {
				uid := userID
				p.UserID = &uid
				p.Name = name
				p.IsBot = false
				p.Stack = buyIn
				return nil
			}
			return fmt.Errorf("assento %d já ocupado", seatNumber)
		}
	}

	uid := userID
	newPlayer := &PlayerState{
		ID:         seatNumber,
		SeatNumber: seatNumber,
		UserID:     &uid,
		Name:       name,
		IsBot:      false,
		Stack:      buyIn,
		CurrentBet: 0,
		Cards:      make([]models.Card, 0),
	}

	tg.Players = append(tg.Players, newPlayer)
	sort.Slice(tg.Players, func(i, j int) bool {
		return tg.Players[i].SeatNumber < tg.Players[j].SeatNumber
	})

	if len(tg.Players) >= 2 && tg.Stage == StageWaiting {
		tg.startNewHandLocked()
	}

	return nil
}

// Adiciona Bot
func (tg *TableGame) AddBot(seatNumber int, name string, stack int64) error {
	tg.mu.Lock()
	defer tg.mu.Unlock()

	for _, p := range tg.Players {
		if p.SeatNumber == seatNumber {
			return fmt.Errorf("assento %d já ocupado", seatNumber)
		}
	}

	botPlayer := &PlayerState{
		ID:         seatNumber,
		SeatNumber: seatNumber,
		Name:       name,
		IsBot:      true,
		Stack:      stack,
		CurrentBet: 0,
		Cards:      make([]models.Card, 0),
	}

	tg.Players = append(tg.Players, botPlayer)
	sort.Slice(tg.Players, func(i, j int) bool {
		return tg.Players[i].SeatNumber < tg.Players[j].SeatNumber
	})

	if len(tg.Players) >= 2 && tg.Stage == StageWaiting {
		tg.startNewHandLocked()
	}

	return nil
}

// Remove jogador da mesa
func (tg *TableGame) LeavePlayer(userID uuid.UUID) bool {
	tg.mu.Lock()
	defer tg.mu.Unlock()

	idx := -1
	for i, p := range tg.Players {
		if p.UserID != nil && *p.UserID == userID {
			idx = i
			break
		}
	}

	if idx == -1 {
		return false
	}

	// Se estava na mão ativa, dá fold
	player := tg.Players[idx]
	player.HasFolded = true

	tg.Players = append(tg.Players[:idx], tg.Players[idx+1:]...)

	if len(tg.Players) < 2 {
		tg.Stage = StageWaiting
		tg.CommunityCards = nil
		tg.Pot = 0
		tg.CurrentBet = 0
	} else if tg.Stage != StageWaiting && tg.Stage != StageShowdown && tg.Stage != StageHandOver {
		// A remoção desloca os índices dos jogadores que estavam depois dele.
		if idx < tg.CurrentTurnIdx {
			tg.CurrentTurnIdx--
		}
		if idx < tg.DealerIdx {
			tg.DealerIdx--
		}
		// Se era a vez dele, avança turno
		if tg.CurrentTurnIdx >= len(tg.Players) {
			tg.CurrentTurnIdx = 0
		}
		if tg.DealerIdx >= len(tg.Players) {
			tg.DealerIdx = 0
		}
		tg.checkRoundOrSurvivorLocked()
	}

	return true
}

func (tg *TableGame) StartNewHand() {
	tg.mu.Lock()
	defer tg.mu.Unlock()
	tg.startNewHandLocked()
}

func (tg *TableGame) startNewHandLocked() {
	if len(tg.Players) < 2 {
		tg.Stage = StageWaiting
		return
	}

	if tg.Stage == StageShowdown || tg.Stage == StageHandOver {
		tg.HandNumber++
	}
	tg.WinnerMessage = ""
	tg.CommunityCards = nil
	tg.Pot = 0
	tg.CurrentBet = tg.BigBlind

	shuffled, err := tg.Shuffler.Shuffle(tg.Deck)
	if err != nil || len(shuffled) != 52 {
		shuffled = NewStandardDeck()
	}

	cardIdx := 0
	numPlayers := len(tg.Players)
	tg.DealerIdx = (tg.DealerIdx + 1) % numPlayers
	sbIdx := (tg.DealerIdx + 1) % numPlayers
	bbIdx := (tg.DealerIdx + 2) % numPlayers
	firstActIdx := (tg.DealerIdx + 3) % numPlayers

	for idx, p := range tg.Players {
		bet := int64(0)
		p.HasFolded = false
		p.IsAllIn = false
		p.HasActed = false
		p.LastAction = ""
		p.IsDealer = (idx == tg.DealerIdx)
		p.IsSmallBlind = (idx == sbIdx)
		p.IsBigBlind = (idx == bbIdx)
		p.HandEval = nil

		if idx == sbIdx {
			sbAmt := tg.SmallBlind
			if sbAmt > p.Stack {
				sbAmt = p.Stack
			}
			p.Stack -= sbAmt
			bet = sbAmt
			p.LastAction = fmt.Sprintf("Small Blind ($%d)", sbAmt)
		} else if idx == bbIdx {
			bbAmt := tg.BigBlind
			if bbAmt > p.Stack {
				bbAmt = p.Stack
			}
			p.Stack -= bbAmt
			bet = bbAmt
			p.LastAction = fmt.Sprintf("Big Blind ($%d)", bbAmt)
		}

		p.CurrentBet = bet
		if p.Stack == 0 {
			p.IsAllIn = true
		}

		p.Cards = []models.Card{shuffled[cardIdx], shuffled[cardIdx+1]}
		cardIdx += 2
	}

	tg.Pot = tg.SmallBlind + tg.BigBlind
	tg.Deck = shuffled[cardIdx:]
	tg.CurrentTurnIdx = firstActIdx
	tg.Stage = StagePreFlop

	// Se o primeiro jogador já estiver all-in, avança
	if tg.Players[tg.CurrentTurnIdx].IsAllIn {
		tg.advanceTurnLocked(tg.CurrentTurnIdx)
	}
}

// Processa Ação do Jogador
func (tg *TableGame) ProcessAction(userID uuid.UUID, action ActionType, customAmount int64) error {
	tg.mu.Lock()
	defer tg.mu.Unlock()

	if tg.Stage == StageWaiting || tg.Stage == StageShowdown || tg.Stage == StageHandOver {
		return fmt.Errorf("mesa não está em rodada de apostas")
	}

	if tg.CurrentTurnIdx < 0 || tg.CurrentTurnIdx >= len(tg.Players) {
		return fmt.Errorf("índice de turno inválido")
	}

	player := tg.Players[tg.CurrentTurnIdx]
	if player.UserID == nil || *player.UserID != userID {
		return fmt.Errorf("não é sua vez de agir")
	}

	if player.HasFolded || player.IsAllIn {
		return fmt.Errorf("jogador não pode agir")
	}

	player.HasActed = true

	switch action {
	case ActionFold:
		player.HasFolded = true
		player.LastAction = "Desistiu (Fold)"

	case ActionCheck:
		if player.CurrentBet < tg.CurrentBet {
			return fmt.Errorf("não é possível dar check com aposta pendente")
		}
		player.LastAction = "Passou a vez (Check)"

	case ActionCall:
		needed := tg.CurrentBet - player.CurrentBet
		callAmt := needed
		if callAmt > player.Stack {
			callAmt = player.Stack
		}
		player.Stack -= callAmt
		player.CurrentBet += callAmt
		tg.Pot += callAmt
		if player.Stack == 0 {
			player.IsAllIn = true
		}
		player.LastAction = fmt.Sprintf("Pagou $%d (Call)", callAmt)

	case ActionRaise:
		targetBet := customAmount
		if targetBet <= tg.CurrentBet {
			targetBet = tg.CurrentBet + tg.BigBlind
		}
		additional := targetBet - player.CurrentBet
		actualAmt := additional
		if actualAmt > player.Stack {
			actualAmt = player.Stack
		}
		player.Stack -= actualAmt
		player.CurrentBet += actualAmt
		tg.Pot += actualAmt
		tg.CurrentBet = player.CurrentBet

		// Reseta hasActed dos outros jogadores ativos
		for i, p := range tg.Players {
			if i != tg.CurrentTurnIdx && !p.HasFolded && !p.IsAllIn {
				p.HasActed = false
			}
		}

		if player.Stack == 0 {
			player.IsAllIn = true
		}
		player.LastAction = fmt.Sprintf("Aumentou para $%d (Raise)", player.CurrentBet)

	case ActionAllIn:
		allInAmt := player.Stack
		player.CurrentBet += allInAmt
		player.Stack = 0
		player.IsAllIn = true
		tg.Pot += allInAmt
		if player.CurrentBet > tg.CurrentBet {
			tg.CurrentBet = player.CurrentBet
			for i, p := range tg.Players {
				if i != tg.CurrentTurnIdx && !p.HasFolded && !p.IsAllIn {
					p.HasActed = false
				}
			}
		}
		player.LastAction = fmt.Sprintf("ALL-IN ($%d) 🔥", allInAmt)

	default:
		return fmt.Errorf("ação desconhecida: %s", action)
	}

	tg.checkRoundOrSurvivorLocked()
	return nil
}

func (tg *TableGame) checkRoundOrSurvivorLocked() {
	// Verifica se sobrou apenas 1 não-desistente
	activeNonFolded := make([]*PlayerState, 0)
	for _, p := range tg.Players {
		if !p.HasFolded {
			activeNonFolded = append(activeNonFolded, p)
		}
	}

	if len(activeNonFolded) == 1 {
		tg.handleSingleSurvivorLocked(activeNonFolded[0])
		return
	}

	// Verifica se a rodada de apostas acabou
	activePlayers := make([]*PlayerState, 0)
	for _, p := range tg.Players {
		if !p.HasFolded && !p.IsAllIn {
			activePlayers = append(activePlayers, p)
		}
	}

	isRoundDone := len(activePlayers) == 0
	if !isRoundDone {
		allActedAndEqual := true
		for _, p := range activePlayers {
			if !p.HasActed || p.CurrentBet != tg.CurrentBet {
				allActedAndEqual = false
				break
			}
		}
		isRoundDone = allActedAndEqual
	}

	if isRoundDone {
		tg.advanceStreetLocked()
	} else {
		tg.advanceTurnLocked(tg.CurrentTurnIdx)
	}
}

func (tg *TableGame) advanceTurnLocked(fromIdx int) {
	n := len(tg.Players)
	if n == 0 {
		return
	}
	nextIdx := (fromIdx + 1) % n
	attempts := 0

	for attempts < n {
		candidate := tg.Players[nextIdx]
		if !candidate.HasFolded && !candidate.IsAllIn {
			tg.CurrentTurnIdx = nextIdx
			return
		}
		nextIdx = (nextIdx + 1) % n
		attempts++
	}

	tg.advanceStreetLocked()
}

func (tg *TableGame) advanceStreetLocked() {
	for _, p := range tg.Players {
		p.CurrentBet = 0
		p.HasActed = false
	}
	tg.CurrentBet = 0

	switch tg.Stage {
	case StagePreFlop:
		if len(tg.Deck) >= 3 {
			tg.CommunityCards = append(tg.CommunityCards, tg.Deck[0], tg.Deck[1], tg.Deck[2])
			tg.Deck = tg.Deck[3:]
		}
		tg.Stage = StageFlop
		tg.setFirstTurnOfStreetLocked()

	case StageFlop:
		if len(tg.Deck) >= 1 {
			tg.CommunityCards = append(tg.CommunityCards, tg.Deck[0])
			tg.Deck = tg.Deck[1:]
		}
		tg.Stage = StageTurn
		tg.setFirstTurnOfStreetLocked()

	case StageTurn:
		if len(tg.Deck) >= 1 {
			tg.CommunityCards = append(tg.CommunityCards, tg.Deck[0])
			tg.Deck = tg.Deck[1:]
		}
		tg.Stage = StageRiver
		tg.setFirstTurnOfStreetLocked()

	case StageRiver:
		tg.executeShowdownLocked()
	}
}

func (tg *TableGame) setFirstTurnOfStreetLocked() {
	n := len(tg.Players)
	if n == 0 {
		return
	}
	nextIdx := (tg.DealerIdx + 1) % n
	for i := 0; i < n; i++ {
		idx := (nextIdx + i) % n
		if !tg.Players[idx].HasFolded && !tg.Players[idx].IsAllIn {
			tg.CurrentTurnIdx = idx
			return
		}
	}
	// Se todos estão all-in / folded, vai para próxima rua
	tg.advanceStreetLocked()
}

func (tg *TableGame) handleSingleSurvivorLocked(winner *PlayerState) {
	tg.Stage = StageShowdown
	winner.Stack += tg.Pot
	tg.WinnerMessage = fmt.Sprintf("🏆 %s venceu o pote de $%d (Todos deram Fold)!", winner.Name, tg.Pot)
}

func (tg *TableGame) executeShowdownLocked() {
	tg.Stage = StageShowdown

	var bestScore = -1
	var winners []*PlayerState

	for _, p := range tg.Players {
		if !p.HasFolded {
			allCards := append([]models.Card{}, p.Cards...)
			allCards = append(allCards, tg.CommunityCards...)
			eval := Evaluate7Cards(allCards)
			p.HandEval = &eval

			if eval.Score > bestScore {
				bestScore = eval.Score
				winners = []*PlayerState{p}
			} else if eval.Score == bestScore {
				winners = append(winners, p)
			}
		}
	}

	if len(winners) > 0 {
		prize := tg.Pot / int64(len(winners))
		winnerNames := ""
		for i, w := range winners {
			w.Stack += prize
			if i > 0 {
				winnerNames += ", "
			}
			winnerNames += w.Name
		}
		handDesc := winners[0].HandEval.Description
		tg.WinnerMessage = fmt.Sprintf("🎉 %s venceu o pote de $%d com %s!", winnerNames, tg.Pot, handDesc)
	}

}

// Retorna estado público seguro (sem revelar cartas fechadas de outros)
func (tg *TableGame) GetPublicState() TableStatePayload {
	tg.mu.Lock()
	defer tg.mu.Unlock()

	publicPlayers := make([]PublicPlayerInfo, 0, len(tg.Players))
	for _, p := range tg.Players {
		info := PublicPlayerInfo{
			ID:           p.ID,
			SeatNumber:   p.SeatNumber,
			UserID:       p.UserID,
			Name:         p.Name,
			IsBot:        p.IsBot,
			Stack:        p.Stack,
			CurrentBet:   p.CurrentBet,
			CardCount:    len(p.Cards),
			HasFolded:    p.HasFolded,
			IsAllIn:      p.IsAllIn,
			HasActed:     p.HasActed,
			LastAction:   p.LastAction,
			IsDealer:     p.IsDealer,
			IsSmallBlind: p.IsSmallBlind,
			IsBigBlind:   p.IsBigBlind,
			HandEval:     p.HandEval,
		}
		// Apenas no Showdown e se não deu Fold, as cartas são reveladas publicamente
		if tg.Stage == StageShowdown && !p.HasFolded {
			info.Cards = p.Cards
		}
		publicPlayers = append(publicPlayers, info)
	}

	commCards := tg.CommunityCards
	if commCards == nil {
		commCards = make([]models.Card, 0)
	}

	return TableStatePayload{
		TableID:        tg.TableID,
		Stage:          tg.Stage,
		HandNumber:     tg.HandNumber,
		Pot:            tg.Pot,
		CurrentBet:     tg.CurrentBet,
		CurrentTurnIdx: tg.CurrentTurnIdx,
		DealerIdx:      tg.DealerIdx,
		SmallBlind:     tg.SmallBlind,
		BigBlind:       tg.BigBlind,
		CommunityCards: commCards,
		Players:        publicPlayers,
		WinnerMessage:  tg.WinnerMessage,
	}
}

// Retorna as cartas privadas do usuário
func (tg *TableGame) GetPrivateCards(userID uuid.UUID) []models.Card {
	tg.mu.Lock()
	defer tg.mu.Unlock()

	for _, p := range tg.Players {
		if p.UserID != nil && *p.UserID == userID {
			cards := make([]models.Card, len(p.Cards))
			copy(cards, p.Cards)
			return cards
		}
	}
	return nil
}

// Avaliador de Mãos Texas Hold'em 7 Cartas
func Evaluate7Cards(cards []models.Card) HandEvaluation {
	if len(cards) < 5 {
		if len(cards) == 2 {
			if cards[0].Value == cards[1].Value {
				return HandEvaluation{
					Rank:        2,
					RankName:    "Par",
					Score:       200 + cards[0].Value,
					Description: fmt.Sprintf("Par de %s", getCardName(cards[0].Value)),
				}
			}
			high := cards[0].Value
			if cards[1].Value > high {
				high = cards[1].Value
			}
			return HandEvaluation{
				Rank:        1,
				RankName:    "Carta Alta",
				Score:       100 + high,
				Description: fmt.Sprintf("Carta Alta %s", getCardName(high)),
			}
		}
		return HandEvaluation{Rank: 1, RankName: "Carta Alta", Score: 100, Description: "Carta Alta"}
	}

	valCounts := make(map[int]int)
	suitCounts := make(map[models.CardSuit][]models.Card)

	for _, c := range cards {
		valCounts[c.Value]++
		suitCounts[c.Suit] = append(suitCounts[c.Suit], c)
	}

	valSet := make(map[int]bool)
	for _, c := range cards {
		valSet[c.Value] = true
	}
	sortedVals := make([]int, 0, len(valSet))
	for v := range valSet {
		sortedVals = append(sortedVals, v)
	}
	sort.Slice(sortedVals, func(i, j int) bool {
		return sortedVals[i] > sortedVals[j]
	})

	// 1. Flush e Straight Flush
	var flushSuit models.CardSuit
	hasFlush := false
	for suit, sc := range suitCounts {
		if len(sc) >= 5 {
			flushSuit = suit
			hasFlush = true
			break
		}
	}

	if hasFlush {
		flushCards := suitCounts[flushSuit]
		flushValSet := make(map[int]bool)
		for _, c := range flushCards {
			flushValSet[c.Value] = true
		}
		flushVals := make([]int, 0, len(flushValSet))
		for v := range flushValSet {
			flushVals = append(flushVals, v)
		}
		sort.Slice(flushVals, func(i, j int) bool {
			return flushVals[i] > flushVals[j]
		})

		sfHigh := getStraightHigh(flushVals)
		if sfHigh > 0 {
			if sfHigh == 14 {
				return HandEvaluation{Rank: 10, RankName: "Royal Flush", Score: 1000, Description: "Royal Flush Imbatível!"}
			}
			return HandEvaluation{
				Rank:        9,
				RankName:    "Straight Flush",
				Score:       900 + sfHigh,
				Description: fmt.Sprintf("Straight Flush para o %s", getCardName(sfHigh)),
			}
		}
	}

	// 2. Quadra
	for val, count := range valCounts {
		if count == 4 {
			return HandEvaluation{
				Rank:        8,
				RankName:    "Quadra",
				Score:       800 + val,
				Description: fmt.Sprintf("Quadra de %s", getCardName(val)),
			}
		}
	}

	// 3. Full House
	triples := make([]int, 0)
	pairs := make([]int, 0)
	for val, count := range valCounts {
		if count == 3 {
			triples = append(triples, val)
		} else if count == 2 {
			pairs = append(pairs, val)
		}
	}
	sort.Slice(triples, func(i, j int) bool { return triples[i] > triples[j] })
	sort.Slice(pairs, func(i, j int) bool { return pairs[i] > pairs[j] })

	if len(triples) >= 2 || (len(triples) >= 1 && len(pairs) >= 1) {
		mainTriple := triples[0]
		mainPair := 0
		if len(triples) >= 2 {
			mainPair = triples[1]
		} else {
			mainPair = pairs[0]
		}
		return HandEvaluation{
			Rank:        7,
			RankName:    "Full House",
			Score:       700 + mainTriple*2 + mainPair,
			Description: fmt.Sprintf("Full House de %s com %s", getCardName(mainTriple), getCardName(mainPair)),
		}
	}

	// 4. Flush
	if hasFlush {
		flushCards := suitCounts[flushSuit]
		sort.Slice(flushCards, func(i, j int) bool {
			return flushCards[i].Value > flushCards[j].Value
		})
		high := flushCards[0].Value
		return HandEvaluation{
			Rank:        6,
			RankName:    "Flush",
			Score:       600 + high,
			Description: fmt.Sprintf("Flush com Carta Alta %s", getCardName(high)),
		}
	}

	// 5. Straight
	straightHigh := getStraightHigh(sortedVals)
	if straightHigh > 0 {
		return HandEvaluation{
			Rank:        5,
			RankName:    "Sequência (Straight)",
			Score:       500 + straightHigh,
			Description: fmt.Sprintf("Sequência para o %s", getCardName(straightHigh)),
		}
	}

	// 6. Trinca
	if len(triples) >= 1 {
		return HandEvaluation{
			Rank:        4,
			RankName:    "Trinca",
			Score:       400 + triples[0],
			Description: fmt.Sprintf("Trinca de %s", getCardName(triples[0])),
		}
	}

	// 7. Dois Pares
	if len(pairs) >= 2 {
		return HandEvaluation{
			Rank:        3,
			RankName:    "Dois Pares",
			Score:       300 + pairs[0]*2 + pairs[1],
			Description: fmt.Sprintf("Dois Pares de %s e %s", getCardName(pairs[0]), getCardName(pairs[1])),
		}
	}

	// 8. Um Par
	if len(pairs) == 1 {
		return HandEvaluation{
			Rank:        2,
			RankName:    "Um Par",
			Score:       200 + pairs[0],
			Description: fmt.Sprintf("Par de %s", getCardName(pairs[0])),
		}
	}

	// 9. Carta Alta
	high := 14
	if len(sortedVals) > 0 {
		high = sortedVals[0]
	}
	return HandEvaluation{
		Rank:        1,
		RankName:    "Carta Alta",
		Score:       100 + high,
		Description: fmt.Sprintf("Carta Alta %s", getCardName(high)),
	}
}

func getStraightHigh(values []int) int {
	vals := append([]int{}, values...)
	hasAce := false
	for _, v := range vals {
		if v == 14 {
			hasAce = true
			break
		}
	}
	if hasAce {
		vals = append(vals, 1) // Ás baixo
	}

	if len(vals) < 5 {
		return 0
	}

	for i := 0; i <= len(vals)-5; i++ {
		if vals[i]-vals[i+1] == 1 &&
			vals[i+1]-vals[i+2] == 1 &&
			vals[i+2]-vals[i+3] == 1 &&
			vals[i+3]-vals[i+4] == 1 {
			return vals[i]
		}
	}
	return 0
}

func getCardName(value int) string {
	switch value {
	case 14:
		return "Ás"
	case 13:
		return "Rei"
	case 12:
		return "Dama"
	case 11:
		return "Valete"
	case 10:
		return "Dez"
	default:
		return fmt.Sprintf("%d", value)
	}
}

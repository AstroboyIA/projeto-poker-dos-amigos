import React, { useState, useEffect, useRef } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import {
  Volume2,
  VolumeX,
  History,
  Trophy,
  Sparkles,
  Flame,
  Plus,
  LogOut,
  AlertTriangle,
} from 'lucide-react';
import { useAuth } from '../context/AuthContext';
import type { Card, ServerTableState, PrivateCardsPayload } from '../types';
import {
  shuffleContinuousDeck,
  evaluate7Cards,
} from '../utils/pokerEngine';
import type { HandEvaluation } from '../utils/pokerEngine';
import { ChipStack, PokerChip } from '../components/poker/PokerChip';
import { soundFX } from '../utils/audio';
import { getTableRoomById, leaveTableSeatRemote } from '../utils/tableRooms';
import { api } from '../services/api';

type GameStage = 'WAITING' | 'DEALING' | 'PRE_FLOP' | 'FLOP' | 'TURN' | 'RIVER' | 'SHOWDOWN' | 'HAND_OVER';

interface TablePlayer {
  id: number;
  seatNumber: number;
  userId?: string;
  name: string;
  isUser: boolean;
  isBot?: boolean;
  stack: number;
  currentBet: number;
  cards: Card[];
  hasFolded: boolean;
  isAllIn: boolean;
  hasActed: boolean;
  lastAction?: string;
  isDealer?: boolean;
  isSmallBlind?: boolean;
  isBigBlind?: boolean;
  handEval?: HandEvaluation;
}

interface ActionLog {
  id: string;
  text: string;
  timestamp: string;
}

export const PokerTablePage: React.FC = () => {
  const { user, token, updateUser } = useAuth();
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();

  // Parâmetros de entrada
  const tableId = searchParams.get('tableId') || 'mesa-vip-01';
  const room = getTableRoomById(tableId);
  const chosenSeat = Number(searchParams.get('seat')) || 3;
  const initialBuyIn = Number(searchParams.get('buyIn')) || 2500;
  const botSeatsParam = searchParams.get('bots');
  const configuredBotSeats = new Set(
    (botSeatsParam ?? room?.botSeats.join(',') ?? '')
      .split(',')
      .map((seat) => Number(seat))
      .filter((seat) => Number.isInteger(seat) && seat >= 1 && seat <= 9 && seat !== chosenSeat)
  );
  const occupiedHumanSeats = new Set(room?.occupiedSeats || [1, 2, 4, 6]);

  // Configuração da Mesa (9-Max)
  const tableName = `${room?.name || "Mesa VIP Ouro #01 (Texas Hold'em)"} (9-Max)`;
  const smallBlindVal = room?.smallBlind || 25;
  const bigBlindVal = room?.bigBlind || 50;
  const MAX_ACTION_TIME = 20; // Tempo máximo de aposta aumentado para 20 segundos

  // Estado Geral do Jogo
  const defaultNames: Record<number, string> = {
    1: room?.createdBy || 'Jonatas (Sócio)',
    2: 'Felipe (Sócio)',
    3: 'Alex King',
    4: "Bruno 'AllIn'",
    5: 'Diego Bluff',
    6: 'Carlos Shark',
    7: 'Lucas Pro',
    8: 'Rafael Tight',
    9: 'Marcelo Call',
  };

  // Se a sala for personalizada e não tiver outros humanos além do criador ou bots, não injeta jogadores fantasmas
  const isPresetRoom = tableId === 'mesa-vip-01' || tableId === 'mesa-amigos-fechada' || tableId.startsWith('a1111111') || tableId.startsWith('a2222222');

  const resolvedPlayers: TablePlayer[] = [];

  // Adiciona o jogador atual
  resolvedPlayers.push({
    id: chosenSeat,
    seatNumber: chosenSeat,
    name: user?.nome_completo || 'Você (VIP)',
    isUser: true,
    stack: initialBuyIn,
    currentBet: 0,
    cards: [],
    hasFolded: false,
    isAllIn: false,
    hasActed: false,
  });

  // Adiciona os bots configurados
  configuredBotSeats.forEach((seat) => {
    resolvedPlayers.push({
      id: seat,
      seatNumber: seat,
      name: `Bot #${seat} (${defaultNames[seat]?.split(' ')[0] || 'Player'})`,
      isUser: false,
      stack: initialBuyIn,
      currentBet: 0,
      cards: [],
      hasFolded: false,
      isAllIn: false,
      hasActed: false,
    });
  });

  // Se for uma sala preset demo, adiciona os outros assentos ocupados
  if (isPresetRoom) {
    occupiedHumanSeats.forEach((seat) => {
      if (seat !== chosenSeat && !configuredBotSeats.has(seat) && !resolvedPlayers.some((p) => p.id === seat)) {
        resolvedPlayers.push({
          id: seat,
          seatNumber: seat,
          name: defaultNames[seat] || `Jogador #${seat}`,
          isUser: false,
          stack: [4850, 5200, 3400, 2100, 2800, 6100, 4200, 3900, 2600][seat - 1] || initialBuyIn,
          currentBet: 0,
          cards: [],
          hasFolded: false,
          isAllIn: false,
          hasActed: false,
        });
      }
    });
  }

  resolvedPlayers.sort((a, b) => a.seatNumber - b.seatNumber);

  const [players, setPlayers] = useState<TablePlayer[]>(resolvedPlayers);
  const [stage, setStage] = useState<GameStage>(resolvedPlayers.length >= 2 ? 'DEALING' : 'WAITING');

  const [handNumber, setHandNumber] = useState<number>(1);
  const [pot, setPot] = useState<number>(0);
  const [currentRoundBet, setCurrentRoundBet] = useState<number>(0);
  const [currentTurnIdx, setCurrentTurnIdx] = useState<number>(0);
  const [dealerIdx, setDealerIdx] = useState<number>(0);
  const [communityCards, setCommunityCards] = useState<Card[]>([]);
  const [deck, setDeck] = useState<Card[]>([]);
  const [actionTimer, setActionTimer] = useState<number>(MAX_ACTION_TIME);
  const [logs, setLogs] = useState<ActionLog[]>([]);
  const [showLogs, setShowLogs] = useState<boolean>(false);
  const [soundEnabled, setSoundEnabled] = useState<boolean>(true);
  const [winnerMessage, setWinnerMessage] = useState<string | null>(null);
  const [countdownNextHand, setCountdownNextHand] = useState<number>(5);
  const [showExitModal, setShowExitModal] = useState<boolean>(false);
  const [raiseAmount, setRaiseAmount] = useState<number>(bigBlindVal * 2);

  const addLog = (text: string) => {
    const timeStr = new Date().toLocaleTimeString('pt-BR', { hour12: false });
    setLogs((prev) => [{ id: Math.random().toString(), text, timestamp: timeStr }, ...prev.slice(0, 30)]);
  };

  const triggerChipSound = () => {
    if (soundEnabled) soundFX.playChipSound();
  };

  const triggerCardSound = () => {
    if (soundEnabled) soundFX.playCardSlide();
  };

  const isProcessingAITurn = useRef(false);

  // Função para adicionar bot dinamicamente em mesa aguardando
  const handleAddBotToTable = () => {
    const currentSeats = new Set(players.map((p) => p.seatNumber));
    let freeSeat = 1;
    while (freeSeat <= 9 && currentSeats.has(freeSeat)) {
      freeSeat++;
    }
    if (freeSeat > 9) return;

    const newBot: TablePlayer = {
      id: freeSeat,
      seatNumber: freeSeat,
      name: `Bot #${freeSeat} (${defaultNames[freeSeat]?.split(' ')[0] || 'Player'})`,
      isUser: false,
      stack: initialBuyIn,
      currentBet: 0,
      cards: [],
      hasFolded: false,
      isAllIn: false,
      hasActed: false,
    };

    setPlayers((prev) => [...prev, newBot].sort((a, b) => a.seatNumber - b.seatNumber));
    addLog(`🤖 Bot #${freeSeat} entrou na mesa.`);
  };

  // 1. INICIAR UMA NOVA MÃO
  const startNewHand = () => {
    setWinnerMessage(null);
    setCommunityCards([]);
    setPot(0);
    setCurrentRoundBet(bigBlindVal);

    // Embaralhamento com Memória Física Contínua
    const shuffled = shuffleContinuousDeck(deck.length === 52 ? deck : null);
    let cardIdx = 0;

    // Novo Dealer
    const nextDealerIdx = (dealerIdx + 1) % players.length;
    setDealerIdx(nextDealerIdx);

    const sbIdx = (nextDealerIdx + 1) % players.length;
    const bbIdx = (nextDealerIdx + 2) % players.length;
    const firstToActIdx = (nextDealerIdx + 3) % players.length;

    // Distribuir 2 cartas para cada jogador sentado.
    const updatedPlayers = players.map((p, idx) => {
      let bet = 0;
      let newStack = p.stack;

      if (idx === sbIdx) {
        const sbAmt = Math.min(smallBlindVal, newStack);
        newStack -= sbAmt;
        bet = sbAmt;
      } else if (idx === bbIdx) {
        const bbAmt = Math.min(bigBlindVal, newStack);
        newStack -= bbAmt;
        bet = bbAmt;
      }

      const pCards = [shuffled[cardIdx++], shuffled[cardIdx++]];

      return {
        ...p,
        stack: newStack,
        currentBet: bet,
        cards: pCards,
        hasFolded: false,
        isAllIn: newStack === 0,
        hasActed: false,
        lastAction: idx === sbIdx ? `Small Blind ($${smallBlindVal})` : idx === bbIdx ? `Big Blind ($${bigBlindVal})` : undefined,
        isDealer: idx === nextDealerIdx,
        isSmallBlind: idx === sbIdx,
        isBigBlind: idx === bbIdx,
        handEval: undefined,
      };
    });

    const initialPot = smallBlindVal + bigBlindVal;
    setPot(initialPot);
    setDeck(shuffled.slice(cardIdx));
    setPlayers(updatedPlayers);
    setCurrentTurnIdx(firstToActIdx);
    setStage('PRE_FLOP');
    setActionTimer(MAX_ACTION_TIME);
    setRaiseAmount(bigBlindVal * 2);

    triggerCardSound();
    triggerChipSound();

    addLog(`--- Mão #${handNumber} iniciada (${updatedPlayers.length} jogadores) ---`);
    addLog(`${updatedPlayers[nextDealerIdx].name} está no Botão (Dealer).`);
    addLog(`${updatedPlayers[sbIdx].name} postou Small Blind ($${smallBlindVal}).`);
    addLog(`${updatedPlayers[bbIdx].name} postou Big Blind ($${bigBlindVal}).`);
    addLog(`Vez de agir: ${updatedPlayers[firstToActIdx].name}`);
  };

  const [isMultiplayerMode, setIsMultiplayerMode] = useState<boolean>(true);
  const [wsConnected, setWsConnected] = useState<boolean>(false);
  const [ownCards, setOwnCards] = useState<Card[]>([]);
  const [isWaitingForAction, setIsWaitingForAction] = useState<boolean>(false);
  const wsRef = useRef<WebSocket | null>(null);

  // WebSocket Integration para Multiplayer Autoritativo
  useEffect(() => {
    const wsUrl = (import.meta.env.VITE_API_BASE_URL || window.location.origin)
      .replace(/^http/, 'ws')
      + `/ws?token=${token || ''}`;

    let socket: WebSocket | null = null;
    let isMounted = true;

    try {
      socket = new WebSocket(wsUrl);
      wsRef.current = socket;

      socket.onopen = () => {
        if (!isMounted) return;
        setWsConnected(true);
        addLog('Conectado ao servidor multiplayer via WebSocket.');

        // Envia JOIN_TABLE
        const joinMsg = {
          type: 'JOIN_TABLE',
          payload: {
            table_id: tableId,
            seat_number: chosenSeat,
            buy_in: initialBuyIn,
          },
        };
        socket?.send(JSON.stringify(joinMsg));
      };

      socket.onmessage = (event) => {
        if (!isMounted) return;
        try {
          const msg = JSON.parse(event.data);

          if (msg.type === 'TABLE_STATE' && msg.payload) {
            const serverState: ServerTableState = msg.payload;
            handleTableStateUpdate(serverState);
          } else if (msg.type === 'PRIVATE_CARDS' && msg.payload) {
            const privatePayload: PrivateCardsPayload = msg.payload;
            if (privatePayload.cards && privatePayload.cards.length > 0) {
              setOwnCards(privatePayload.cards);
              triggerCardSound();
            }
          }
        } catch (err) {
          console.error('Erro ao processar mensagem WS:', err);
        }
      };

      socket.onerror = (e) => {
        console.warn('Erro na conexão WebSocket da mesa:', e);
      };

      socket.onclose = () => {
        if (!isMounted) return;
        setWsConnected(false);
        addLog('Conexão WebSocket com o servidor encerrada.');
      };
    } catch (err) {
      console.warn('Falha ao instanciar WebSocket:', err);
      setIsMultiplayerMode(false);
    }

    return () => {
      isMounted = false;
      if (socket && socket.readyState === WebSocket.OPEN) {
        socket.send(JSON.stringify({ type: 'LEAVE_TABLE' }));
        socket.close();
      }
      wsRef.current = null;
    };
  }, [tableId, chosenSeat, initialBuyIn, token]);

  const handleTableStateUpdate = (serverState: ServerTableState) => {
    setIsWaitingForAction(false);
    setStage(serverState.stage);
    setHandNumber(serverState.hand_number);
    setPot(serverState.pot);
    setCurrentRoundBet(serverState.current_round_bet);
    setCurrentTurnIdx(serverState.current_turn_idx);
    setDealerIdx(serverState.dealer_idx);
    setCommunityCards(serverState.community_cards || []);
    if (serverState.winner_message) {
      setWinnerMessage(serverState.winner_message);
      if (soundEnabled) soundFX.playWinFanfare();
    } else {
      setWinnerMessage(null);
    }

    // Mapeia jogadores do servidor para o TablePlayer local
    const mappedPlayers: TablePlayer[] = serverState.players.map((sp) => {
      const isCurrentUser = sp.user_id === user?.id || sp.seat_number === chosenSeat;
      return {
        id: sp.id,
        seatNumber: sp.seat_number,
        userId: sp.user_id,
        name: isCurrentUser ? `${user?.nome_completo || 'Você'} (VIP)` : sp.name,
        isUser: isCurrentUser,
        isBot: sp.is_bot,
        stack: sp.stack,
        currentBet: sp.current_bet,
        cards: sp.cards || [],
        hasFolded: sp.has_folded,
        isAllIn: sp.is_all_in,
        hasActed: sp.has_acted,
        lastAction: sp.last_action,
        isDealer: sp.is_dealer,
        isSmallBlind: sp.is_small_blind,
        isBigBlind: sp.is_big_blind,
        handEval: sp.hand_eval ? {
          rank: sp.hand_eval.rank,
          rankName: sp.hand_eval.rank_name,
          score: sp.hand_eval.score,
          description: sp.hand_eval.description,
        } : undefined,
      };
    });

    setPlayers(mappedPlayers);
    setActionTimer(MAX_ACTION_TIME);
  };

  useEffect(() => {
    soundFX.enabled = soundEnabled;
  }, [soundEnabled]);

  useEffect(() => {
    if (!isMultiplayerMode) {
      if (players.length >= 2) {
        startNewHand();
      } else {
        setStage('WAITING');
      }
    }
  }, [isMultiplayerMode]);

  // 2. TEMPORIZADOR DE AÇÃO (Action Timer Clock de 20s)
  useEffect(() => {
    if (stage === 'SHOWDOWN' || stage === 'HAND_OVER' || stage === 'DEALING') return;

    const interval = setInterval(() => {
      setActionTimer((prev) => {
        if (prev <= 1) {
          if (!isMultiplayerMode) {
            handleAutoTimeoutAction();
          }
          return MAX_ACTION_TIME;
        }
        return prev - 1;
      });
    }, 1000);

    return () => clearInterval(interval);
  }, [currentTurnIdx, stage, players, isMultiplayerMode]);

  const handleAutoTimeoutAction = () => {
    const activePlayer = players[currentTurnIdx];
    if (!activePlayer || activePlayer.hasFolded || activePlayer.isAllIn) return;

    if (activePlayer.currentBet >= currentRoundBet) {
      handlePlayerAction('CHECK');
    } else {
      handlePlayerAction('FOLD');
    }
  };

  // 3. FLUXO DE INTELIGÊNCIA / RESPOSTA DE BOTS
  // Em modo multiplayer, o frontend NUNCA simula ações para outros jogadores humanos
  useEffect(() => {
    if (stage === 'SHOWDOWN' || stage === 'HAND_OVER' || stage === 'DEALING') return;
    if (isMultiplayerMode) return; // No multiplayer o backend é autoritativo

    const activePlayer = players[currentTurnIdx];
    if (!activePlayer) return;

    if (activePlayer.hasFolded || activePlayer.isAllIn) {
      advanceTurn(players, currentTurnIdx);
      return;
    }

    if (activePlayer.isUser) {
      isProcessingAITurn.current = false;
      return;
    }

    if (!isProcessingAITurn.current) {
      isProcessingAITurn.current = true;
      const thinkTime = 1200 + Math.random() * 1100;

      const timer = setTimeout(() => {
        const toCall = currentRoundBet - activePlayer.currentBet;

        if (toCall === 0) {
          if (Math.random() < 0.82) {
            handlePlayerAction('CHECK');
          } else {
            const betAmt = Math.min(activePlayer.stack, bigBlindVal * 2);
            handlePlayerAction('RAISE', betAmt);
          }
        } else {
          const rand = Math.random();
          if (rand < 0.28 && toCall > bigBlindVal * 3) {
            handlePlayerAction('FOLD');
          } else if (rand < 0.88) {
            handlePlayerAction('CALL');
          } else {
            const raiseVal = Math.min(activePlayer.stack, currentRoundBet * 2);
            handlePlayerAction('RAISE', raiseVal);
          }
        }
        isProcessingAITurn.current = false;
      }, thinkTime);

      return () => clearTimeout(timer);
    }
  }, [currentTurnIdx, stage, currentRoundBet, isMultiplayerMode]);

  // 4. EXECUÇÃO DE AÇÕES DE UM JOGADOR
  const handlePlayerAction = (action: 'FOLD' | 'CHECK' | 'CALL' | 'RAISE' | 'ALL_IN', customAmount?: number) => {
    if (isMultiplayerMode && wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
      setIsWaitingForAction(true);
      const actionMsg = {
        type: 'PLAYER_ACTION',
        payload: {
          action,
          amount: action === 'RAISE' ? (customAmount || raiseAmount) : undefined,
        },
      };
      wsRef.current.send(JSON.stringify(actionMsg));
      return;
    }

    const updated = [...players];
    const player = updated[currentTurnIdx];
    if (!player || player.hasFolded || player.isAllIn) return;

    player.hasActed = true;
    let newPot = pot;
    let newRoundBet = currentRoundBet;

    switch (action) {
      case 'FOLD':
        player.hasFolded = true;
        player.lastAction = 'Desistiu (Fold)';
        triggerCardSound();
        addLog(`${player.name} deu Fold.`);
        break;

      case 'CHECK':
        player.lastAction = 'Passou a vez (Check)';
        addLog(`${player.name} deu Check.`);
        break;

      case 'CALL': {
        const needed = currentRoundBet - player.currentBet;
        const callAmt = Math.min(needed, player.stack);
        player.stack -= callAmt;
        player.currentBet += callAmt;
        newPot += callAmt;
        if (player.stack === 0) player.isAllIn = true;
        player.lastAction = `Pagou $${callAmt} (Call)`;
        triggerChipSound();
        addLog(`${player.name} deu Call de $${callAmt}.`);
        break;
      }

      case 'RAISE': {
        const targetBet = customAmount || raiseAmount;
        const additional = targetBet - player.currentBet;
        const actualAmt = Math.min(additional, player.stack);
        player.stack -= actualAmt;
        player.currentBet += actualAmt;
        newPot += actualAmt;
        newRoundBet = player.currentBet;

        updated.forEach((p, i) => {
          if (i !== currentTurnIdx && !p.hasFolded && !p.isAllIn) {
            p.hasActed = false;
          }
        });

        if (player.stack === 0) player.isAllIn = true;
        player.lastAction = `Aumentou para $${player.currentBet} (Raise)`;
        triggerChipSound();
        addLog(`${player.name} aumentou a aposta para $${player.currentBet}!`);
        break;
      }

      case 'ALL_IN': {
        const allInAmt = player.stack;
        player.currentBet += allInAmt;
        player.stack = 0;
        player.isAllIn = true;
        newPot += allInAmt;
        if (player.currentBet > newRoundBet) {
          newRoundBet = player.currentBet;
          updated.forEach((p, i) => {
            if (i !== currentTurnIdx && !p.hasFolded && !p.isAllIn) {
              p.hasActed = false;
            }
          });
        }
        player.lastAction = `ALL-IN ($${allInAmt}) 🔥`;
        triggerChipSound();
        addLog(`${player.name} foi ALL-IN com $${allInAmt}! 🔥`);
        break;
      }
    }

    setPot(newPot);
    setCurrentRoundBet(newRoundBet);
    setPlayers(updated);
    setActionTimer(MAX_ACTION_TIME);

    const activeNonFolded = updated.filter((p) => !p.hasFolded);
    if (activeNonFolded.length === 1) {
      handleSingleSurvivorWin(updated, activeNonFolded[0], newPot);
      return;
    }

    checkBettingRoundComplete(updated, newRoundBet, currentTurnIdx);
  };

  // 5. VERIFICAÇÃO DE CONCLUSÃO DA RODADA DE APOSTAS
  const checkBettingRoundComplete = (currentPlayers: TablePlayer[], roundBet: number, currentIdx: number) => {
    const activePlayers = currentPlayers.filter((p) => !p.hasFolded && !p.isAllIn);

    const isRoundDone =
      activePlayers.length === 0 ||
      activePlayers.every((p) => p.hasActed && p.currentBet === roundBet);

    if (isRoundDone) {
      advanceStreet(currentPlayers);
    } else {
      advanceTurn(currentPlayers, currentIdx);
    }
  };

  const advanceTurn = (currentPlayers: TablePlayer[], fromIdx: number) => {
    let nextIdx = (fromIdx + 1) % currentPlayers.length;
    let attempts = 0;

    while (attempts < currentPlayers.length) {
      const candidate = currentPlayers[nextIdx];
      if (!candidate.hasFolded && !candidate.isAllIn) {
        setCurrentTurnIdx(nextIdx);
        setActionTimer(MAX_ACTION_TIME);
        addLog(`Vez de agir: ${candidate.name}`);
        return;
      }
      nextIdx = (nextIdx + 1) % currentPlayers.length;
      attempts++;
    }

    advanceStreet(currentPlayers);
  };

  // 6. TRANSIÇÃO AUTOMÁTICA DE RUAS
  const advanceStreet = (currentPlayers: TablePlayer[]) => {
    const resetPlayers = currentPlayers.map((p) => ({
      ...p,
      currentBet: 0,
      hasActed: false,
    }));
    setCurrentRoundBet(0);

    let nextDeck = [...deck];
    let newCards = [...communityCards];

    triggerChipSound();
    triggerCardSound();

    if (stage === 'PRE_FLOP') {
      newCards = [nextDeck.shift()!, nextDeck.shift()!, nextDeck.shift()!];
      setCommunityCards(newCards);
      setDeck(nextDeck);
      setStage('FLOP');
      addLog(`🎰 FLOP revelado: ${newCards.map((c) => c.code).join(' ')}`);
      setFirstTurnOfStreet(resetPlayers, dealerIdx);
    } else if (stage === 'FLOP') {
      const turnCard = nextDeck.shift()!;
      newCards = [...newCards, turnCard];
      setCommunityCards(newCards);
      setDeck(nextDeck);
      setStage('TURN');
      addLog(`🎲 TURN revelado: ${turnCard.code}`);
      setFirstTurnOfStreet(resetPlayers, dealerIdx);
    } else if (stage === 'TURN') {
      const riverCard = nextDeck.shift()!;
      newCards = [...newCards, riverCard];
      setCommunityCards(newCards);
      setDeck(nextDeck);
      setStage('RIVER');
      addLog(`🌊 RIVER revelado: ${riverCard.code}`);
      setFirstTurnOfStreet(resetPlayers, dealerIdx);
    } else if (stage === 'RIVER') {
      executeShowdown(resetPlayers, newCards);
    }
  };

  const setFirstTurnOfStreet = (currentPlayers: TablePlayer[], dIdx: number) => {
    let nextIdx = (dIdx + 1) % currentPlayers.length;
    for (let i = 0; i < currentPlayers.length; i++) {
      const idx = (nextIdx + i) % currentPlayers.length;
      if (!currentPlayers[idx].hasFolded && !currentPlayers[idx].isAllIn) {
        setCurrentTurnIdx(idx);
        setActionTimer(MAX_ACTION_TIME);
        setPlayers(currentPlayers);
        return;
      }
    }
    setPlayers(currentPlayers);
  };

  // 7. SHOWDOWN E CÁLCULO DE VENCEDORES
  const executeShowdown = (currentPlayers: TablePlayer[], tableCards: Card[]) => {
    setStage('SHOWDOWN');
    addLog(`🏆 --- SHOWDOWN ---`);

    const evaluatedPlayers = currentPlayers.map((p) => {
      if (p.hasFolded) return p;
      const combined = [...p.cards, ...tableCards];
      const evaluation = evaluate7Cards(combined);
      return {
        ...p,
        handEval: evaluation,
      };
    });

    let bestScore = -1;
    let winners: TablePlayer[] = [];

    evaluatedPlayers.forEach((p) => {
      if (!p.hasFolded && p.handEval) {
        addLog(`${p.name} mostrou ${p.cards.map((c) => c.code).join(' ')}: ${p.handEval.description}`);
        if (p.handEval.score > bestScore) {
          bestScore = p.handEval.score;
          winners = [p];
        } else if (p.handEval.score === bestScore) {
          winners.push(p);
        }
      }
    });

    const prizePerWinner = Math.floor(pot / winners.length);
    const finalPlayers = evaluatedPlayers.map((p) => {
      if (winners.some((w) => w.id === p.id)) {
        return { ...p, stack: p.stack + prizePerWinner };
      }
      return p;
    });

    const winnerNames = winners.map((w) => w.name).join(', ');
    const bestHandDesc = winners[0]?.handEval?.description || 'Melhor Mão';
    const msg = `🎉 ${winnerNames} venceu o pote de $${pot} com ${bestHandDesc}!`;
    setWinnerMessage(msg);
    addLog(msg);
    setPlayers(finalPlayers);

    if (soundEnabled) soundFX.playWinFanfare();
    scheduleNextHand();
  };

  const handleSingleSurvivorWin = (currentPlayers: TablePlayer[], winner: TablePlayer, totalPot: number) => {
    setStage('SHOWDOWN');
    const finalPlayers = currentPlayers.map((p) => {
      if (p.id === winner.id) {
        return { ...p, stack: p.stack + totalPot };
      }
      return p;
    });

    const msg = `🏆 ${winner.name} venceu o pote de $${totalPot} (Todos deram Fold)!`;
    setWinnerMessage(msg);
    addLog(msg);
    setPlayers(finalPlayers);

    if (soundEnabled) soundFX.playWinFanfare();
    scheduleNextHand();
  };

  // 8. CONTAGEM REGRESSIVA PARA PRÓXIMA MÃO
  const scheduleNextHand = () => {
    setStage('HAND_OVER');
    let timeLeft = 5;
    setCountdownNextHand(timeLeft);

    const timer = setInterval(() => {
      timeLeft -= 1;
      setCountdownNextHand(timeLeft);
      if (timeLeft <= 0) {
        clearInterval(timer);
        if (!isMultiplayerMode) {
          setHandNumber((h) => h + 1);
          startNewHand();
        }
      }
    }, 1000);
  };

  const [isExiting, setIsExiting] = useState(false);

  // Saída Graciosa da Mesa com devolução (Cash-Out) de fichas
  const handleConfirmExit = async () => {
    if (isExiting) return;
    setIsExiting(true);
    const remainingChips = userPlayer ? userPlayer.stack : 0;
    try {
      if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
        wsRef.current.send(JSON.stringify({ type: 'LEAVE_TABLE' }));
      }
      if (token && remainingChips > 0) {
        const updatedUser = await api.cashOut(token, remainingChips);
        updateUser(updatedUser);
      } else if (user && remainingChips > 0) {
        updateUser({
          ...user,
          saldo_fichas: user.saldo_fichas + remainingChips,
        });
      }
      await leaveTableSeatRemote(token, tableId, chosenSeat);
    } catch (e) {
      console.warn('Erro ao processar cash-out:', e);
    }
    const dest = user?.role === 'admin_gerente' || user?.role === 'gerente' ? '/manager' : '/player';
    navigate(dest);
  };

  // Botões de Fichas Rápidas para Apostas
  const handleAddQuickChips = (amt: number) => {
    triggerChipSound();
    setRaiseAmount((prev) => {
      const maxStack = userPlayer?.stack || 5000;
      return Math.min(maxStack, prev + amt);
    });
  };

  const handleSetPotFraction = (fraction: number) => {
    triggerChipSound();
    const calculated = Math.max(currentRoundBet + bigBlindVal, Math.floor(pot * fraction));
    const maxStack = userPlayer?.stack || 5000;
    setRaiseAmount(Math.min(maxStack, calculated));
  };

  const userPlayer = players.find((p) => p.isUser);
  const isUserTurn = currentTurnIdx === players.findIndex((p) => p.isUser) && stage !== 'SHOWDOWN' && stage !== 'HAND_OVER';
  const toCallAmount = userPlayer ? Math.max(0, currentRoundBet - userPlayer.currentBet) : 0;
  const canCheck = toCallAmount === 0;

  const effectiveUserCards = (isMultiplayerMode && ownCards.length === 2)
    ? ownCards
    : (userPlayer?.cards || []);

  const userHandEval = effectiveUserCards.length === 2 && communityCards.length >= 3
    ? evaluate7Cards([...effectiveUserCards, ...communityCards])
    : null;

  // Mapeamento das 9 Posições da Mesa
  const seatPositions = [
    { top: '2%', left: '20%', transform: '-translate-x-1/2', betPos: 'bottom' as const }, // Assento 1
    { top: '2%', left: '50%', transform: '-translate-x-1/2', betPos: 'bottom' as const }, // Assento 2
    { top: '2%', right: '20%', transform: 'translate-x-1/2', betPos: 'bottom' as const },  // Assento 3
    { top: '32%', right: '2%', transform: '-translate-y-1/2', betPos: 'left' as const },    // Assento 4
    { top: '68%', right: '2%', transform: '-translate-y-1/2', betPos: 'left' as const },    // Assento 5
    { bottom: '2%', right: '32%', transform: 'translate-x-1/2', betPos: 'top' as const },  // Assento 6
    { bottom: '2%', left: '32%', transform: '-translate-x-1/2', betPos: 'top' as const },   // Assento 7
    { top: '68%', left: '2%', transform: '-translate-y-1/2', betPos: 'right' as const },    // Assento 8
    { top: '32%', left: '2%', transform: '-translate-y-1/2', betPos: 'right' as const },    // Assento 9
  ];

  return (
    <div className="min-h-screen bg-[#07090e] text-white flex flex-col justify-between p-2 sm:p-4 select-none">
      {/* 1. Header Superior da Mesa com Botão de Sair da Mesa */}
      <div className="flex items-center justify-between border-b border-[#d4af37]/30 pb-2.5">
        <div className="flex items-center space-x-3">
          <button
            onClick={() => setShowExitModal(true)}
            className="flex items-center space-x-1.5 px-3 py-1.5 rounded-lg bg-red-950/80 hover:bg-red-900 border border-red-500/60 text-red-200 hover:text-white text-xs font-extrabold uppercase tracking-wider transition cursor-pointer shadow-md"
            title="Sair da Mesa de Poker"
          >
            <LogOut size={15} />
            <span>SAIR DA MESA</span>
          </button>
          <div>
            <div className="flex items-center space-x-2">
              <h1 className="text-xs sm:text-sm font-extrabold text-[#f5d77f] uppercase tracking-wider">
                {tableName}
              </h1>
              <span className="text-[10px] bg-[#d4af37]/20 border border-[#d4af37]/50 text-[#f5d77f] font-mono px-1.5 py-0.2 rounded">
                Mão #{handNumber}
              </span>
            </div>
            <p className="text-[10px] text-zinc-400">
              Blinds: <span className="text-zinc-200 font-bold">${smallBlindVal}/${bigBlindVal}</span> • Seu Assento: #{chosenSeat} • {players.length} Jogadores • {wsConnected ? <span className="text-emerald-400 font-bold">● Online</span> : <span className="text-amber-400 font-bold">○ Conectando</span>}
            </p>
          </div>
        </div>

        {/* Status da Rua, Histórico e Mute */}
        <div className="flex items-center space-x-2">
          <div className="px-2.5 py-1 bg-zinc-900 border border-zinc-700 rounded-lg text-xs font-bold text-zinc-300 uppercase tracking-widest hidden sm:block">
            {stage === 'WAITING' && 'SALA DE ESPERA ⏳'}
            {stage === 'PRE_FLOP' && 'PRÉ-FLOP'}
            {stage === 'FLOP' && 'FLOP (3 Cartas)'}
            {stage === 'TURN' && 'TURN (4ª Carta)'}
            {stage === 'RIVER' && 'RIVER (5ª Carta)'}
            {stage === 'SHOWDOWN' && 'SHOWDOWN 🏆'}
            {stage === 'HAND_OVER' && `PRÓXIMA MÃO EM ${countdownNextHand}s`}
          </div>

          <button
            onClick={() => setShowLogs(!showLogs)}
            className={`p-2 rounded-lg border text-xs transition cursor-pointer flex items-center gap-1 ${
              showLogs ? 'bg-[#d4af37] text-black border-white' : 'bg-zinc-900 border-zinc-700 text-zinc-300'
            }`}
            title="Histórico de Jogadas"
          >
            <History size={16} />
          </button>

          <button
            onClick={() => setSoundEnabled(!soundEnabled)}
            className="p-2 rounded-lg bg-zinc-900 border border-zinc-700 text-zinc-300 hover:text-[#d4af37] transition cursor-pointer"
            title={soundEnabled ? 'Silenciar Áudio' : 'Ativar Áudio'}
          >
            {soundEnabled ? <Volume2 size={16} /> : <VolumeX size={16} />}
          </button>
        </div>
      </div>

      {/* Modal de Confirmação para Sair da Mesa */}
      {showExitModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/80 p-4">
          <div className="w-full max-w-sm bg-[#12151d] border-2 border-red-500/80 rounded-2xl p-6 text-center space-y-4 shadow-[0_0_40px_rgba(239,68,68,0.3)]">
            <div className="w-12 h-12 rounded-full bg-red-950/80 border border-red-500 flex items-center justify-center mx-auto text-red-400">
              <AlertTriangle size={24} />
            </div>

            <div className="space-y-1">
              <h3 className="text-base font-extrabold text-white uppercase tracking-wider">
                DESEJA SAIR DA MESA?
              </h3>
              <p className="text-xs text-zinc-400 leading-relaxed">
                Seu saldo de <span className="text-[#f5d77f] font-bold font-mono">${userPlayer?.stack.toLocaleString('pt-BR')}</span> fichas será preservado e você retornará ao painel principal.
              </p>
            </div>

            <div className="grid grid-cols-2 gap-3 pt-2">
              <button
                onClick={() => setShowExitModal(false)}
                className="py-2.5 px-4 rounded-xl bg-zinc-800 hover:bg-zinc-700 text-xs font-bold text-zinc-200 uppercase tracking-wider transition cursor-pointer"
              >
                CANCELAR
              </button>
              <button
                disabled={isExiting}
                onClick={handleConfirmExit}
                className="py-2.5 px-4 rounded-xl bg-gradient-to-r from-red-600 to-red-800 hover:from-red-500 hover:to-red-700 text-xs font-extrabold text-white uppercase tracking-wider transition shadow-lg cursor-pointer disabled:opacity-50"
              >
                {isExiting ? 'CASH-OUT...' : 'SIM, SAIR'}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Drawer de Histórico / Logs */}
      {showLogs && (
        <div className="bg-zinc-950/95 border border-[#d4af37]/40 rounded-xl p-3 my-2 max-h-36 overflow-y-auto text-xs space-y-1 font-mono">
          <div className="flex justify-between items-center text-[#d4af37] font-bold border-b border-zinc-800 pb-1 mb-1">
            <span>HISTÓRICO DA MESA</span>
            <button onClick={() => setShowLogs(false)} className="text-zinc-500 hover:text-white">✕</button>
          </div>
          {logs.map((log) => (
            <div key={log.id} className="text-zinc-300 leading-tight">
              <span className="text-zinc-600">[{log.timestamp}]</span> {log.text}
            </div>
          ))}
        </div>
      )}

      {/* 2. Feltro de Poker Central com 9 Jogadores e Fichas 3D Estáticas */}
      <div className="relative my-2 flex-grow flex items-center justify-center">
        {/* Mesa Oval de Feltro 9-Max */}
        <div className="w-full max-w-5xl h-[430px] sm:h-[490px] rounded-[200px] sm:rounded-[240px] poker-felt border-[14px] sm:border-[18px] border-[#181109] shadow-[0_0_50px_rgba(0,0,0,0.9),inset_0_0_60px_rgba(0,0,0,0.8)] relative flex flex-col items-center justify-center p-4">
          {/* Logo da Mesa */}
          <div className="text-center opacity-30 select-none pointer-events-none mb-1">
            <div className="text-[#d4af37] text-sm sm:text-base font-black tracking-widest uppercase">
              CLUB POKER DOS AMIGOS 9-MAX
            </div>
            <div className="text-[9px] text-zinc-300 tracking-widest">CONTINUOUS PHYSICAL SHUFFLE ENGINE</div>
          </div>

          {/* Pote Central com Pilhas de Fichas 3D Estáticas */}
          <div className="flex flex-col items-center justify-center mb-3">
            {pot > 0 && (
              <div className="mb-2">
                <ChipStack amount={pot} size="md" showLabel={false} />
              </div>
            )}
            <div className="bg-black/85 border border-[#d4af37]/80 rounded-full px-5 py-1 shadow-2xl text-center">
              <span className="text-[9px] text-zinc-400 uppercase tracking-widest block font-bold">POTE TOTAL</span>
              <span className="text-sm sm:text-lg font-black text-[#f5d77f] font-mono">${pot.toLocaleString('pt-BR')}</span>
            </div>
          </div>

          {/* Cartas Comunitárias */}
          <div className="flex items-center justify-center space-x-1.5 sm:space-x-2">
            {communityCards.map((card, idx) => (
              <CardView key={idx} card={card} />
            ))}
            {Array.from({ length: 5 - communityCards.length }).map((_, idx) => (
              <div
                key={`empty-${idx}`}
                className="w-9 h-13 sm:w-11 sm:h-15 rounded-md border-2 border-dashed border-emerald-900/60 bg-emerald-950/20"
              />
            ))}
          </div>

          {/* Banner de Vencedor / Showdown */}
          {winnerMessage && (
            <div className="absolute top-1/2 -translate-y-1/2 bg-black/95 border-2 border-[#d4af37] px-5 py-3 rounded-2xl text-center shadow-[0_0_35px_rgba(212,175,55,0.8)] z-20">
              <p className="text-xs sm:text-sm font-extrabold text-[#f5d77f] flex items-center justify-center gap-1.5">
                <Trophy size={18} className="text-yellow-400" />
                {winnerMessage}
              </p>
              <p className="text-[10px] text-zinc-400 mt-0.5">
                Próxima mão iniciando em {countdownNextHand}s...
              </p>
            </div>
          )}

          {/* Sala de Espera / Aguardando Jogadores */}
          {stage === 'WAITING' && (
            <div className="absolute top-1/2 -translate-y-1/2 bg-black/95 border-2 border-[#d4af37]/80 px-6 py-4 rounded-2xl text-center shadow-[0_0_40px_rgba(0,0,0,0.9)] z-20 space-y-3 max-w-sm">
              <div>
                <p className="text-sm font-black text-[#f5d77f] uppercase tracking-wider">
                  Mesa Criada / Aguardando
                </p>
                <p className="text-xs text-zinc-300 mt-1">
                  {players.length === 1
                    ? 'Há apenas 1 jogador na mesa. São necessários ao menos 2 jogadores para iniciar.'
                    : `${players.length} jogadores prontos para jogar.`}
                </p>
              </div>

              <div className="flex flex-col gap-2 pt-1">
                {players.length >= 2 && (
                  <button
                    type="button"
                    onClick={() => startNewHand()}
                    className="w-full py-2.5 rounded-xl gold-btn text-black font-extrabold text-xs uppercase tracking-wider cursor-pointer shadow-lg hover:scale-105 transition"
                  >
                    ▶️ INICIAR PARTIDA AGORA
                  </button>
                )}

                {players.length < 9 && (
                  <button
                    type="button"
                    onClick={handleAddBotToTable}
                    className="w-full py-2 rounded-xl bg-zinc-900 border border-[#d4af37]/60 hover:bg-zinc-800 text-[#f5d77f] font-bold text-xs uppercase tracking-wider cursor-pointer transition"
                  >
                    🤖 ADICIONAR BOT DE TREINO
                  </button>
                )}
              </div>
            </div>
          )}

          {/* Renderização dos jogadores sentados ao redor da mesa com Temporizador de 20s */}
          {players.map((p, idx) => {
            const pos = seatPositions[p.seatNumber - 1] || seatPositions[0];
            const isTurn = currentTurnIdx === idx && stage !== 'SHOWDOWN' && stage !== 'HAND_OVER';
            const showCards = stage === 'SHOWDOWN' && !p.hasFolded;

            const posStyle: React.CSSProperties = {
              position: 'absolute',
              top: pos.top,
              bottom: pos.bottom,
              left: pos.left,
              right: pos.right,
              transform: pos.transform,
            };

            return (
              <div key={p.id} style={posStyle}>
                <PlayerSeatWidget
                  player={p}
                  isActiveTurn={isTurn}
                  actionTimer={actionTimer}
                  maxActionTimer={MAX_ACTION_TIME}
                  showCards={showCards}
                  betPosition={pos.betPos}
                />
              </div>
            );
          })}
        </div>
      </div>

      {/* 3. Painel Inferior de Fichas e Ações do Jogador com Action Clock 20s */}
      <div className="bg-[#10131a] border border-[#d4af37]/40 rounded-2xl p-3 sm:p-4 max-w-4xl mx-auto w-full shadow-2xl space-y-3">
        {/* Barra de Fichas Rápidas e Proporções de Pote */}
        {isUserTurn && !userPlayer?.hasFolded && (
          <div className="flex flex-wrap items-center justify-between gap-2 border-b border-zinc-800 pb-2.5">
            <span className="text-[10px] uppercase font-bold text-[#d4af37] flex items-center gap-1">
              <span>🪙</span> FICHAS RÁPIDAS:
            </span>

            {/* Fichas Unitárias */}
            <div className="flex items-center space-x-1 sm:space-x-1.5">
              <button
                type="button"
                onClick={() => handleAddQuickChips(25)}
                className="flex items-center space-x-1 px-2 py-1 bg-zinc-900 hover:bg-zinc-800 border border-emerald-500/40 rounded-lg text-[10px] font-mono font-bold text-emerald-300 transition cursor-pointer"
              >
                <PokerChip value={25} size="sm" color="green" />
                <span>+$25</span>
              </button>
              <button
                type="button"
                onClick={() => handleAddQuickChips(50)}
                className="flex items-center space-x-1 px-2 py-1 bg-zinc-900 hover:bg-zinc-800 border border-blue-500/40 rounded-lg text-[10px] font-mono font-bold text-blue-300 transition cursor-pointer"
              >
                <PokerChip value={50} size="sm" color="blue" />
                <span>+$50</span>
              </button>
              <button
                type="button"
                onClick={() => handleAddQuickChips(100)}
                className="flex items-center space-x-1 px-2 py-1 bg-zinc-900 hover:bg-zinc-800 border border-yellow-500/40 rounded-lg text-[10px] font-mono font-bold text-amber-300 transition cursor-pointer"
              >
                <PokerChip value={100} size="sm" color="black" />
                <span>+$100</span>
              </button>
              <button
                type="button"
                onClick={() => handleAddQuickChips(500)}
                className="flex items-center space-x-1 px-2 py-1 bg-zinc-900 hover:bg-zinc-800 border border-purple-500/40 rounded-lg text-[10px] font-mono font-bold text-purple-300 transition cursor-pointer"
              >
                <PokerChip value={500} size="sm" color="purple" />
                <span>+$500</span>
              </button>
            </div>

            {/* Frações do Pote */}
            <div className="flex items-center space-x-1 text-[10px] font-bold">
              <button
                type="button"
                onClick={() => handleSetPotFraction(0.5)}
                className="px-2 py-1 rounded bg-zinc-800 hover:bg-zinc-700 text-zinc-300 border border-zinc-700 cursor-pointer"
              >
                ½ POT
              </button>
              <button
                type="button"
                onClick={() => handleSetPotFraction(1.0)}
                className="px-2 py-1 rounded bg-zinc-800 hover:bg-zinc-700 text-zinc-300 border border-zinc-700 cursor-pointer"
              >
                POT
              </button>
              <button
                type="button"
                onClick={() => setRaiseAmount(userPlayer?.stack || 5000)}
                className="px-2 py-1 rounded bg-amber-950 hover:bg-amber-900 text-amber-300 border border-amber-600 cursor-pointer"
              >
                MAX
              </button>
            </div>
          </div>
        )}

        {/* Informações do Jogador e Botoeira de Ação */}
        <div className="flex flex-col sm:flex-row items-center justify-between gap-3">
          {/* Suas Cartas e Informações de Jogo */}
          <div className="flex items-center space-x-3 sm:space-x-4 w-full sm:w-auto">
            {/* Suas 2 Cartas Fechadas */}
            <div className="flex space-x-1.5 flex-shrink-0">
              {effectiveUserCards && effectiveUserCards.length === 2 ? (
                <>
                  <CardView card={effectiveUserCards[0]} />
                  <CardView card={effectiveUserCards[1]} />
                </>
              ) : (
                <>
                  <CardView hidden />
                  <CardView hidden />
                </>
              )}
            </div>

            {/* Nome, Stack com Ficha e Leitura da Mão */}
            <div className="flex-grow">
              <div className="flex items-center space-x-2">
                <span className="font-extrabold text-xs sm:text-sm text-[#f5d77f]">
                  {userPlayer?.name} (Assento #{chosenSeat})
                </span>
                {isUserTurn ? (
                  <span className="text-[9px] bg-emerald-950 border border-emerald-500 text-emerald-300 font-extrabold px-1.5 py-0.5 rounded animate-pulse">
                    SUA VEZ ({actionTimer}s)
                  </span>
                ) : (
                  <span className="text-[9px] bg-zinc-800 text-zinc-400 font-bold px-1.5 py-0.5 rounded">
                    {players[currentTurnIdx]?.name} está agindo...
                  </span>
                )}
              </div>

              {/* Stack Visual com Mini Ficha */}
              <div className="flex items-center space-x-2 mt-0.5">
                <div className="flex items-center space-x-1 text-xs text-zinc-300 font-mono">
                  <PokerChip value={100} size="sm" color="gold" />
                  <span>
                    Stack: <strong className="text-white">${userPlayer?.stack.toLocaleString('pt-BR') || 0}</strong>
                  </span>
                </div>
                {userPlayer?.currentBet ? (
                  <span className="text-[10px] text-amber-300 font-mono">
                    (Aposta: ${userPlayer.currentBet})
                  </span>
                ) : null}
              </div>

              {userHandEval && (
                <p className="text-[11px] text-[#f5d77f] font-semibold flex items-center gap-1 mt-0.5">
                  <Sparkles size={12} className="text-yellow-400" />
                  {userHandEval.description}
                </p>
              )}
            </div>
          </div>

          {/* Botoeira de Ação Interativa */}
          <div className="flex flex-wrap items-center justify-end gap-2 w-full sm:w-auto">
            {/* FOLD */}
            <button
              disabled={!isUserTurn || userPlayer?.hasFolded || isWaitingForAction}
              onClick={() => handlePlayerAction('FOLD')}
              className={`px-3.5 py-2 sm:py-2.5 rounded-lg font-bold text-xs uppercase tracking-wider transition cursor-pointer ${
                isUserTurn && !userPlayer?.hasFolded && !isWaitingForAction
                  ? 'bg-red-950 hover:bg-red-900 border border-red-500 text-red-200 shadow-md'
                  : 'bg-zinc-900 border border-zinc-800 text-zinc-600 cursor-not-allowed'
              }`}
            >
              FOLD
            </button>

            {/* CHECK ou CALL */}
            {canCheck ? (
              <button
                disabled={!isUserTurn || userPlayer?.hasFolded || isWaitingForAction}
                onClick={() => handlePlayerAction('CHECK')}
                className={`px-4 py-2 sm:py-2.5 rounded-lg font-bold text-xs uppercase tracking-wider transition cursor-pointer ${
                  isUserTurn && !userPlayer?.hasFolded && !isWaitingForAction
                    ? 'bg-blue-950 hover:bg-blue-900 border border-blue-500 text-blue-200 shadow-md'
                    : 'bg-zinc-900 border border-zinc-800 text-zinc-600 cursor-not-allowed'
                }`}
              >
                CHECK (MESA)
              </button>
            ) : (
              <button
                disabled={!isUserTurn || userPlayer?.hasFolded || isWaitingForAction}
                onClick={() => handlePlayerAction('CALL')}
                className={`px-4 py-2 sm:py-2.5 rounded-lg font-bold text-xs uppercase tracking-wider transition cursor-pointer ${
                  isUserTurn && !userPlayer?.hasFolded && !isWaitingForAction
                    ? 'bg-emerald-950 hover:bg-emerald-900 border border-emerald-500 text-emerald-200 shadow-md'
                    : 'bg-zinc-900 border border-zinc-800 text-zinc-600 cursor-not-allowed'
                }`}
              >
                CALL (${toCallAmount})
              </button>
            )}

            {/* RAISE com Seletor e Ficha */}
            <div className="flex items-center space-x-1.5">
              <input
                type="number"
                disabled={!isUserTurn || userPlayer?.hasFolded || isWaitingForAction}
                value={raiseAmount}
                onChange={(e) => setRaiseAmount(Number(e.target.value))}
                min={currentRoundBet + bigBlindVal}
                max={userPlayer?.stack || 5000}
                className="w-16 sm:w-20 bg-zinc-900 border border-[#d4af37]/50 rounded-lg px-2 py-1.5 text-xs text-center font-mono text-white disabled:opacity-40"
              />
              <button
                disabled={!isUserTurn || userPlayer?.hasFolded || (userPlayer?.stack || 0) < raiseAmount || isWaitingForAction}
                onClick={() => handlePlayerAction('RAISE')}
                className={`px-4 py-2 sm:py-2.5 rounded-lg font-extrabold text-xs uppercase tracking-wider transition cursor-pointer flex items-center gap-1 ${
                  isUserTurn && !userPlayer?.hasFolded && !isWaitingForAction
                    ? 'gold-btn text-black'
                    : 'bg-zinc-900 border border-zinc-800 text-zinc-600 cursor-not-allowed'
                }`}
              >
                <Plus size={14} />
                <span>RAISE</span>
              </button>
            </div>

            {/* ALL-IN */}
            <button
              disabled={!isUserTurn || userPlayer?.hasFolded || (userPlayer?.stack || 0) === 0 || isWaitingForAction}
              onClick={() => handlePlayerAction('ALL_IN')}
              className={`px-3 py-2 sm:py-2.5 rounded-lg font-black text-xs uppercase tracking-wider transition cursor-pointer flex items-center gap-1 ${
                isUserTurn && !userPlayer?.hasFolded && !isWaitingForAction
                  ? 'bg-amber-600 hover:bg-amber-500 text-black shadow-lg'
                  : 'bg-zinc-900 border border-zinc-800 text-zinc-600 cursor-not-allowed'
              }`}
            >
              <Flame size={14} />
              <span>ALL-IN</span>
            </button>
          </div>
        </div>
      </div>
    </div>
  );
};

// Subcomponente de Jogador Sentado na Mesa com Barra de Tempo Progressiva (20s)
interface PlayerSeatWidgetProps {
  player: TablePlayer;
  isActiveTurn: boolean;
  actionTimer: number;
  maxActionTimer: number;
  showCards: boolean;
  betPosition: 'bottom' | 'top' | 'left' | 'right';
}

const PlayerSeatWidget: React.FC<PlayerSeatWidgetProps> = ({
  player,
  isActiveTurn,
  actionTimer,
  maxActionTimer,
  showCards,
  betPosition,
}) => {
  const timerPercentage = (actionTimer / maxActionTimer) * 100;

  return (
    <div className={`relative flex flex-col items-center select-none transition-all ${player.hasFolded ? 'opacity-40' : ''}`}>
      {/* Botões Especiais (Dealer / Blinds) */}
      <div className="flex space-x-1 mb-0.5">
        {player.isDealer && (
          <span className="text-[9px] bg-yellow-400 text-black font-extrabold px-1.5 rounded-full shadow">D</span>
        )}
        {player.isSmallBlind && (
          <span className="text-[9px] bg-blue-600 text-white font-extrabold px-1.5 rounded-full shadow">SB</span>
        )}
        {player.isBigBlind && (
          <span className="text-[9px] bg-red-600 text-white font-extrabold px-1.5 rounded-full shadow">BB</span>
        )}
      </div>

      {/* Caixa do Jogador */}
      <div
        className={`relative bg-black/90 rounded-xl px-2 py-1 text-center min-w-[78px] sm:min-w-[94px] border transition shadow-xl ${
          isActiveTurn
            ? 'border-yellow-400 ring-2 ring-yellow-400/60 shadow-[0_0_20px_rgba(234,179,8,0.5)] scale-105'
            : 'border-[#d4af37]/40'
        }`}
      >
        {/* Barra de Tempo Animada Progressiva (20s) */}
        {isActiveTurn && (
          <div className="absolute -top-1 left-0 right-0 h-1 bg-zinc-800 rounded-full overflow-hidden">
            <div
              className={`h-full transition-all duration-1000 ${
                actionTimer <= 5 ? 'bg-red-500 animate-pulse' : 'bg-yellow-400'
              }`}
              style={{ width: `${timerPercentage}%` }}
            />
          </div>
        )}

        <p className="text-[10px] sm:text-[11px] font-bold text-white truncate max-w-[85px]">{player.name}</p>
        <div className="flex items-center justify-center space-x-1 mt-0.5">
          <PokerChip value={100} size="sm" color="gold" />
          <p className="text-[10px] sm:text-[11px] text-[#f5d77f] font-mono font-extrabold">${player.stack.toLocaleString('pt-BR')}</p>
        </div>

        {/* Última Ação do Jogador */}
        {player.lastAction && (
          <span className="text-[8px] sm:text-[9px] text-zinc-300 block truncate mt-0.5 font-medium">
            {player.lastAction}
          </span>
        )}
      </div>

      {/* Cartas do Jogador */}
      <div className="flex space-x-1 mt-0.5">
        {showCards && player.cards.length === 2 ? (
          <>
            <CardView card={player.cards[0]} />
            <CardView card={player.cards[1]} />
          </>
        ) : !player.hasFolded ? (
          <>
            <div className="w-4 h-6 sm:w-5 sm:h-7 rounded bg-gradient-to-br from-red-900 to-amber-950 border border-[#d4af37]/60 shadow" />
            <div className="w-4 h-6 sm:w-5 sm:h-7 rounded bg-gradient-to-br from-red-900 to-amber-950 border border-[#d4af37]/60 shadow" />
          </>
        ) : (
          <span className="text-[8px] sm:text-[9px] text-red-400 font-extrabold uppercase">FOLDED</span>
        )}
      </div>

      {/* Resposta Visual das Fichas Apostadas na Mesa */}
      {player.currentBet > 0 && (
        <div
          className={`absolute pointer-events-none z-10 ${
            betPosition === 'bottom'
              ? 'top-[112%] left-1/2 -translate-x-1/2'
              : betPosition === 'top'
              ? 'bottom-[112%] left-1/2 -translate-x-1/2'
              : betPosition === 'left'
              ? 'right-[112%] top-1/2 -translate-y-1/2'
              : 'left-[112%] top-1/2 -translate-y-1/2'
          }`}
        >
          <ChipStack amount={player.currentBet} size="sm" showLabel={true} />
        </div>
      )}
    </div>
  );
};

// Componente Visual de Carta de Baralho
interface CardViewProps {
  card?: Card;
  hidden?: boolean;
}

const CardView: React.FC<CardViewProps> = ({ card, hidden }) => {
  if (hidden || !card) {
    return (
      <div className="w-8 h-12 sm:w-10 sm:h-14 rounded-md bg-gradient-to-br from-red-900 to-amber-950 border border-[#d4af37]/60 flex items-center justify-center shadow-md select-none">
        <span className="text-[#d4af37] text-xs font-bold">♠</span>
      </div>
    );
  }

  const isRed = card.suit === 'H' || card.suit === 'D';
  const suitSymbol = card.suit === 'H' ? '♥' : card.suit === 'D' ? '♦' : card.suit === 'S' ? '♠' : '♣';
  const valStr =
    card.value === 14
      ? 'A'
      : card.value === 13
      ? 'K'
      : card.value === 12
      ? 'Q'
      : card.value === 11
      ? 'J'
      : card.value === 10
      ? '10'
      : card.value;

  return (
    <div className="w-8 h-12 sm:w-10 sm:h-14 rounded-md bg-white border border-zinc-400 flex flex-col justify-between p-1 shadow-lg text-black select-none font-bold">
      <div className={`text-[10px] sm:text-[11px] leading-none ${isRed ? 'text-red-600' : 'text-zinc-950'}`}>
        {valStr}
      </div>
      <div className={`text-xs sm:text-sm text-center leading-none ${isRed ? 'text-red-600' : 'text-zinc-950'}`}>
        {suitSymbol}
      </div>
      <div className={`text-[10px] sm:text-[11px] text-right leading-none ${isRed ? 'text-red-600' : 'text-zinc-950'}`}>
        {valStr}
      </div>
    </div>
  );
};

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
  Coins,
} from 'lucide-react';
import { useAuth } from '../context/AuthContext';
import { api } from '../services/api';
import type { Card, ServerTableState, PrivateCardsPayload } from '../types';
import {
  evaluate7Cards,
} from '../utils/pokerEngine';
import type { HandEvaluation } from '../utils/pokerEngine';
import { ChipStack, PokerChip } from '../components/poker/PokerChip';
import { soundFX } from '../utils/audio';
import { getTableRoomById, leaveTableSeatRemote } from '../utils/tableRooms';

type GameStage = 'WAITING' | 'DEALING' | 'PRE_FLOP' | 'FLOP' | 'TURN' | 'RIVER' | 'SHOWDOWN' | 'HAND_OVER';

interface TablePlayer {
  id: number;
  seatNumber: number;
  userId?: string;
  name: string;
  isUser: boolean;
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

const normalizeTableState = (payload: unknown): ServerTableState | null => {
  if (!payload || typeof payload !== 'object') return null;
  const state = payload as Partial<ServerTableState>;
  if (!Array.isArray(state.players)) return null;

  const players = state.players.filter(
    (player): player is ServerTableState['players'][number] =>
      Boolean(player) &&
      typeof player === 'object' &&
      Number.isInteger(player.seat_number) &&
      player.seat_number >= 1 &&
      typeof player.name === 'string'
  );
  if (players.length === 0) return null;

  const currentTurnIdx =
    typeof state.current_turn_idx === 'number' && Number.isInteger(state.current_turn_idx)
      ? state.current_turn_idx
      : 0;
  const handNumber = typeof state.hand_number === 'number' && Number.isFinite(state.hand_number) ? state.hand_number : 1;
  const pot = typeof state.pot === 'number' && Number.isFinite(state.pot) ? state.pot : 0;
  const currentRoundBet =
    typeof state.current_round_bet === 'number' && Number.isFinite(state.current_round_bet)
      ? state.current_round_bet
      : 0;
  const dealerIdx = typeof state.dealer_idx === 'number' && Number.isInteger(state.dealer_idx) ? state.dealer_idx : 0;
  return {
    ...(state as ServerTableState),
    stage: state.stage || 'WAITING',
    hand_number: handNumber,
    pot,
    current_round_bet: currentRoundBet,
    current_turn_idx: Math.max(0, Math.min(currentTurnIdx, players.length - 1)),
    dealer_idx: dealerIdx,
    community_cards: Array.isArray(state.community_cards) ? state.community_cards : [],
    players,
  };
};

export const PokerTablePage: React.FC = () => {
  const { user, token, updateUser, refreshUser } = useAuth();
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();

  // Parâmetros de entrada
  const tableId = searchParams.get('tableId') || 'mesa-vip-01';
  const room = getTableRoomById(tableId);
  const chosenSeat = Number(searchParams.get('seat')) || 3;
  const initialBuyIn = Number(searchParams.get('buyIn')) || 2500;

  // Configuração da Mesa (9-Max)
  const tableName = `${room?.name || "Mesa VIP Ouro #01 (Texas Hold'em)"} (9-Max)`;
  const smallBlindVal = room?.smallBlind || 25;
  const bigBlindVal = room?.bigBlind || 50;
  const MAX_ACTION_TIME = 20; // Tempo máximo de aposta aumentado para 20 segundos

  const initialPlayers: TablePlayer[] = [{
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
  }];

  const [players, setPlayers] = useState<TablePlayer[]>(initialPlayers);
  const [stage, setStage] = useState<GameStage>('WAITING');

  const [handNumber, setHandNumber] = useState<number>(1);
  const [pot, setPot] = useState<number>(0);
  const [currentRoundBet, setCurrentRoundBet] = useState<number>(0);
  const [currentTurnIdx, setCurrentTurnIdx] = useState<number>(0);
  const [communityCards, setCommunityCards] = useState<Card[]>([]);
  const [actionTimer, setActionTimer] = useState<number>(MAX_ACTION_TIME);
  const [logs, setLogs] = useState<ActionLog[]>([]);
  const [showLogs, setShowLogs] = useState<boolean>(false);
  const [soundEnabled, setSoundEnabled] = useState<boolean>(true);
  const [winnerMessage, setWinnerMessage] = useState<string | null>(null);
  const [showExitModal, setShowExitModal] = useState<boolean>(false);
  const [rebuyAmount, setRebuyAmount] = useState<number>(room?.buyInMin || 1000);
  const [isRebuying, setIsRebuying] = useState(false);
  const [rebuyError, setRebuyError] = useState<string | null>(null);
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

  const [wsConnected, setWsConnected] = useState<boolean>(false);
  const [ownCards, setOwnCards] = useState<Card[]>([]);
  const [isWaitingForAction, setIsWaitingForAction] = useState<boolean>(false);
  const wsRef = useRef<WebSocket | null>(null);
  const previousServerTurnRef = useRef<number | null>(null);
  const previousServerStageRef = useRef<GameStage | null>(null);

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
            const serverState = normalizeTableState(msg.payload);
            if (serverState) {
              handleTableStateUpdate(serverState);
            }
          } else if (msg.type === 'ERROR' && msg.payload) {
            const errorPayload = typeof msg.payload === 'string' ? JSON.parse(msg.payload) : msg.payload;
            addLog(`Erro na mesa: ${errorPayload.message || 'Não foi possível entrar na mesa.'}`);
            setIsWaitingForAction(false);
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
    if (!Array.isArray(serverState.players)) return;

    setIsWaitingForAction(false);
    setStage(serverState.stage);
    setHandNumber(serverState.hand_number);
    setPot(serverState.pot);
    setCurrentRoundBet(serverState.current_round_bet);
    setCurrentTurnIdx(serverState.current_turn_idx);
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
        stack: Number.isFinite(sp.stack) ? sp.stack : 0,
        currentBet: Number.isFinite(sp.current_bet) ? sp.current_bet : 0,
        cards: Array.isArray(sp.cards) ? sp.cards : [],
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
    if (mappedPlayers.length === 0) {
      setCurrentTurnIdx(0);
    } else if (serverState.current_turn_idx >= mappedPlayers.length) {
      setCurrentTurnIdx(0);
    }
    setActionTimer((previous) => {
      const turnChanged = previousServerTurnRef.current !== serverState.current_turn_idx;
      const stageChanged = previousServerStageRef.current !== serverState.stage;
      previousServerTurnRef.current = serverState.current_turn_idx;
      previousServerStageRef.current = serverState.stage;
      return turnChanged || stageChanged ? MAX_ACTION_TIME : previous;
    });
  };

  useEffect(() => {
    soundFX.enabled = soundEnabled;
  }, [soundEnabled]);

  const handleStartTable = () => {
    if (players.length < 2) return;
    if (wsRef.current?.readyState !== WebSocket.OPEN) {
      addLog('Aguarde a conexão com o servidor para iniciar a mesa.');
      return;
    }
    wsRef.current.send(JSON.stringify({ type: 'START_TABLE' }));
  };

  // 2. TEMPORIZADOR DE AÇÃO (Action Timer Clock de 20s)
  useEffect(() => {
    if (stage === 'WAITING' || stage === 'SHOWDOWN' || stage === 'HAND_OVER' || stage === 'DEALING') return;

    const interval = setInterval(() => {
      setActionTimer((prev) => {
        if (prev <= 1) {
          return 0;
        }
        return prev - 1;
      });
    }, 1000);

    return () => clearInterval(interval);
  }, [currentTurnIdx, stage]);

  // 3. Envia ações ao servidor autoritativo.
  const handlePlayerAction = (action: 'FOLD' | 'CHECK' | 'CALL' | 'RAISE' | 'ALL_IN', customAmount?: number) => {
    if (wsRef.current?.readyState !== WebSocket.OPEN) {
      addLog('Conexão com o servidor indisponível; ação não enviada.');
      return;
    }
    setIsWaitingForAction(true);
    wsRef.current.send(JSON.stringify({
      type: 'PLAYER_ACTION',
      payload: {
        action,
        amount: action === 'RAISE' ? (customAmount || raiseAmount) : undefined,
      },
    }));
  };

  const [isExiting, setIsExiting] = useState(false);

  // Saída Graciosa da Mesa com devolução (Cash-Out) de fichas
  const handleConfirmExit = async (destination?: string) => {
    if (isExiting) return;
    setIsExiting(true);
    try {
      await leaveTableSeatRemote(token, tableId, chosenSeat);
      if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
        wsRef.current.send(JSON.stringify({ type: 'LEAVE_TABLE' }));
      }
      await refreshUser();
    } catch (e) {
      console.warn('Erro ao processar cash-out:', e);
    }
    const dest = destination || (user?.role === 'admin_gerente' || user?.role === 'gerente' ? '/manager' : '/player');
    navigate(dest);
  };

  const handleRebuy = async () => {
    if (!token || isRebuying) return;
    setIsRebuying(true);
    setRebuyError(null);
    try {
      const updatedUser = await api.rebuy(token, rebuyAmount, tableId);
      updateUser(updatedUser);
    } catch (error) {
      setRebuyError(error instanceof Error ? error.message : 'Falha ao recarregar fichas.');
    } finally {
      setIsRebuying(false);
    }
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
  const isUserTurn =
    currentTurnIdx === players.findIndex((p) => p.isUser) &&
    stage !== 'WAITING' &&
    stage !== 'DEALING' &&
    stage !== 'SHOWDOWN' &&
    stage !== 'HAND_OVER';
  const availableRebuy = Math.min(room?.buyInMax || 5000, user?.saldo_fichas || 0);
  const minimumRebuy = room?.buyInMin || 1000;
  const requiresRebuyDecision = Boolean(
    userPlayer &&
    userPlayer.stack <= 0 &&
    (stage === 'SHOWDOWN' || stage === 'HAND_OVER' || stage === 'WAITING')
  );
  const toCallAmount = userPlayer ? Math.max(0, currentRoundBet - userPlayer.currentBet) : 0;
  const canCheck = toCallAmount === 0;

  const effectiveUserCards = ownCards.length === 2 ? ownCards : (userPlayer?.cards || []);

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
            {stage === 'HAND_OVER' && 'AGUARDANDO PRÓXIMA MÃO'}
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
                Seu saldo de <span className="text-[#f5d77f] font-bold font-mono">${userPlayer?.stack?.toLocaleString('pt-BR') || 0}</span> fichas será preservado e você retornará ao painel principal.
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
                onClick={() => void handleConfirmExit()}
                className="py-2.5 px-4 rounded-xl bg-gradient-to-r from-red-600 to-red-800 hover:from-red-500 hover:to-red-700 text-xs font-extrabold text-white uppercase tracking-wider transition shadow-lg cursor-pointer disabled:opacity-50"
              >
                {isExiting ? 'CASH-OUT...' : 'SIM, SAIR'}
              </button>
            </div>
          </div>
        </div>
      )}

      {requiresRebuyDecision && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/90 p-4">
          <div className="w-full max-w-md rounded-2xl border-2 border-amber-400/80 bg-[#12151d] p-6 text-center shadow-[0_0_40px_rgba(245,158,11,0.25)] space-y-4">
            <div className="mx-auto flex h-12 w-12 items-center justify-center rounded-full border border-amber-400 bg-amber-950/70 text-amber-300">
              <Coins size={24} />
            </div>
            <div className="space-y-1">
              <h3 className="text-base font-extrabold uppercase tracking-wider text-white">Você ficou sem fichas</h3>
              <p className="text-xs leading-relaxed text-zinc-300">
                Recarregue usando seu saldo fora da mesa ou saia agora. A próxima mão aguarda sua decisão.
              </p>
            </div>
            <label className="block space-y-1 text-left text-xs font-bold text-zinc-300">
              Valor da recarga
              <input
                type="number"
                min={minimumRebuy}
                max={availableRebuy}
                step={1}
                value={rebuyAmount}
                onChange={(event) => setRebuyAmount(Number(event.target.value))}
                className="w-full rounded-lg border border-zinc-700 bg-zinc-950 px-3 py-2 text-sm text-white outline-none focus:border-[#d4af37]"
              />
              <span className="block text-[11px] font-normal text-zinc-400">
                Disponível fora da mesa: {user?.saldo_fichas?.toLocaleString('pt-BR') || 0} fichas • Limite da mesa: {room?.buyInMax || 5000}
              </span>
            </label>
            {rebuyError && <p className="text-xs text-red-300">{rebuyError}</p>}
            <div className="grid grid-cols-1 gap-2 sm:grid-cols-3">
              <button
                disabled={isRebuying || rebuyAmount < minimumRebuy || rebuyAmount > availableRebuy}
                onClick={() => void handleRebuy()}
                className="rounded-xl gold-btn px-3 py-2.5 text-xs font-extrabold uppercase text-black disabled:cursor-not-allowed disabled:opacity-40"
              >
                {isRebuying ? 'RECARREGANDO...' : 'RECARREGAR'}
              </button>
              <button
                disabled={isExiting}
                onClick={() => void handleConfirmExit('/wallet')}
                className="rounded-xl border border-emerald-500 bg-emerald-950 px-3 py-2.5 text-xs font-extrabold uppercase text-emerald-200 hover:bg-emerald-900 disabled:opacity-50"
              >
                COMPRAR FICHAS
              </button>
              <button
                disabled={isExiting}
                onClick={() => void handleConfirmExit()}
                className="rounded-xl border border-red-500 bg-red-950 px-3 py-2.5 text-xs font-extrabold uppercase text-red-200 hover:bg-red-900 disabled:opacity-50"
              >
                SAIR DA MESA
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
                A próxima mão será iniciada pelo servidor.
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
                  {players.length < 2
                    ? 'Aguardando a entrada de pelo menos mais um jogador para iniciar.'
                    : `${players.length} jogadores prontos. Inicie a mesa quando todos estiverem preparados.`}
                </p>
              </div>

              <div className="flex flex-col gap-2 pt-1">
                {players.length >= 2 && (
                  <button
                    type="button"
                    onClick={handleStartTable}
                    disabled={!wsConnected}
                    className="w-full py-2.5 rounded-xl gold-btn text-black font-extrabold text-xs uppercase tracking-wider cursor-pointer shadow-lg hover:scale-105 transition"
                  >
                    ▶️ INICIAR MESA
                  </button>
                )}

              </div>
            </div>
          )}

          {/* Renderização dos jogadores sentados ao redor da mesa com Temporizador de 20s */}
          {players.map((p, idx) => {
            const pos = seatPositions[p.seatNumber - 1] || seatPositions[0];
            const isTurn =
              currentTurnIdx === idx &&
              stage !== 'WAITING' &&
              stage !== 'DEALING' &&
              stage !== 'SHOWDOWN' &&
              stage !== 'HAND_OVER';
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

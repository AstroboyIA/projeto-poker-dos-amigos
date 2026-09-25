import React, { useMemo, useState, useEffect, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import { ArrowLeft, Bot, CheckCircle2, Coins, Lock, Plus, ShieldCheck, Unlock, Users, X, RefreshCw } from 'lucide-react';
import { HeaderLogo } from '../components/common/HeaderLogo';
import { useAuth } from '../context/AuthContext';
import {
  countAvailableSeats,
  createRemoteTable,
  fetchRemoteTables,
  getCachedTableRooms,
  pokerTableToRoom,
  saveCachedTableRooms,
  type SeatMode,
  type TableRoom,
} from '../utils/tableRooms';
import type { PokerTable } from '../types';

const seats = Array.from({ length: 9 }, (_, idx) => idx + 1);

export const TableLobbyPage: React.FC = () => {
  const { user, token } = useAuth();
  const navigate = useNavigate();
  const [tables, setTables] = useState<TableRoom[]>(() => getCachedTableRooms());
  const [loadingTables, setLoadingTables] = useState(false);
  const [showCreate, setShowCreate] = useState(false);
  const [lockedTable, setLockedTable] = useState<TableRoom | null>(null);
  const [passwordAttempt, setPasswordAttempt] = useState('');
  const [passwordError, setPasswordError] = useState('');
  const tablesRequestVersion = React.useRef(0);

  const loadTables = useCallback(async () => {
    const requestVersion = ++tablesRequestVersion.current;
    setLoadingTables(true);
    try {
      const remote = await fetchRemoteTables(token);
      if (requestVersion !== tablesRequestVersion.current) return;
      setTables(remote);
    } finally {
      setLoadingTables(false);
    }
  }, [token]);

  useEffect(() => {
    loadTables();
  }, [loadTables]);

  // WebSocket para sincronizar criação/alteração de mesas em tempo real
  useEffect(() => {
    const wsUrl = (import.meta.env.VITE_API_BASE_URL || window.location.origin)
      .replace(/^http/, 'ws')
      + `/ws?token=${token || ''}`;

    let socket: WebSocket | null = null;
    try {
      socket = new WebSocket(wsUrl);
      socket.onopen = () => {
        void loadTables();
      };
      socket.onmessage = (event) => {
        try {
          const msg = JSON.parse(event.data);
          if (msg.type === 'TABLES_UPDATED' && msg.payload) {
            const rawTables: PokerTable[] = JSON.parse(
              typeof msg.payload === 'string' ? msg.payload : JSON.stringify(msg.payload)
            );
            const rooms = rawTables.map(pokerTableToRoom);
            tablesRequestVersion.current++;
            saveCachedTableRooms(rooms);
            setTables(rooms);
          }
        } catch {
          // Ignora mensagens de chat ou outros tipos
        }
      };
    } catch {
      // Ignora erro de conexão WS se offline
    }

    return () => {
      if (socket) socket.close();
    };
  }, [loadTables, token]);

  const visibleTables = useMemo(
    () => tables.filter((table) => countAvailableSeats(table) > 0),
    [tables]
  );

  const enterTable = (table: TableRoom) => {
    navigate(`/table/select-seat?tableId=${table.id}`);
  };

  const handleJoinTable = (table: TableRoom) => {
    if (table.password) {
      setLockedTable(table);
      setPasswordAttempt('');
      setPasswordError('');
      return;
    }

    enterTable(table);
  };

  const handleConfirmPassword = () => {
    if (!lockedTable) return;
    if (passwordAttempt !== lockedTable.password) {
      setPasswordError('Senha incorreta para esta mesa.');
      return;
    }

    enterTable(lockedTable);
  };

  const handleCreated = (table: TableRoom) => {
    setShowCreate(false);
    navigate(`/table/select-seat?tableId=${table.id}`);
  };

  return (
    <div className="min-h-screen bg-[#07090e] text-white p-4 sm:p-6 max-w-6xl mx-auto space-y-5">
      <div className="flex items-center justify-between border-b border-[#d4af37]/30 pb-3">
        <button
          onClick={() => navigate(-1)}
          className="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-zinc-900 border border-[#d4af37]/40 hover:bg-zinc-800 text-[#d4af37] text-xs font-bold transition cursor-pointer"
        >
          <ArrowLeft size={16} />
          <span>VOLTAR</span>
        </button>

        <div className="flex items-center gap-2">
          <button
            onClick={() => loadTables()}
            disabled={loadingTables}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-zinc-900 border border-[#d4af37]/40 hover:bg-zinc-800 text-[#d4af37] text-xs font-bold transition cursor-pointer"
            title="Atualizar Mesas"
          >
            <RefreshCw size={14} className={loadingTables ? 'animate-spin' : ''} />
            <span>ATUALIZAR</span>
          </button>

          <button
            onClick={() => setShowCreate(true)}
            className="flex items-center gap-2 px-4 py-2 rounded-lg gold-btn text-black text-xs font-extrabold uppercase tracking-wider cursor-pointer"
          >
            <Plus size={16} />
            <span>Criar Mesa</span>
          </button>
        </div>
      </div>

      <div className="text-center space-y-2">
        <HeaderLogo />
        <h1 className="text-xl sm:text-3xl font-extrabold tracking-wider text-[#f5d77f] uppercase">
          Lobby de Mesas
        </h1>
        <p className="text-xs sm:text-sm text-zinc-400">
          Escolha uma mesa com assentos livres ou crie uma sala configurada por você.
        </p>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        {visibleTables.map((table) => {
          const availableSeats = countAvailableSeats(table);
          return (
            <div key={table.id} className="poker-card-frame rounded-xl p-4 space-y-4">
              <div className="flex items-start justify-between gap-3">
                <div className="space-y-1">
                  <div className="flex items-center gap-2">
                    <h2 className="text-base font-extrabold text-[#f5d77f] uppercase tracking-wider">{table.name}</h2>
                    {table.password ? <Lock size={15} className="text-amber-300" /> : <Unlock size={15} className="text-emerald-300" />}
                  </div>
                  <p className="text-[11px] text-zinc-400">
                    Criada por <span className="text-zinc-200 font-bold">{table.createdBy}</span>
                  </p>
                </div>
                <span className="rounded-lg bg-emerald-950/70 border border-emerald-500/50 px-2.5 py-1 text-[11px] font-extrabold text-emerald-200">
                  {availableSeats} livres
                </span>
              </div>

              <div className="grid grid-cols-3 gap-2 text-xs">
                <InfoTile icon={<Users size={14} />} label="Jogadores" value={`${table.occupiedSeats.length}/${table.maxSeats}`} />
                <InfoTile icon={<Bot size={14} />} label="Bots" value={`${table.botSeats.length}`} />
                <InfoTile icon={<Coins size={14} />} label="Blinds" value={`$${table.smallBlind}/$${table.bigBlind}`} />
              </div>

              <div className="flex flex-wrap gap-1.5">
                {seats.map((seat) => {
                  const isOccupied = table.occupiedSeats.includes(seat);
                  const isBot = table.botSeats.includes(seat);
                  return (
                    <span
                      key={seat}
                      className={`w-8 h-8 rounded-full border flex items-center justify-center text-[10px] font-extrabold ${
                        isOccupied
                          ? 'bg-zinc-800 border-zinc-500 text-zinc-200'
                          : isBot
                            ? 'bg-[#d4af37] border-yellow-200 text-black'
                            : 'bg-emerald-950 border-emerald-500 text-emerald-200'
                      }`}
                      title={isOccupied ? 'Ocupado' : isBot ? 'Bot' : 'Livre'}
                    >
                      {isBot ? <Bot size={13} /> : seat}
                    </span>
                  );
                })}
              </div>

              <button
                onClick={() => handleJoinTable(table)}
                className="w-full py-3 rounded-xl bg-emerald-700 hover:bg-emerald-600 text-white text-xs font-extrabold uppercase tracking-widest transition cursor-pointer"
              >
                Entrar na Mesa
              </button>
            </div>
          );
        })}
      </div>

      {visibleTables.length === 0 && (
        <div className="poker-card-frame rounded-xl p-8 text-center space-y-3">
          <ShieldCheck className="w-10 h-10 text-[#d4af37] mx-auto" />
          <p className="text-sm font-bold text-zinc-200">Nenhuma mesa disponível no momento.</p>
          <button
            onClick={() => setShowCreate(true)}
            className="px-5 py-2.5 rounded-lg gold-btn text-black text-xs font-extrabold uppercase tracking-wider cursor-pointer"
          >
            Criar primeira mesa
          </button>
        </div>
      )}

      {showCreate && (
        <CreateTableModal
          onClose={() => setShowCreate(false)}
          onCreated={handleCreated}
          token={token}
          createdBy={user?.nome_completo || 'Jogador'}
        />
      )}

      {lockedTable && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/80 p-4">
          <div className="w-full max-w-sm bg-[#12151d] border border-[#d4af37]/60 rounded-xl p-5 space-y-4">
            <div className="flex items-center justify-between">
              <h3 className="text-sm font-extrabold uppercase tracking-wider text-[#f5d77f]">Mesa bloqueada</h3>
              <button onClick={() => setLockedTable(null)} className="text-zinc-400 hover:text-white cursor-pointer">
                <X size={18} />
              </button>
            </div>
            <input
              type="password"
              value={passwordAttempt}
              onChange={(event) => setPasswordAttempt(event.target.value)}
              placeholder="Senha da mesa"
              className="w-full rounded-lg bg-zinc-950 border border-zinc-700 px-3 py-2 text-sm text-white outline-none focus:border-[#d4af37]"
            />
            {passwordError && <p className="text-xs text-red-300">{passwordError}</p>}
            <button onClick={handleConfirmPassword} className="w-full py-2.5 rounded-lg gold-btn text-black text-xs font-extrabold uppercase cursor-pointer">
              Desbloquear e entrar
            </button>
          </div>
        </div>
      )}
    </div>
  );
};

const InfoTile: React.FC<{ icon: React.ReactNode; label: string; value: string }> = ({ icon, label, value }) => (
  <div className="rounded-lg bg-[#0b0e14] border border-zinc-800 p-2">
    <div className="flex items-center gap-1 text-[#d4af37]">{icon}<span className="text-[10px] uppercase font-bold">{label}</span></div>
    <p className="text-sm font-extrabold text-white mt-1">{value}</p>
  </div>
);

const CreateTableModal: React.FC<{
  onClose: () => void;
  onCreated: (table: TableRoom) => void;
  token: string | null;
  createdBy: string;
}> = ({ onClose, onCreated, token, createdBy }) => {
  const [name, setName] = useState('Nova Mesa Cash Game');
  const [smallBlind, setSmallBlind] = useState(25);
  const [bigBlind, setBigBlind] = useState(50);
  const [buyInMin, setBuyInMin] = useState(1000);
  const [buyInMax, setBuyInMax] = useState(5000);
  const [password, setPassword] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [seatModes, setSeatModes] = useState<Record<number, SeatMode>>({
    2: 'bot',
    3: 'open',
    4: 'open',
    5: 'bot',
    6: 'open',
    7: 'open',
    8: 'open',
    9: 'open',
  });

  const botSeats = seats.filter((seat) => seat !== 1 && seatModes[seat] === 'bot');
  const openSeats = seats.filter((seat) => seat !== 1 && seatModes[seat] !== 'bot');

  const handleCreate = async () => {
    if (submitting) return;
    setSubmitting(true);
    try {
      const table = await createRemoteTable(token || '', {
        name: name.trim() || 'Mesa sem nome',
        smallBlind,
        bigBlind,
        buyInMin,
        buyInMax,
        maxSeats: 9,
        botSeats,
        occupiedSeats: [],
        password: password.trim() || undefined,
        createdBy,
      });

      onCreated(table);
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/85 p-4 overflow-y-auto">
      <div className="w-full max-w-3xl bg-[#12151d] border border-[#d4af37]/60 rounded-xl p-5 space-y-4">
        <div className="flex items-center justify-between">
          <h3 className="text-sm font-extrabold uppercase tracking-wider text-[#f5d77f]">Criar nova mesa</h3>
          <button onClick={onClose} className="text-zinc-400 hover:text-white cursor-pointer">
            <X size={18} />
          </button>
        </div>

        <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
          <label className="space-y-1 text-xs font-bold text-zinc-300">
            Nome da mesa
            <input value={name} onChange={(event) => setName(event.target.value)} className="w-full rounded-lg bg-zinc-950 border border-zinc-700 px-3 py-2 text-sm text-white outline-none focus:border-[#d4af37]" />
          </label>
          <label className="space-y-1 text-xs font-bold text-zinc-300">
            Senha opcional
            <input type="password" value={password} onChange={(event) => setPassword(event.target.value)} className="w-full rounded-lg bg-zinc-950 border border-zinc-700 px-3 py-2 text-sm text-white outline-none focus:border-[#d4af37]" />
          </label>
          <NumberField label="Small blind" value={smallBlind} onChange={setSmallBlind} />
          <NumberField label="Big blind" value={bigBlind} onChange={setBigBlind} />
          <NumberField label="Buy-in minimo" value={buyInMin} onChange={setBuyInMin} />
          <NumberField label="Buy-in maximo" value={buyInMax} onChange={setBuyInMax} />
        </div>

        <div className="space-y-2">
          <div className="flex items-center justify-between gap-2">
            <p className="text-xs font-extrabold uppercase tracking-wider text-[#d4af37]">Assentos da sala</p>
            <p className="text-[11px] text-zinc-400">{openSeats.length} livres para players • {botSeats.length} bots</p>
          </div>
          <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-4 gap-2">
            {seats.map((seat) => (
              <div key={seat} className="rounded-xl border border-zinc-800 bg-[#0b0e14] p-2 space-y-2">
                <div className="flex items-center justify-between">
                  <span className="text-xs font-bold text-zinc-200">Assento #{seat}</span>
                  {seat === 1 && <CheckCircle2 size={14} className="text-emerald-300" />}
                </div>
                {seat === 1 ? (
                  <p className="text-[10px] text-emerald-300 font-bold uppercase">Disponível</p>
                ) : (
                  <div className="grid grid-cols-2 rounded-lg border border-zinc-700 overflow-hidden text-[10px] font-extrabold uppercase">
                    <button
                      type="button"
                      onClick={() => setSeatModes((prev) => ({ ...prev, [seat]: 'open' }))}
                      className={`px-2 py-1 cursor-pointer ${seatModes[seat] === 'open' ? 'bg-emerald-600 text-white' : 'bg-zinc-900 text-zinc-400'}`}
                    >
                      Livre
                    </button>
                    <button
                      type="button"
                      onClick={() => setSeatModes((prev) => ({ ...prev, [seat]: 'bot' }))}
                      className={`px-2 py-1 cursor-pointer ${seatModes[seat] === 'bot' ? 'bg-[#d4af37] text-black' : 'bg-zinc-900 text-zinc-400'}`}
                    >
                      Bot
                    </button>
                  </div>
                )}
              </div>
            ))}
          </div>
        </div>

        <button onClick={handleCreate} className="w-full py-3 rounded-xl gold-btn text-black text-xs font-extrabold uppercase tracking-widest cursor-pointer">
          Criar mesa e entrar
        </button>
      </div>
    </div>
  );
};

const NumberField: React.FC<{ label: string; value: number; onChange: (value: number) => void }> = ({ label, value, onChange }) => (
  <label className="space-y-1 text-xs font-bold text-zinc-300">
    {label}
    <input
      type="number"
      min={0}
      value={value}
      onChange={(event) => onChange(Number(event.target.value))}
      className="w-full rounded-lg bg-zinc-950 border border-zinc-700 px-3 py-2 text-sm text-white outline-none focus:border-[#d4af37]"
    />
  </label>
);

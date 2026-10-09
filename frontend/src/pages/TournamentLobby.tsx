import React, { useCallback, useEffect, useState } from 'react';
import { ArrowLeft, Calendar, Coins, Play, Plus, RefreshCw, Trophy, Users } from 'lucide-react';
import { useNavigate } from 'react-router-dom';
import { HeaderLogo } from '../components/common/HeaderLogo';
import { useAuth } from '../context/AuthContext';
import { api } from '../services/api';
import type { Tournament } from '../types';

export const TournamentLobbyPage: React.FC = () => {
  const { user, token, refreshUser } = useAuth();
  const navigate = useNavigate();
  const [tournaments, setTournaments] = useState<Tournament[]>([]);
  const [loading, setLoading] = useState(true);
  const [showCreate, setShowCreate] = useState(false);
  const [busyID, setBusyID] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const loadTournaments = useCallback(async () => {
    if (!token) return;
    try {
      const result = await api.getTournaments(token);
      setTournaments(result);
      setError(null);
    } catch (loadError) {
      setError(loadError instanceof Error ? loadError.message : 'Falha ao carregar torneios');
    } finally {
      setLoading(false);
    }
  }, [token]);

  useEffect(() => {
    void loadTournaments();
    const interval = window.setInterval(() => void loadTournaments(), 5000);
    return () => window.clearInterval(interval);
  }, [loadTournaments]);

  const register = async (tournament: Tournament) => {
    if (!token || busyID) return;
    setBusyID(tournament.id);
    setError(null);
    try {
      await api.registerTournament(token, tournament.id);
      await refreshUser();
      await loadTournaments();
    } catch (registerError) {
      setError(registerError instanceof Error ? registerError.message : 'Falha na inscrição');
    } finally {
      setBusyID(null);
    }
  };

  const start = async (tournament: Tournament) => {
    if (!token || busyID) return;
    setBusyID(tournament.id);
    setError(null);
    try {
      await api.startTournament(token, tournament.id);
      await loadTournaments();
    } catch (startError) {
      setError(startError instanceof Error ? startError.message : 'Falha ao iniciar torneio');
    } finally {
      setBusyID(null);
    }
  };

  const leave = async (tournament: Tournament) => {
    if (!token || busyID) return;
    const isOwner = tournament.creator_user_id === user?.id;
    const confirmation = isOwner
      ? 'Ao sair, o controle do torneio será transferido ao inscrito mais antigo e seu buy-in será devolvido, se estiver inscrito. Continuar?'
      : 'Ao sair, sua inscrição será cancelada e o buy-in devolvido. Continuar?';
    if (!window.confirm(confirmation)) return;
    setBusyID(tournament.id);
    setError(null);
    try {
      await api.leaveTournament(token, tournament.id);
      await refreshUser();
      await loadTournaments();
    } catch (leaveError) {
      setError(leaveError instanceof Error ? leaveError.message : 'Falha ao sair do torneio');
    } finally {
      setBusyID(null);
    }
  };

  const enter = (tournament: Tournament) => {
    if (tournament.table_id && tournament.current_user_seat) {
      navigate(`/table/live?tableId=${tournament.table_id}&seat=${tournament.current_user_seat}&buyIn=${tournament.starting_stack}&mode=tournament`);
    }
  };

  return (
    <main className="min-h-screen bg-[#07090e] text-white p-4 sm:p-6 max-w-6xl mx-auto space-y-5">
      <div className="flex items-center justify-between border-b border-[#d4af37]/30 pb-3">
        <button onClick={() => navigate(-1)} className="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-zinc-900 border border-[#d4af37]/40 text-[#d4af37] text-xs font-bold">
          <ArrowLeft size={16} /> Voltar
        </button>
        <div className="flex gap-2">
          <button onClick={() => void loadTournaments()} disabled={loading} className="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-zinc-900 border border-[#d4af37]/40 text-[#d4af37] text-xs font-bold">
            <RefreshCw size={14} className={loading ? 'animate-spin' : ''} /> Atualizar
          </button>
          <button onClick={() => setShowCreate(true)} className="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-[#d4af37] text-black text-xs font-extrabold">
            <Plus size={15} /> Criar torneio
          </button>
        </div>
      </div>

      <header className="text-center space-y-2">
        <HeaderLogo />
        <h1 className="text-xl sm:text-3xl font-extrabold tracking-wider text-[#f5d77f] uppercase">Torneios Sit &amp; Go</h1>
        <p className="text-xs sm:text-sm text-zinc-400">Jogadores podem criar torneios de mesa única com até 9 vagas. O dono da sala inicia; se sair antes, o controle passa ao inscrito mais antigo.</p>
      </header>

      {error && <p role="alert" className="rounded-lg border border-red-500/40 bg-red-950/50 p-3 text-sm text-red-200">{error}</p>}

      {loading && tournaments.length === 0 ? (
        <p className="py-12 text-center text-zinc-400">Carregando torneios...</p>
      ) : tournaments.length === 0 ? (
        <div className="rounded-xl border border-[#d4af37]/30 bg-zinc-900/50 p-8 text-center text-zinc-300">
          Nenhum torneio criado. Crie o primeiro Sit & Go.
        </div>
      ) : (
        <div className="grid gap-4 md:grid-cols-2">
          {tournaments.map((tournament) => (
            <article key={tournament.id} className="rounded-xl border border-[#d4af37]/30 bg-[#12151c] p-4 space-y-4">
              <div className="flex items-start justify-between gap-3">
                <div>
                  <h2 className="font-extrabold text-[#f5d77f]">{tournament.nome}</h2>
                  <p className="text-xs text-zinc-400 mt-1">{statusLabel(tournament.status)}</p>
                </div>
                <Trophy className="text-[#d4af37]" size={20} />
              </div>

              <div className="grid grid-cols-2 gap-2">
                <Stat icon={<Coins size={14} />} label="Buy-in" value={`${tournament.buy_in.toLocaleString('pt-BR')} fichas`} />
                <Stat icon={<Trophy size={14} />} label="Prêmio acumulado" value={`${tournament.garantido.toLocaleString('pt-BR')} fichas`} />
                <Stat icon={<Users size={14} />} label="Inscritos" value={`${tournament.inscritos}/${tournament.max_inscritos}`} />
                <Stat icon={<Calendar size={14} />} label="Blinds" value={`${tournament.small_blind}/${tournament.big_blind} · ${tournament.blind_interval_min} min`} />
                <Stat icon={<Coins size={14} />} label="Stack inicial" value={`${tournament.starting_stack.toLocaleString('pt-BR')} fichas`} />
                <Stat icon={<Play size={14} />} label="Nível" value={String(tournament.blind_level + 1)} />
              </div>

              {tournament.status === 'aberto' && (
                <div className="flex flex-wrap gap-2">
                  {tournament.current_user_entry ? (
                    <span className="flex-1 rounded-lg border border-emerald-500/40 bg-emerald-950/50 py-2 text-center text-xs font-bold text-emerald-200">Inscrição confirmada · assento #{tournament.current_user_seat}</span>
                  ) : (
                    <button
                      onClick={() => void register(tournament)}
                      disabled={busyID !== null || tournament.inscritos >= tournament.max_inscritos}
                      className="flex-1 rounded-lg bg-emerald-700 py-2 text-xs font-extrabold text-white disabled:opacity-50"
                    >
                      {busyID === tournament.id ? 'Processando...' : `Inscrever-se · ${tournament.buy_in.toLocaleString('pt-BR')} fichas`}
                    </button>
                  )}
                  {(tournament.current_user_entry || tournament.creator_user_id === user?.id) && (
                    <button
                      onClick={() => void leave(tournament)}
                      disabled={busyID !== null}
                      className="rounded-lg border border-red-500/50 px-3 py-2 text-xs font-bold text-red-200 disabled:opacity-50"
                    >
                      {tournament.creator_user_id === user?.id ? 'Sair e transferir' : 'Sair do torneio'}
                    </button>
                  )}
                  {tournament.creator_user_id === user?.id && (
                    <button
                      onClick={() => void start(tournament)}
                      disabled={busyID !== null || tournament.inscritos < 2}
                      className="rounded-lg bg-[#d4af37] px-4 py-2 text-xs font-extrabold text-black disabled:opacity-50"
                      title={tournament.inscritos < 2 ? 'São necessários pelo menos dois jogadores' : 'Iniciar torneio'}
                    >
                      Iniciar
                    </button>
                  )}
                </div>
              )}

              {tournament.status === 'em_andamento' && (
                tournament.current_user_status === 'playing' ? (
                  <button onClick={() => enter(tournament)} className="w-full rounded-lg bg-emerald-700 py-2.5 text-xs font-extrabold uppercase text-white">
                    Entrar na mesa · assento #{tournament.current_user_seat}
                  </button>
                ) : tournament.current_user_status === 'eliminated' ? (
                  <p className="text-center text-xs text-zinc-400">Você foi eliminado; a inscrição permanece no torneio até a premiação.</p>
                ) : <p className="text-center text-xs text-zinc-400">Inscrições encerradas; torneio em andamento.</p>
              )}

              {tournament.status === 'concluido' && (
                <p className="rounded-lg bg-amber-950/40 p-3 text-center text-xs text-amber-100">
                  Vencedor: {tournament.winner_name || tournament.winner_user_id || '—'} · prêmio {tournament.garantido.toLocaleString('pt-BR')} fichas
                </p>
              )}
            </article>
          ))}
        </div>
      )}

      {showCreate && (
        <CreateTournamentModal
          token={token}
          onClose={() => setShowCreate(false)}
          onCreated={async () => {
            setShowCreate(false);
            await loadTournaments();
          }}
        />
      )}
    </main>
  );
};

const statusLabel = (status: string) => ({
  aberto: 'Inscrições abertas · início manual',
  em_andamento: 'Em andamento',
  concluido: 'Concluído',
}[status] || status);

const Stat: React.FC<{ icon: React.ReactNode; label: string; value: string }> = ({ icon, label, value }) => (
  <div className="rounded-lg border border-zinc-800 bg-[#0b0e14] p-2">
    <div className="flex items-center gap-1 text-[10px] font-bold uppercase text-[#d4af37]">{icon}{label}</div>
    <p className="mt-1 text-xs font-bold text-white">{value}</p>
  </div>
);

const CreateTournamentModal: React.FC<{
  token: string | null;
  onClose: () => void;
  onCreated: () => Promise<void>;
}> = ({ token, onClose, onCreated }) => {
  const [name, setName] = useState('Sit & Go dos Amigos');
  const [buyIn, setBuyIn] = useState(100);
  const [maxEntries, setMaxEntries] = useState(9);
  const [startingStack, setStartingStack] = useState(5000);
  const [blindInterval, setBlindInterval] = useState(10);
  const [smallBlind, setSmallBlind] = useState(25);
  const [bigBlind, setBigBlind] = useState(50);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const submit = async (event: React.FormEvent) => {
    event.preventDefault();
    if (!token || saving) return;
    setSaving(true);
    setError(null);
    try {
      await api.createTournament(token, {
        nome: name.trim(),
        buy_in: buyIn,
        max_inscritos: maxEntries,
        starting_stack: startingStack,
        blind_interval_min: blindInterval,
        small_blind: smallBlind,
        big_blind: bigBlind,
      });
      await onCreated();
    } catch (createError) {
      setError(createError instanceof Error ? createError.message : 'Falha ao criar torneio');
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/80 p-4">
      <form onSubmit={submit} className="w-full max-w-lg space-y-4 rounded-xl border border-[#d4af37]/50 bg-[#12151c] p-5">
        <div className="flex items-center justify-between">
          <h2 className="font-extrabold uppercase text-[#f5d77f]">Criar Sit &amp; Go</h2>
          <button type="button" onClick={onClose} className="text-sm text-zinc-400">Fechar</button>
        </div>
        <label className="block text-xs text-zinc-300">Nome
          <input required maxLength={150} value={name} onChange={(e) => setName(e.target.value)} className="mt-1 w-full rounded bg-zinc-950 p-2 text-white" />
        </label>
        <div className="grid grid-cols-2 gap-3">
          <NumberField label="Buy-in (fichas)" value={buyIn} min={1} onChange={setBuyIn} />
          <NumberField label="Máximo de jogadores" value={maxEntries} min={2} max={9} onChange={setMaxEntries} />
          <NumberField label="Stack inicial" value={startingStack} min={1} onChange={setStartingStack} />
          <NumberField label="Intervalo dos blinds (min)" value={blindInterval} min={1} onChange={setBlindInterval} />
          <NumberField label="Small blind" value={smallBlind} min={1} onChange={setSmallBlind} />
          <NumberField label="Big blind" value={bigBlind} min={1} onChange={setBigBlind} />
        </div>
        {error && <p role="alert" className="text-xs text-red-300">{error}</p>}
        <button disabled={saving} className="w-full rounded-lg bg-[#d4af37] py-2.5 text-xs font-extrabold uppercase text-black disabled:opacity-50">
          {saving ? 'Criando...' : 'Criar torneio'}
        </button>
      </form>
    </div>
  );
};

const NumberField: React.FC<{
  label: string;
  value: number;
  min: number;
  max?: number;
  onChange: (value: number) => void;
}> = ({ label, value, min, max, onChange }) => (
  <label className="block text-xs text-zinc-300">{label}
    <input
      type="number"
      min={min}
      max={max}
      required
      value={value}
      onChange={(event) => onChange(Number(event.target.value))}
      className="mt-1 w-full rounded bg-zinc-950 p-2 text-white"
    />
  </label>
);

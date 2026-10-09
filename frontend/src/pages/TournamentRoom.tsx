import React, { useCallback, useEffect, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { ArrowLeft, CheckCircle2, RefreshCw, ShieldCheck, Users } from 'lucide-react';
import { useAuth } from '../context/AuthContext';
import { api } from '../services/api';
import type { TournamentRoom as TournamentRoomData } from '../types';

export const TournamentRoomPage: React.FC = () => {
  const { tournamentId = '' } = useParams();
  const { user, token, refreshUser } = useAuth();
  const navigate = useNavigate();
  const [room, setRoom] = useState<TournamentRoomData | null>(null);
  const [selectedSeat, setSelectedSeat] = useState<number | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');

  const loadRoom = useCallback(async () => {
    if (!token || !tournamentId) return;
    try {
      const result = await api.getTournamentRoom(token, tournamentId);
      setRoom(result);
      setError('');
      setSelectedSeat((current) => {
        if (current && !result.seats.some((seat) => seat.seat_number === current && seat.user_id !== user?.id)) {
          return current;
        }
        return result.current_user_seat ?? null;
      });
      if (result.status === 'em_andamento' && result.current_user_status === 'playing' && result.current_user_seat) {
        navigate(`/table/live?tableId=${result.table_id}&seat=${result.current_user_seat}&buyIn=${result.starting_stack}&mode=tournament`, { replace: true });
      }
    } catch (loadError) {
      setError(loadError instanceof Error ? loadError.message : 'Falha ao carregar a sala');
    } finally {
      setLoading(false);
    }
  }, [navigate, token, tournamentId, user?.id]);

  useEffect(() => {
    void loadRoom();
    const interval = window.setInterval(() => void loadRoom(), 2500);
    return () => window.clearInterval(interval);
  }, [loadRoom]);

  const confirmSeat = async () => {
    if (!token || !room || !selectedSeat || saving || room.status !== 'aberto') return;
    if (room.current_user_seat === selectedSeat) return;
    setSaving(true);
    setError('');
    try {
      if (room.current_user_entry) {
        await api.changeTournamentSeat(token, tournamentId, selectedSeat);
      } else {
        await api.registerTournament(token, tournamentId, selectedSeat);
        await refreshUser();
      }
      await loadRoom();
    } catch (saveError) {
      setError(saveError instanceof Error ? saveError.message : 'Falha ao confirmar o assento');
      await loadRoom();
    } finally {
      setSaving(false);
    }
  };

  const startTournament = async () => {
    if (!token || !room || saving) return;
    setSaving(true);
    setError('');
    try {
      await api.startTournament(token, tournamentId);
      await loadRoom();
    } catch (startError) {
      setError(startError instanceof Error ? startError.message : 'Falha ao iniciar torneio');
    } finally {
      setSaving(false);
    }
  };

  const ownSeat = room?.current_user_seat;
  const selectedSeatIsFree = !!selectedSeat && !room?.seats.some(
    (seat) => seat.seat_number === selectedSeat && seat.user_id !== user?.id
  );
  const canConfirm = !!selectedSeat && selectedSeatIsFree && selectedSeat !== ownSeat && room?.status === 'aberto';
  const isOwner = room?.creator_user_id === user?.id;

  return (
    <main className="min-h-screen bg-[#07090e] text-white p-4 sm:p-6 max-w-5xl mx-auto space-y-5">
      <div className="flex items-center justify-between border-b border-[#d4af37]/30 pb-3">
        <button onClick={() => navigate('/tournaments')} className="flex items-center gap-2 rounded-lg bg-zinc-900 px-3 py-2 text-xs font-bold text-[#d4af37]">
          <ArrowLeft size={16} /> Torneios
        </button>
        <button onClick={() => void loadRoom()} disabled={loading} className="flex items-center gap-2 rounded-lg bg-zinc-900 px-3 py-2 text-xs font-bold text-[#d4af37]">
          <RefreshCw size={14} className={loading ? 'animate-spin' : ''} /> Atualizar
        </button>
      </div>

      <header className="text-center space-y-2">
        <h1 className="text-2xl font-extrabold uppercase tracking-wider text-[#f5d77f]">{room?.name || 'Sala do torneio'}</h1>
        <p className="text-sm text-zinc-400">
          {room?.status === 'aberto'
            ? 'Sala aberta · escolha seu assento; você poderá mudar enquanto o torneio não começar.'
            : room?.current_user_status === 'eliminated'
              ? 'Torneio em andamento · você foi eliminado.'
              : room?.status === 'concluido'
                ? 'Torneio concluído.'
                : 'Torneio em andamento.'}
        </p>
        {room && <p className="text-xs text-zinc-500">Buy-in: {room.buy_in.toLocaleString('pt-BR')} fichas · Blinds {room.small_blind}/{room.big_blind}</p>}
      </header>

      {error && <p role="alert" className="rounded-lg border border-red-500/40 bg-red-950/50 p-3 text-sm text-red-200">{error}</p>}
      {loading && !room ? (
        <p className="py-12 text-center text-zinc-400">Carregando sala...</p>
      ) : room ? (
        <>
          <section className="rounded-2xl border border-[#d4af37]/30 bg-[#12151c] p-4 sm:p-6">
            <div className="mb-4 flex items-center justify-center gap-2 text-xs font-bold uppercase tracking-wider text-[#d4af37]">
              <Users size={16} />
              {room.seats.length}/{room.max_seats} assentos ocupados
            </div>
            <div className="grid grid-cols-3 gap-3 sm:grid-cols-5">
              {Array.from({ length: room.max_seats }, (_, index) => {
                const seatNumber = index + 1;
                const occupant = room.seats.find((seat) => seat.seat_number === seatNumber);
                const isMine = occupant?.user_id === user?.id;
                const isSelected = selectedSeat === seatNumber;
                const available = !occupant || isMine;
                return (
                  <button
                    key={seatNumber}
                    type="button"
                    disabled={!available || room.status !== 'aberto'}
                    onClick={() => setSelectedSeat(seatNumber)}
                    className={`min-h-24 rounded-xl border p-3 text-center transition ${
                      isSelected
                        ? 'border-[#f5d77f] bg-amber-900/50 text-[#f5d77f] ring-2 ring-[#d4af37]/40'
                        : isMine
                          ? 'border-emerald-500/60 bg-emerald-950/40 text-emerald-200'
                          : available && room.status === 'aberto'
                            ? 'border-emerald-500/40 bg-emerald-950/20 text-emerald-200 hover:bg-emerald-900/40'
                            : 'border-zinc-700 bg-zinc-900 text-zinc-400'
                    } disabled:cursor-not-allowed`}
                  >
                    <span className="block text-xs font-extrabold">Assento {seatNumber}</span>
                    <span className="mt-2 block truncate text-xs">
                      {occupant ? (isMine ? 'Você' : occupant.player_name) : room.status === 'aberto' ? 'Livre' : '—'}
                    </span>
                    {isSelected && <CheckCircle2 size={16} className="mx-auto mt-2" />}
                  </button>
                );
              })}
            </div>
          </section>

          {room.status === 'aberto' ? (
            <div className="flex flex-col gap-3 sm:flex-row sm:justify-center">
              <button
                onClick={() => void confirmSeat()}
                disabled={!canConfirm || saving}
                className="rounded-lg bg-emerald-700 px-6 py-3 text-xs font-extrabold uppercase text-white disabled:cursor-not-allowed disabled:opacity-50"
              >
                {saving ? 'Processando...' : room.current_user_entry ? 'Confirmar novo assento' : 'Inscrever-se neste assento'}
              </button>
              {isOwner && (
                <button
                  onClick={() => void startTournament()}
                  disabled={saving || room.seats.length < 2}
                  className="flex items-center justify-center gap-2 rounded-lg bg-[#d4af37] px-6 py-3 text-xs font-extrabold uppercase text-black disabled:cursor-not-allowed disabled:opacity-50"
                  title={room.seats.length < 2 ? 'São necessários ao menos dois jogadores inscritos' : 'Iniciar torneio'}
                >
                  <ShieldCheck size={16} /> Iniciar torneio
                </button>
              )}
            </div>
          ) : room.current_user_status === 'eliminated' ? (
            <p className="text-center text-sm text-zinc-400">Sua inscrição permanece registrada até a premiação do torneio.</p>
          ) : !room.current_user_entry ? (
            <p className="text-center text-sm text-zinc-400">As inscrições foram encerradas; você não está inscrito neste torneio.</p>
          ) : null}

          {room.status === 'aberto' && !room.current_user_entry && (
            <p className="text-center text-xs text-zinc-500">O buy-in será debitado uma única vez quando confirmar a inscrição. Depois disso, você poderá trocar para qualquer assento livre antes do início.</p>
          )}
        </>
      ) : null}
    </main>
  );
};

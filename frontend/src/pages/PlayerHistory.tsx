import React, { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { ArrowLeft, History } from 'lucide-react';
import { useAuth } from '../context/AuthContext';
import { api } from '../services/api';
import type { GameHistoryEntry } from '../types';

export const PlayerHistoryPage: React.FC = () => {
  const { token } = useAuth();
  const navigate = useNavigate();
  const [history, setHistory] = useState<GameHistoryEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    if (!token) {
      setLoading(false);
      setError('Sessão não autenticada');
      return;
    }
    api.getPlayerGameHistory(token)
      .then(setHistory)
      .catch((err: unknown) => setError(err instanceof Error ? err.message : 'Falha ao carregar histórico'))
      .finally(() => setLoading(false));
  }, [token]);

  const chips = (amount: number) => `${amount.toLocaleString('pt-BR')} fichas`;
  const date = (value: string) => new Date(value).toLocaleString('pt-BR');

  return (
    <main className="min-h-screen max-w-4xl mx-auto p-6 text-white space-y-5">
      <button onClick={() => navigate(-1)} className="flex items-center gap-2 text-[#f5d77f] text-sm">
        <ArrowLeft size={16} /> Voltar
      </button>
      <header className="flex items-center gap-3">
        <History className="text-[#f5d77f]" />
        <h1 className="text-2xl font-bold">Meu histórico de partidas</h1>
      </header>
      {error && <p className="rounded border border-red-500/50 p-3 text-red-200">{error}</p>}
      {loading ? (
        <p className="text-sm text-zinc-400">Carregando histórico...</p>
      ) : history.length === 0 ? (
        <p className="rounded-xl border border-[#d4af37]/30 bg-[#12151c] p-4 text-sm text-zinc-400">
          Você ainda não participou de nenhuma partida.
        </p>
      ) : (
        <section className="space-y-3">
          {history.map((entry) => (
            <article key={`${entry.game_type}-${entry.id}`} className="rounded-xl border border-[#d4af37]/30 bg-[#12151c] p-4 space-y-3">
              <div className="flex flex-wrap items-start justify-between gap-2">
                <div>
                  <h2 className="font-bold text-[#f5d77f]">{entry.table_name}</h2>
                  <p className="text-xs text-zinc-400">
                    {entry.game_type === 'torneio' ? 'Torneio' : 'Cash game'} · {date(entry.started_at)}
                  </p>
                </div>
                {entry.game_type === 'torneio' && (
                  <span className={`rounded-full px-3 py-1 text-xs font-bold ${
                    entry.won === true ? 'bg-emerald-900/60 text-emerald-200' :
                      entry.won === false ? 'bg-zinc-800 text-zinc-300' : 'bg-amber-900/50 text-amber-200'
                  }`}>
                    {entry.won === true ? 'Vencedor' : entry.won === false ? 'Não venceu' : 'Não finalizado'}
                  </span>
                )}
              </div>
              <div className="grid gap-2 text-sm sm:grid-cols-2">
                <p className="text-zinc-300">Total investido: <strong className="text-white">{chips(entry.amount_invested_chips)}</strong></p>
                {entry.finished_at ? (
                  <p className="text-zinc-300">
                    {entry.game_type === 'torneio' ? 'Prêmio recebido' : 'Saldo retirado'}:{' '}
                    <strong className="text-white">{chips(entry.payout_chips)}</strong>
                  </p>
                ) : (
                  <p className="text-amber-200">Partida não finalizada</p>
                )}
              </div>
              {entry.finished_at && <p className="text-xs text-zinc-500">Finalizada em {date(entry.finished_at)}</p>}
            </article>
          ))}
        </section>
      )}
    </main>
  );
};

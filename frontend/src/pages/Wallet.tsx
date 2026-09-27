import React, { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { ArrowLeft, WalletCards } from 'lucide-react';
import { useAuth } from '../context/AuthContext';
import { api } from '../services/api';

export const WalletPage: React.FC = () => {
  const { token, user, updateUser } = useAuth();
  const navigate = useNavigate();
  const [wallet, setWallet] = useState<{ balance_cents: number; available_cents: number; reserved_cents: number } | null>(null);
  const [ledger, setLedger] = useState<Array<{ type: string; amount_cents: number; created_at: string }>>([]);
  const [error, setError] = useState('');

  const load = async () => {
    if (!token) return;
    try {
      const data = await api.getWallet(token);
      setWallet(data.wallet);
      setLedger(data.ledger);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Falha ao carregar carteira');
    }
  };

  useEffect(() => { void load(); }, [token]);

  const mockDeposit = async () => {
    if (!token) return;
    try {
      const updated = await api.devDeposit(token, 10000);
      updateUser(updated);
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Falha no depósito MOCK/DEV');
    }
  };

  const money = (cents: number) => (cents / 100).toLocaleString('pt-BR', { style: 'currency', currency: 'BRL' });

  return (
    <main className="min-h-screen max-w-3xl mx-auto p-6 text-white space-y-5">
      <button onClick={() => navigate(-1)} className="flex items-center gap-2 text-[#f5d77f] text-sm"><ArrowLeft size={16} /> Voltar</button>
      <header className="flex items-center gap-3"><WalletCards className="text-[#f5d77f]" /><h1 className="text-2xl font-bold">Minha carteira</h1></header>
      {error && <p className="rounded border border-red-500/50 p-3 text-red-200">{error}</p>}
      <section className="grid gap-3 sm:grid-cols-3">
        <div className="rounded-xl border border-[#d4af37]/40 bg-[#12151c] p-4"><p className="text-xs text-zinc-400">Saldo total</p><strong>{money(wallet?.balance_cents ?? 0)}</strong></div>
        <div className="rounded-xl border border-[#d4af37]/40 bg-[#12151c] p-4"><p className="text-xs text-zinc-400">Disponível</p><strong>{money(wallet?.available_cents ?? 0)}</strong></div>
        <div className="rounded-xl border border-[#d4af37]/40 bg-[#12151c] p-4"><p className="text-xs text-zinc-400">Reservado em mesas</p><strong>{money(wallet?.reserved_cents ?? 0)}</strong></div>
      </section>
      <section className="rounded-xl border border-[#d4af37]/30 bg-[#12151c] p-4 space-y-3">
        <h2 className="font-bold">Adicionar saldo <span className="text-xs text-amber-300">(MOCK/DEV)</span></h2>
        <p className="text-xs text-zinc-400">Não é pagamento real e não confirma transações externas.</p>
        <button onClick={() => void mockDeposit()} className="rounded bg-[#d4af37] px-4 py-2 text-sm font-bold text-black">Adicionar R$ 100 (MOCK/DEV)</button>
      </section>
      <section className="rounded-xl border border-[#d4af37]/30 bg-[#12151c] p-4">
        <h2 className="mb-3 font-bold">Histórico</h2>
        {ledger.length === 0 ? <p className="text-sm text-zinc-400">Nenhuma movimentação.</p> : ledger.map((entry) => (
          <div key={`${entry.created_at}-${entry.type}`} className="flex justify-between border-b border-zinc-800 py-2 text-sm">
            <span>{entry.type}</span><span>{money(entry.amount_cents)}</span>
          </div>
        ))}
      </section>
      <p className="text-xs text-zinc-500">{user?.email}</p>
    </main>
  );
};

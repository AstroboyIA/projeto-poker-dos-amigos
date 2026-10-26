import React, { useState } from 'react';
import { ArrowLeft, Save, UserRound } from 'lucide-react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';
import { api } from '../services/api';

export const PlayerSettingsPage: React.FC = () => {
  const { user, token, updateSession } = useAuth();
  const navigate = useNavigate();
  const [nomeCompleto, setNomeCompleto] = useState(user?.nome_completo ?? '');
  const [telefone, setTelefone] = useState(user?.telefone ?? '');
  const [dataNascimento, setDataNascimento] = useState(user?.data_nascimento ?? '');
  const [cidadeEstado, setCidadeEstado] = useState(user?.cidade_estado ?? '');
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');

  const handleSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!token) {
      setError('Sua sessão expirou. Entre novamente para salvar as alterações.');
      return;
    }

    setSaving(true);
    setError('');
    setSuccess('');
    try {
      const updatedSession = await api.updateUserProfile(token, {
        nome_completo: nomeCompleto.trim(),
        telefone: telefone.trim(),
        data_nascimento: dataNascimento.trim(),
        cidade_estado: cidadeEstado.trim(),
      });
      updateSession(updatedSession.token, updatedSession.user);
      setSuccess('Dados da conta atualizados.');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Falha ao atualizar os dados da conta.');
    } finally {
      setSaving(false);
    }
  };

  return (
    <main className="min-h-screen max-w-3xl mx-auto p-6 text-white space-y-6">
      <button onClick={() => navigate('/player')} className="flex items-center gap-2 text-[#f5d77f] text-sm">
        <ArrowLeft size={16} /> Voltar ao painel
      </button>

      <header className="flex items-center gap-3">
        <UserRound className="text-[#f5d77f]" />
        <div>
          <h1 className="text-2xl font-bold">Configurações da conta</h1>
          <p className="text-sm text-zinc-400">Atualize seus dados pessoais.</p>
        </div>
      </header>

      {error && <p role="alert" className="rounded border border-red-500/50 p-3 text-sm text-red-200">{error}</p>}
      {success && <p role="status" className="rounded border border-emerald-500/50 p-3 text-sm text-emerald-200">{success}</p>}

      <form onSubmit={handleSubmit} className="space-y-5">
        <section className="rounded-xl border border-[#d4af37]/30 bg-[#12151c] p-5 space-y-4">
          <h2 className="font-bold text-[#f5d77f]">Dados pessoais</h2>

          <label className="block space-y-1.5">
            <span className="text-sm text-zinc-300">Nome completo</span>
            <input
              value={nomeCompleto}
              onChange={(event) => setNomeCompleto(event.target.value)}
              maxLength={255}
              autoComplete="name"
              className="w-full rounded-lg border border-zinc-700 bg-zinc-900 px-3 py-2 text-white outline-none focus:border-[#d4af37]"
            />
          </label>

          <div className="grid gap-4 sm:grid-cols-2">
            <label className="block space-y-1.5">
              <span className="text-sm text-zinc-300">Telefone</span>
              <input
                value={telefone}
                onChange={(event) => setTelefone(event.target.value)}
                maxLength={50}
                autoComplete="tel"
                className="w-full rounded-lg border border-zinc-700 bg-zinc-900 px-3 py-2 text-white outline-none focus:border-[#d4af37]"
              />
            </label>
            <label className="block space-y-1.5">
              <span className="text-sm text-zinc-300">Data de nascimento</span>
              <input
                value={dataNascimento}
                onChange={(event) => setDataNascimento(event.target.value)}
                maxLength={20}
                placeholder="DD/MM/AAAA"
                autoComplete="bday"
                className="w-full rounded-lg border border-zinc-700 bg-zinc-900 px-3 py-2 text-white outline-none focus:border-[#d4af37]"
              />
            </label>
          </div>

          <label className="block space-y-1.5">
            <span className="text-sm text-zinc-300">Cidade/estado</span>
            <input
              value={cidadeEstado}
              onChange={(event) => setCidadeEstado(event.target.value)}
              maxLength={100}
              autoComplete="address-level2"
              className="w-full rounded-lg border border-zinc-700 bg-zinc-900 px-3 py-2 text-white outline-none focus:border-[#d4af37]"
            />
          </label>
        </section>

        <section className="rounded-xl border border-zinc-800 bg-zinc-900/60 p-5 space-y-3">
          <h2 className="font-bold text-zinc-200">Identificação da conta</h2>
          <p className="text-xs text-zinc-400">Username e e-mail não podem ser alterados nesta etapa.</p>
          <div className="grid gap-3 sm:grid-cols-2">
            <div>
              <p className="text-xs text-zinc-500">Username</p>
              <p className="text-sm text-zinc-300">{user?.username || '—'}</p>
            </div>
            <div>
              <p className="text-xs text-zinc-500">E-mail</p>
              <p className="text-sm text-zinc-300">{user?.email || 'Não informado'}</p>
            </div>
          </div>
        </section>

        <button
          type="submit"
          disabled={saving}
          className="flex items-center justify-center gap-2 rounded-lg bg-[#d4af37] px-5 py-2.5 font-bold text-black transition hover:bg-[#f5d77f] disabled:cursor-not-allowed disabled:opacity-60"
        >
          <Save size={16} />
          {saving ? 'Salvando...' : 'Salvar alterações'}
        </button>
      </form>
    </main>
  );
};

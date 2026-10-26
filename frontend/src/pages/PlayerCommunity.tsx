import React, { useState } from 'react';
import { ArrowLeft, AtSign, CalendarDays, MapPin, Users, UserRound } from 'lucide-react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';

type CommunitySection = 'profile' | 'friends';

export const PlayerCommunityPage: React.FC = () => {
  const { user } = useAuth();
  const navigate = useNavigate();
  const [section, setSection] = useState<CommunitySection>('profile');
  const displayName = user?.nome_completo || user?.username || 'Jogador';
  const initials = displayName
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((part) => part.charAt(0).toUpperCase())
    .join('');

  return (
    <main className="min-h-screen max-w-5xl mx-auto p-6 text-white space-y-6">
      <button onClick={() => navigate('/player')} className="flex items-center gap-2 text-[#f5d77f] text-sm">
        <ArrowLeft size={16} /> Voltar ao painel
      </button>

      <header className="flex items-center gap-3">
        <Users className="text-[#f5d77f]" />
        <div>
          <h1 className="text-2xl font-bold">Jogadores</h1>
          <p className="text-sm text-zinc-400">Seu perfil e seus amigos do clube.</p>
        </div>
      </header>

      <div className="grid gap-6 md:grid-cols-[220px_minmax(0,1fr)]">
        <nav aria-label="Seções de jogadores" className="flex gap-2 md:flex-col">
          <button
            type="button"
            onClick={() => setSection('profile')}
            aria-current={section === 'profile' ? 'page' : undefined}
            className={`flex flex-1 items-center gap-3 rounded-xl border px-4 py-3 text-left transition md:flex-none ${
              section === 'profile'
                ? 'border-[#d4af37]/60 bg-[#d4af37]/10 text-[#f5d77f]'
                : 'border-zinc-800 bg-[#12151c] text-zinc-300 hover:border-zinc-600'
            }`}
          >
            <UserRound size={18} />
            <span className="text-sm font-semibold">Meu perfil</span>
          </button>
          <button
            type="button"
            onClick={() => setSection('friends')}
            aria-current={section === 'friends' ? 'page' : undefined}
            className={`flex flex-1 items-center gap-3 rounded-xl border px-4 py-3 text-left transition md:flex-none ${
              section === 'friends'
                ? 'border-[#d4af37]/60 bg-[#d4af37]/10 text-[#f5d77f]'
                : 'border-zinc-800 bg-[#12151c] text-zinc-300 hover:border-zinc-600'
            }`}
          >
            <Users size={18} />
            <span className="text-sm font-semibold">Amigos</span>
          </button>
        </nav>

        <section aria-live="polite" className="min-w-0">
          {section === 'profile' ? (
            <div className="space-y-5">
              <header>
                <h2 className="text-xl font-bold text-[#f5d77f]">Meu perfil</h2>
                <p className="mt-1 text-sm text-zinc-400">Informações da sua conta no clube.</p>
              </header>

              <article className="rounded-xl border border-[#d4af37]/30 bg-[#12151c] p-5 sm:p-6">
                <div className="flex flex-col gap-5 sm:flex-row sm:items-center">
                  <div className="flex h-20 w-20 shrink-0 items-center justify-center rounded-full border-2 border-[#d4af37]/70 bg-[#d4af37]/10 text-2xl font-extrabold text-[#f5d77f]">
                    {initials || 'J'}
                  </div>
                  <div className="min-w-0">
                    <h3 className="truncate text-xl font-bold text-white">{displayName}</h3>
                    <p className="mt-1 text-sm text-[#d4af37]">@{user?.username || 'jogador'}</p>
                    <span className="mt-3 inline-flex rounded-full border border-emerald-700/60 bg-emerald-950/50 px-3 py-1 text-xs font-semibold text-emerald-300">
                      {user?.status === 'ativo' ? 'Membro ativo' : 'Membro do clube'}
                    </span>
                  </div>
                </div>

                <div className="mt-6 grid gap-4 border-t border-zinc-800 pt-5 sm:grid-cols-2">
                  <ProfileDetail icon={<AtSign size={16} />} label="Username" value={user?.username || '—'} />
                  <ProfileDetail icon={<MapPin size={16} />} label="Cidade/estado" value={user?.cidade_estado || 'Não informado'} />
                  <ProfileDetail icon={<CalendarDays size={16} />} label="Data de nascimento" value={user?.data_nascimento || 'Não informada'} />
                  <ProfileDetail icon={<UserRound size={16} />} label="Nome" value={user?.nome_completo || 'Não informado'} />
                </div>

                <button
                  type="button"
                  onClick={() => navigate('/account/settings')}
                  className="mt-6 rounded-lg border border-[#d4af37]/50 px-4 py-2 text-sm font-semibold text-[#f5d77f] transition hover:bg-[#d4af37]/10"
                >
                  Editar meus dados
                </button>
              </article>
            </div>
          ) : (
            <div className="space-y-5">
              <header>
                <h2 className="text-xl font-bold text-[#f5d77f]">Amigos</h2>
                <p className="mt-1 text-sm text-zinc-400">Acompanhe seus amigos e encontre outros jogadores do clube.</p>
              </header>

              <div className="flex min-h-64 flex-col items-center justify-center rounded-xl border border-dashed border-zinc-700 bg-[#12151c] px-6 py-10 text-center">
                <div className="mb-4 flex h-14 w-14 items-center justify-center rounded-full border border-[#d4af37]/40 bg-[#d4af37]/10">
                  <Users size={24} className="text-[#f5d77f]" />
                </div>
                <h3 className="font-bold text-zinc-200">Sua lista de amigos começa aqui</h3>
                <p className="mt-2 max-w-md text-sm text-zinc-400">
                  A lista está vazia. A busca de jogadores e os pedidos de amizade ainda não estão disponíveis.
                </p>
              </div>
            </div>
          )}
        </section>
      </div>
    </main>
  );
};

const ProfileDetail: React.FC<{ icon: React.ReactNode; label: string; value: string }> = ({ icon, label, value }) => (
  <div className="flex min-w-0 items-start gap-3">
    <span className="mt-0.5 text-[#d4af37]">{icon}</span>
    <div className="min-w-0">
      <p className="text-xs text-zinc-500">{label}</p>
      <p className="truncate text-sm text-zinc-200">{value}</p>
    </div>
  </div>
);

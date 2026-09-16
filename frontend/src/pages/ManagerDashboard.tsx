import React, { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Users,
  Trophy,
  Calendar,
  DollarSign,
  Layers,
  TrendingUp,
  Megaphone,
  Settings,
  LogOut,
  Clock,
  ShieldCheck,
} from 'lucide-react';
import { HeaderLogo } from '../components/common/HeaderLogo';
import { useAuth } from '../context/AuthContext';

export const ManagerDashboard: React.FC = () => {
  const { user, logout } = useAuth();
  const navigate = useNavigate();
  const [currentDateTime, setCurrentDateTime] = useState('');
  const [selectedModule, setSelectedModule] = useState<string | null>(null);

  useEffect(() => {
    const updateTime = () => {
      const now = new Date();
      const day = String(now.getDate()).padStart(2, '0');
      const month = String(now.getMonth() + 1).padStart(2, '0');
      const year = now.getFullYear();
      const hours = String(now.getHours()).padStart(2, '0');
      const minutes = String(now.getMinutes()).padStart(2, '0');
      setCurrentDateTime(`${day}/${month}/${year} ${hours}:${minutes}`);
    };
    updateTime();
    const interval = setInterval(updateTime, 1000);
    return () => clearInterval(interval);
  }, []);

  const handleLogout = () => {
    logout();
    navigate('/');
  };

  const modules = [
    {
      id: 'membros',
      title: 'MEMBROS',
      desc: 'Gerencie os membros do clube',
      icon: Users,
      color: 'from-amber-600 to-yellow-800',
      action: () => setSelectedModule('Membros do Clube (Gerenciamento)'),
    },
    {
      id: 'torneios',
      title: 'TORNEIOS',
      desc: 'Crie e gerencie torneios',
      icon: Calendar,
      color: 'from-blue-700 to-indigo-950',
      action: () => setSelectedModule('Gestão de Torneios e Estruturas'),
    },
    {
      id: 'ranking',
      title: 'RANKING',
      desc: 'Acompanhe o ranking dos jogadores',
      icon: Trophy,
      color: 'from-purple-800 to-amber-900',
      action: () => setSelectedModule('Ranking e Pontuação dos Membros'),
    },
    {
      id: 'financeiro',
      title: 'FINANCEIRO',
      desc: 'Entradas, saídas e relatórios',
      icon: DollarSign,
      color: 'from-yellow-700 to-amber-950',
      action: () => setSelectedModule('Fluxo de Caixa, Rake e Extratos'),
    },
    {
      id: 'mesas',
      title: 'MESAS',
      desc: 'Gerencie mesas e partidas',
      icon: Layers,
      color: 'from-red-900 to-amber-950',
      action: () => navigate('/table/lobby'),
    },
    {
      id: 'estatisticas',
      title: 'ESTATÍSTICAS',
      desc: 'Desempenho e relatórios',
      icon: TrendingUp,
      color: 'from-cyan-900 to-blue-950',
      action: () => setSelectedModule('Métricas Globais e Análise de Mãos'),
    },
    {
      id: 'comunicados',
      title: 'COMUNICADOS',
      desc: 'Avisos e comunicados para o clube',
      icon: Megaphone,
      color: 'from-emerald-800 to-teal-950',
      action: () => setSelectedModule('Publicação de Avisos e Notificações'),
    },
    {
      id: 'configuracoes',
      title: 'CONFIGURAÇÕES',
      desc: 'Ajustes e preferências do sistema',
      icon: Settings,
      color: 'from-zinc-700 to-zinc-900',
      action: () => setSelectedModule('Configurações Gerais do Sistema'),
    },
  ];

  return (
    <div className="min-h-screen flex flex-col justify-between p-4 max-w-5xl mx-auto py-6">
      {/* Topo do Painel */}
      <div className="space-y-4">
        <HeaderLogo />

        <div className="flex items-center justify-center space-x-3 text-center pt-2">
          <span className="text-[#d4af37]">◆</span>
          <h2 className="text-lg md:text-xl font-extrabold tracking-widest text-[#f5d77f] uppercase drop-shadow">
            ACESSO AOS MÓDULOS (PAINEL GERENCIAL)
          </h2>
          <span className="text-[#d4af37]">◆</span>
        </div>

        {/* Modal de Módulo Simulado */}
        {selectedModule && (
          <div className="p-4 bg-zinc-900/95 border border-[#d4af37]/60 rounded-xl text-center space-y-2">
            <p className="text-xs text-[#d4af37] uppercase font-bold tracking-wider">Módulo Selecionado:</p>
            <p className="text-white text-base font-semibold">{selectedModule}</p>
            <button
              onClick={() => setSelectedModule(null)}
              className="mt-2 px-4 py-1.5 bg-[#d4af37] text-black text-xs font-bold rounded uppercase cursor-pointer"
            >
              Fechar Módulo
            </button>
          </div>
        )}

        {/* Grid de 8 Módulos */}
        <div className="grid grid-cols-2 sm:grid-cols-4 gap-3.5 sm:gap-5 pt-2">
          {modules.map((mod) => {
            const Icon = mod.icon;
            return (
              <button
                key={mod.id}
                onClick={mod.action}
                className="module-card rounded-xl p-4 sm:p-5 flex flex-col items-center justify-center text-center space-y-2.5 cursor-pointer group"
              >
                <div
                  className={`w-14 h-14 sm:w-16 sm:h-16 rounded-xl bg-gradient-to-br ${mod.color} border border-[#d4af37]/40 flex items-center justify-center shadow-lg group-hover:scale-110 transition`}
                >
                  <Icon className="w-7 h-7 sm:w-8 sm:h-8 text-[#f5d77f]" />
                </div>
                <div>
                  <h3 className="text-xs sm:text-sm font-extrabold tracking-wider text-[#f5d77f] uppercase">
                    {mod.title}
                  </h3>
                  <p className="text-[10px] sm:text-[11px] text-zinc-400 font-medium line-clamp-2 mt-0.5">
                    {mod.desc}
                  </p>
                </div>
              </button>
            );
          })}
        </div>
      </div>

      {/* Banner de Boas-Vindas */}
      <div className="my-6 p-4 rounded-xl bg-gradient-to-r from-amber-950/40 via-zinc-900/80 to-amber-950/40 border border-[#d4af37]/30 flex items-center justify-between">
        <div className="flex items-center space-x-3">
          <div className="p-2 rounded-lg bg-[#d4af37]/20 border border-[#d4af37]/40">
            <ShieldCheck className="w-6 h-6 text-[#f5d77f]" />
          </div>
          <div>
            <h4 className="text-sm font-bold text-[#f5d77f]">
              Bem-vindo, {user?.nome_completo || 'Gerente'}!
            </h4>
            <p className="text-xs text-zinc-400">Acesse o módulo desejado para administrar o clube</p>
          </div>
        </div>
        <div className="hidden sm:flex items-center space-x-2 text-red-500 font-bold text-lg">
          <span>♠</span>
          <span className="text-zinc-200">♥</span>
          <span className="text-[#d4af37]">♦</span>
          <span className="text-zinc-200">♣</span>
        </div>
      </div>

      {/* Barra Inferior Fixa/Estilizada */}
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-3 pt-2">
        {/* Status do Usuário */}
        <div className="bg-[#12151c] border border-[#d4af37]/30 rounded-xl px-4 py-2.5 flex items-center space-x-3">
          <div className="w-8 h-8 rounded-full bg-[#d4af37]/20 border border-[#d4af37] flex items-center justify-center text-[#f5d77f] font-bold text-xs">
            {user?.nome_completo?.charAt(0) || 'G'}
          </div>
          <div>
            <p className="text-xs font-bold text-zinc-200">{user?.nome_completo || 'Gerente'}</p>
            <p className="text-[10px] text-[#d4af37] font-semibold flex items-center gap-1">
              👑 GERENTE / MEMBRO DO CLUBE
            </p>
          </div>
        </div>

        {/* Data e Hora */}
        <div className="bg-[#12151c] border border-[#d4af37]/30 rounded-xl px-4 py-2.5 flex items-center justify-center space-x-2 text-zinc-300">
          <Clock className="w-4 h-4 text-[#d4af37]" />
          <span className="text-xs font-mono font-bold tracking-wider">{currentDateTime}</span>
        </div>

        {/* Botão Sair */}
        <button
          onClick={handleLogout}
          className="bg-gradient-to-r from-red-950 to-red-900 hover:from-red-900 hover:to-red-800 border border-red-500/50 rounded-xl px-4 py-2.5 flex items-center justify-center space-x-2 text-red-200 hover:text-white font-bold text-xs uppercase tracking-wider transition cursor-pointer"
        >
          <LogOut className="w-4 h-4" />
          <span>SAIR DO SISTEMA</span>
        </button>
      </div>
    </div>
  );
};

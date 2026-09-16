import React, { useState } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import { User, Phone, Mail, Calendar, MapPin, ArrowLeft } from 'lucide-react';
import { HeaderLogo } from '../components/common/HeaderLogo';
import { api } from '../services/api';
import { useAuth } from '../context/AuthContext';

export const RegisterPage: React.FC = () => {
  const [nomeCompleto, setNomeCompleto] = useState('');
  const [telefone, setTelefone] = useState('');
  const [email, setEmail] = useState('');
  const senha = 'poker123';
  const [dataNasc, setDataNasc] = useState('');
  const [cidadeEstado, setCidadeEstado] = useState('');
  const [aceitouTermos, setAceitouTermos] = useState(false);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const { login } = useAuth();
  const navigate = useNavigate();

  const handleRegister = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!aceitouTermos) {
      setError('Você deve aceitar o regulamento do clube e confirmar maioridade.');
      return;
    }

    setError('');
    setLoading(true);

    try {
      const data = await api.register({
        nome_completo: nomeCompleto,
        telefone,
        email,
        senha,
        data_nascimento: dataNasc,
        cidade_estado: cidadeEstado,
        aceitou_termos: aceitouTermos,
      });

      login(data.token, data.user);
      navigate('/player');
    } catch (err: any) {
      setError(err.message || 'Falha ao registrar inscrição.');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center p-4 py-8">
      <div className="w-full max-w-lg poker-card-frame rounded-2xl p-6 sm:p-8 space-y-5 relative">
        <Link
          to="/"
          className="inline-flex items-center space-x-1 text-xs text-zinc-400 hover:text-[#d4af37] transition"
        >
          <ArrowLeft size={14} />
          <span>VOLTAR AO LOGIN</span>
        </Link>

        {/* Brasão Superior */}
        <HeaderLogo />

        <div className="flex items-center justify-center space-x-2 text-center">
          <span className="text-[#d4af37] text-xs">◆</span>
          <h2 className="text-base sm:text-lg font-bold tracking-widest text-[#f0cc66] uppercase">
            FORMULÁRIO DE ENTRADA
          </h2>
          <span className="text-[#d4af37] text-xs">◆</span>
        </div>

        {error && (
          <div className="bg-red-950/80 border border-red-500/50 text-red-200 text-sm px-4 py-2.5 rounded-lg text-center">
            {error}
          </div>
        )}

        <form onSubmit={handleRegister} className="space-y-3.5">
          {/* Nome Completo */}
          <div className="space-y-1">
            <label className="text-[11px] font-semibold tracking-wider text-[#d4af37] uppercase flex items-center gap-1.5">
              <User size={13} className="text-[#d4af37]" />
              NOME COMPLETO
            </label>
            <input
              type="text"
              value={nomeCompleto}
              onChange={(e) => setNomeCompleto(e.target.value)}
              placeholder="Digite seu nome completo"
              required
              className="w-full bg-[#12141a]/90 border border-[#d4af37]/40 rounded-lg px-3.5 py-2 text-sm text-white placeholder-zinc-600 focus:outline-none focus:border-[#f0cc66]"
            />
          </div>

          {/* Telefone / WhatsApp */}
          <div className="space-y-1">
            <label className="text-[11px] font-semibold tracking-wider text-[#d4af37] uppercase flex items-center gap-1.5">
              <Phone size={13} className="text-[#d4af37]" />
              TELEFONE / WHATSAPP
            </label>
            <input
              type="text"
              value={telefone}
              onChange={(e) => setTelefone(e.target.value)}
              placeholder="(00) 00000-0000"
              required
              className="w-full bg-[#12141a]/90 border border-[#d4af37]/40 rounded-lg px-3.5 py-2 text-sm text-white placeholder-zinc-600 focus:outline-none focus:border-[#f0cc66]"
            />
          </div>

          {/* E-mail */}
          <div className="space-y-1">
            <label className="text-[11px] font-semibold tracking-wider text-[#d4af37] uppercase flex items-center gap-1.5">
              <Mail size={13} className="text-[#d4af37]" />
              E-MAIL
            </label>
            <input
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder="seuemail@exemplo.com"
              required
              className="w-full bg-[#12141a]/90 border border-[#d4af37]/40 rounded-lg px-3.5 py-2 text-sm text-white placeholder-zinc-600 focus:outline-none focus:border-[#f0cc66]"
            />
          </div>

          {/* Data de Nascimento */}
          <div className="space-y-1">
            <label className="text-[11px] font-semibold tracking-wider text-[#d4af37] uppercase flex items-center gap-1.5">
              <Calendar size={13} className="text-[#d4af37]" />
              DATA DE NASCIMENTO
            </label>
            <input
              type="text"
              value={dataNasc}
              onChange={(e) => setDataNasc(e.target.value)}
              placeholder="DD / MM / AAAA"
              required
              className="w-full bg-[#12141a]/90 border border-[#d4af37]/40 rounded-lg px-3.5 py-2 text-sm text-white placeholder-zinc-600 focus:outline-none focus:border-[#f0cc66]"
            />
          </div>

          {/* Cidade / Estado */}
          <div className="space-y-1">
            <label className="text-[11px] font-semibold tracking-wider text-[#d4af37] uppercase flex items-center gap-1.5">
              <MapPin size={13} className="text-[#d4af37]" />
              CIDADE / ESTADO
            </label>
            <input
              type="text"
              value={cidadeEstado}
              onChange={(e) => setCidadeEstado(e.target.value)}
              placeholder="Ex: São Paulo / SP"
              required
              className="w-full bg-[#12141a]/90 border border-[#d4af37]/40 rounded-lg px-3.5 py-2 text-sm text-white placeholder-zinc-600 focus:outline-none focus:border-[#f0cc66]"
            />
          </div>

          {/* Termos e Declaração +18 */}
          <div className="pt-2">
            <label className="flex items-start space-x-2.5 cursor-pointer">
              <input
                type="checkbox"
                checked={aceitouTermos}
                onChange={(e) => setAceitouTermos(e.target.checked)}
                className="mt-0.5 w-4 h-4 rounded border-zinc-700 bg-zinc-900 text-[#d4af37] focus:ring-0"
              />
              <span className="text-[10px] sm:text-[11px] text-zinc-300 uppercase leading-relaxed tracking-wider">
                DECLARO QUE LI E ACEITO O REGULAMENTO DO CLUBE. SOU MAIOR DE 18 ANOS.
              </span>
            </label>
          </div>

          {/* Botão Enviar Inscrição */}
          <button
            type="submit"
            disabled={loading}
            className="w-full py-3 rounded-lg gold-btn text-black font-extrabold text-sm tracking-widest uppercase transition mt-4 cursor-pointer disabled:opacity-50"
          >
            {loading ? 'ENVIANDO...' : 'ENVIAR INSCRIÇÃO'}
          </button>
        </form>

        {/* Naipes Decorativos */}
        <div className="flex items-center justify-center space-x-4 text-xs pt-1 select-none">
          <span className="text-red-500">♥</span>
          <span className="text-[#d4af37]">♦</span>
          <span className="text-zinc-300">♠</span>
          <span className="text-[#d4af37]">♣</span>
        </div>
      </div>
    </div>
  );
};

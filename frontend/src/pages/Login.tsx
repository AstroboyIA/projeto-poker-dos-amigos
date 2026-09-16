import React, { useState } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import { Mail, Lock, Eye, EyeOff } from 'lucide-react';
import { HeaderLogo } from '../components/common/HeaderLogo';
import { api } from '../services/api';
import { useAuth } from '../context/AuthContext';

export const LoginPage: React.FC = () => {
  const [email, setEmail] = useState('jonatas@pokerdosamigos.com');
  const [senha, setSenha] = useState('poker123');
  const [showPassword, setShowPassword] = useState(false);
  const [lembrarMe, setLembrarMe] = useState(true);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const { login } = useAuth();
  const navigate = useNavigate();

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setLoading(true);

    try {
      const data = await api.login(email, senha, lembrarMe);
      login(data.token, data.user);
      if (data.user.role === 'admin_gerente' || data.user.role === 'gerente') {
        navigate('/manager');
      } else {
        navigate('/player');
      }
    } catch (err: any) {
      setError(err.message || 'Falha na autenticação');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center p-4">
      <div className="w-full max-w-md poker-card-frame rounded-2xl p-6 sm:p-8 space-y-6 relative overflow-hidden">
        {/* Brasão Superior */}
        <HeaderLogo />

        <div className="flex items-center justify-center space-x-2 text-center">
          <span className="text-[#d4af37] text-xs">◆</span>
          <h2 className="text-xl font-bold tracking-widest text-[#f0cc66] uppercase">LOGIN</h2>
          <span className="text-[#d4af37] text-xs">◆</span>
        </div>

        {error && (
          <div className="bg-red-950/80 border border-red-500/50 text-red-200 text-sm px-4 py-2.5 rounded-lg text-center">
            {error}
          </div>
        )}

        {/* Formulário */}
        <form onSubmit={handleLogin} className="space-y-4">
          {/* Campo E-mail */}
          <div className="space-y-1">
            <label className="text-[11px] font-semibold tracking-wider text-[#d4af37] uppercase flex items-center gap-1.5">
              <Mail size={14} className="text-[#d4af37]" />
              E-MAIL
            </label>
            <div className="relative">
              <input
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                placeholder="Digite seu e-mail"
                required
                className="w-full bg-[#12141a]/90 border border-[#d4af37]/40 rounded-lg px-3.5 py-2.5 text-sm text-white placeholder-zinc-500 focus:outline-none focus:border-[#f0cc66] focus:ring-1 focus:ring-[#f0cc66] transition"
              />
            </div>
          </div>

          {/* Campo Senha */}
          <div className="space-y-1">
            <label className="text-[11px] font-semibold tracking-wider text-[#d4af37] uppercase flex items-center gap-1.5">
              <Lock size={14} className="text-[#d4af37]" />
              SENHA
            </label>
            <div className="relative">
              <input
                type={showPassword ? 'text' : 'password'}
                value={senha}
                onChange={(e) => setSenha(e.target.value)}
                placeholder="Digite sua senha"
                required
                className="w-full bg-[#12141a]/90 border border-[#d4af37]/40 rounded-lg px-3.5 py-2.5 text-sm text-white placeholder-zinc-500 focus:outline-none focus:border-[#f0cc66] focus:ring-1 focus:ring-[#f0cc66] transition pr-10"
              />
              <button
                type="button"
                onClick={() => setShowPassword(!showPassword)}
                className="absolute right-3 top-1/2 -translate-y-1/2 text-zinc-400 hover:text-[#d4af37] transition"
              >
                {showPassword ? <EyeOff size={16} /> : <Eye size={16} />}
              </button>
            </div>
          </div>

          {/* Opções extras: Lembrar-me e Esqueci a Senha */}
          <div className="flex items-center justify-between text-xs text-zinc-400 pt-1">
            <label className="flex items-center space-x-2 cursor-pointer hover:text-zinc-200">
              <input
                type="checkbox"
                checked={lembrarMe}
                onChange={(e) => setLembrarMe(e.target.checked)}
                className="w-4 h-4 rounded border-zinc-700 bg-zinc-900 text-[#d4af37] focus:ring-0 focus:ring-offset-0"
              />
              <span className="tracking-wide uppercase text-[11px]">LEMBRAR-ME</span>
            </label>
            <button
              type="button"
              onClick={() => alert('Recuperação de senha enviada para seu e-mail cadastrado.')}
              className="hover:text-[#f0cc66] tracking-wide uppercase text-[11px] transition underline-offset-2 hover:underline"
            >
              ESQUECI MINHA SENHA
            </button>
          </div>

          {/* Botão Entrar */}
          <button
            type="submit"
            disabled={loading}
            className="w-full py-3 rounded-lg gold-btn text-black font-extrabold text-sm tracking-widest uppercase transition mt-2 cursor-pointer disabled:opacity-50"
          >
            {loading ? 'AUTENTICANDO...' : 'ENTRAR'}
          </button>
        </form>

        {/* Divisor OU */}
        <div className="relative flex py-2 items-center">
          <div className="flex-grow border-t border-zinc-800"></div>
          <span className="flex-shrink mx-4 text-zinc-500 text-xs uppercase tracking-widest">OU</span>
          <div className="flex-grow border-t border-zinc-800"></div>
        </div>

        {/* Botões Sociais */}
        <div className="space-y-2.5">
          <button
            type="button"
            onClick={() => alert('Login com Google integrado via OAuth2.')}
            className="w-full flex items-center justify-center space-x-3 py-2.5 px-4 rounded-lg bg-[#151821] border border-zinc-800 hover:border-[#d4af37]/50 text-xs font-semibold tracking-wider text-zinc-200 uppercase transition cursor-pointer"
          >
            <span className="font-bold text-red-500">G</span>
            <span>ENTRAR COM GOOGLE</span>
          </button>
          <button
            type="button"
            onClick={() => alert('Login com Facebook integrado via OAuth2.')}
            className="w-full flex items-center justify-center space-x-3 py-2.5 px-4 rounded-lg bg-[#151821] border border-zinc-800 hover:border-[#d4af37]/50 text-xs font-semibold tracking-wider text-zinc-200 uppercase transition cursor-pointer"
          >
            <span className="font-bold text-blue-500">f</span>
            <span>ENTRAR COM FACEBOOK</span>
          </button>
        </div>

        {/* Link para Cadastro */}
        <div className="text-center pt-2 border-t border-zinc-900">
          <p className="text-xs text-zinc-400">
            NÃO TEM UMA CONTA?{' '}
            <Link
              to="/register"
              className="text-[#d4af37] font-bold hover:text-[#f0cc66] uppercase tracking-wider ml-1 underline-offset-2 hover:underline"
            >
              CADASTRE-SE
            </Link>
          </p>
        </div>

        {/* Naipes Decorativos no Rodapé */}
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

import React, { useEffect, useState } from 'react';
import {
  ArrowLeft,
  Bell,
  Gamepad2,
  KeyRound,
  Palette,
  Save,
  Settings2,
  Shield,
  UserRound,
  Volume2,
} from 'lucide-react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';
import { api } from '../services/api';

const settingsTopics = [
  { id: 'account', label: 'Conta', description: 'Dados e segurança', icon: UserRound },
  { id: 'sounds', label: 'Sons', description: 'Áudio e efeitos', icon: Volume2 },
  { id: 'themes', label: 'Temas', description: 'Aparência da aplicação', icon: Palette },
  { id: 'table', label: 'Mesa de jogo', description: 'Preferências de jogo', icon: Gamepad2 },
  { id: 'notifications', label: 'Notificações', description: 'Avisos e alertas', icon: Bell },
  { id: 'privacy', label: 'Privacidade', description: 'Sessão e dados', icon: Shield },
] as const;

const formatPhone = (value: string) => {
  const digits = value.replace(/\D/g, '').slice(0, 11);
  if (digits.length <= 2) return digits ? `(${digits}` : '';

  const areaCode = digits.slice(0, 2);
  const localNumber = digits.slice(2);
  const prefixLength = digits.length > 10 ? 5 : 4;
  const prefix = localNumber.slice(0, prefixLength);
  const suffix = localNumber.slice(prefixLength, prefixLength + 4);
  return `(${areaCode}) ${prefix}${suffix ? `-${suffix}` : ''}`;
};

const formatBirthDate = (value: string) => {
  const digits = value.replace(/\D/g, '').slice(0, 8);
  if (digits.length <= 2) return digits;
  if (digits.length <= 4) return `${digits.slice(0, 2)}/${digits.slice(2)}`;
  return `${digits.slice(0, 2)}/${digits.slice(2, 4)}/${digits.slice(4)}`;
};

const isValidBirthDate = (value: string) => {
  if (!/^\d{2}\/\d{2}\/\d{4}$/.test(value)) return false;
  const [day, month, year] = value.split('/').map(Number);
  const date = new Date(year, month - 1, day);
  const today = new Date();
  today.setHours(0, 0, 0, 0);
  return date.getFullYear() === year && date.getMonth() === month - 1 &&
    date.getDate() === day && date <= today;
};

interface BrazilianState {
  id: number;
  sigla: string;
  nome: string;
}

interface BrazilianCity {
  id: number;
  nome: string;
}

const parseCityState = (value: string) => {
  const match = value.match(/^(.*?)\s*\/\s*([A-Z]{2})$/);
  return match ? { city: match[1].trim(), state: match[2] } : { city: '', state: '' };
};

export const PlayerSettingsPage: React.FC = () => {
  const { user, token, updateSession } = useAuth();
  const navigate = useNavigate();
  const savedLocation = parseCityState(user?.cidade_estado ?? '');
  const [activeTopic, setActiveTopic] = useState<string>('account');
  const [nomeCompleto, setNomeCompleto] = useState(user?.nome_completo ?? '');
  const [telefone, setTelefone] = useState(user?.telefone ?? '');
  const [dataNascimento, setDataNascimento] = useState(user?.data_nascimento ?? '');
  const [selectedState, setSelectedState] = useState(savedLocation.state);
  const [selectedCity, setSelectedCity] = useState(savedLocation.city);
  const [states, setStates] = useState<BrazilianState[]>([]);
  const [cities, setCities] = useState<BrazilianCity[]>([]);
  const [loadingStates, setLoadingStates] = useState(true);
  const [loadingCities, setLoadingCities] = useState(Boolean(savedLocation.state));
  const [locationError, setLocationError] = useState('');
  const [senhaAtual, setSenhaAtual] = useState('');
  const [novaSenha, setNovaSenha] = useState('');
  const [confirmarSenha, setConfirmarSenha] = useState('');
  const [saving, setSaving] = useState(false);
  const [changingPassword, setChangingPassword] = useState(false);
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');
  const [passwordError, setPasswordError] = useState('');
  const [passwordSuccess, setPasswordSuccess] = useState('');

  useEffect(() => {
    const controller = new AbortController();
    const loadStates = async () => {
      try {
        const response = await fetch('https://servicodados.ibge.gov.br/api/v1/localidades/estados?orderBy=nome', {
          signal: controller.signal,
        });
        if (!response.ok) throw new Error('Não foi possível carregar a lista de estados.');
        const data: BrazilianState[] = await response.json();
        setStates(data);
        if (savedLocation.state && !data.some((state) => state.sigla === savedLocation.state)) {
          setSelectedState('');
          setSelectedCity('');
        }
      } catch (err) {
        if (controller.signal.aborted) return;
        setLocationError(err instanceof Error ? err.message : 'Falha ao carregar estados.');
      } finally {
        if (!controller.signal.aborted) setLoadingStates(false);
      }
    };
    void loadStates();
    return () => controller.abort();
  }, [savedLocation.state]);

  useEffect(() => {
    if (!selectedState) return;

    const controller = new AbortController();
    const loadCities = async () => {
      setLocationError('');
      try {
        const response = await fetch(
          `https://servicodados.ibge.gov.br/api/v1/localidades/estados/${selectedState}/municipios?orderBy=nome`,
          { signal: controller.signal },
        );
        if (!response.ok) throw new Error('Não foi possível carregar as cidades deste estado.');
        const data: BrazilianCity[] = await response.json();
        setCities(data);
        setSelectedCity((currentCity) => data.some((city) => city.nome === currentCity) ? currentCity : '');
      } catch (err) {
        if (controller.signal.aborted) return;
        setCities([]);
        setLocationError(err instanceof Error ? err.message : 'Falha ao carregar cidades.');
      } finally {
        if (!controller.signal.aborted) setLoadingCities(false);
      }
    };
    void loadCities();
    return () => controller.abort();
  }, [selectedState]);

  const handleSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (telefone && !/^\(\d{2}\) (?:\d{4}|\d{5})-\d{4}$/.test(telefone)) {
      setError('Informe um telefone válido no formato (00) 00000-0000.');
      setSuccess('');
      return;
    }
    if (dataNascimento && !isValidBirthDate(dataNascimento)) {
      setError('Informe uma data válida no formato DD/MM/AAAA, que não seja futura.');
      setSuccess('');
      return;
    }
    if ((selectedState && !selectedCity) || (!selectedState && selectedCity)) {
      setError('Selecione primeiro o estado e depois a cidade.');
      setSuccess('');
      return;
    }
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
        cidade_estado: selectedCity && selectedState ? `${selectedCity}/${selectedState}` : '',
      });
      updateSession(updatedSession.token, updatedSession.user);
      setSuccess('Dados da conta atualizados.');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Falha ao atualizar os dados da conta.');
    } finally {
      setSaving(false);
    }
  };

  const handlePasswordSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!token) {
      setPasswordError('Sua sessão expirou. Entre novamente para alterar a senha.');
      return;
    }
    if (novaSenha !== confirmarSenha) {
      setPasswordError('A confirmação da nova senha não corresponde.');
      setPasswordSuccess('');
      return;
    }

    setChangingPassword(true);
    setPasswordError('');
    setPasswordSuccess('');
    try {
      await api.changePassword(token, senhaAtual, novaSenha);
      setSenhaAtual('');
      setNovaSenha('');
      setConfirmarSenha('');
      setPasswordSuccess('Senha alterada com sucesso.');
    } catch (err) {
      setPasswordError(err instanceof Error ? err.message : 'Falha ao alterar senha.');
    } finally {
      setChangingPassword(false);
    }
  };

  return (
    <main className="min-h-screen max-w-6xl mx-auto p-6 text-white space-y-6">
      <button
        onClick={() => navigate(user?.role === 'gerente' || user?.role === 'admin_gerente' ? '/manager' : '/player')}
        className="flex items-center gap-2 text-[#f5d77f] text-sm"
      >
        <ArrowLeft size={16} /> Voltar ao painel
      </button>

      <header className="flex items-center gap-3">
        <Settings2 className="text-[#f5d77f]" />
        <div>
          <h1 className="text-2xl font-bold">Configurações</h1>
          <p className="text-sm text-zinc-400">Gerencie sua conta e suas preferências.</p>
        </div>
      </header>

      <div className="grid gap-6 lg:grid-cols-[240px_minmax(0,1fr)]">
        <nav aria-label="Tópicos de configurações" className="space-y-2">
          {settingsTopics.map((topic) => {
            const Icon = topic.icon;
            const selected = activeTopic === topic.id;
            return (
              <button
                key={topic.id}
                type="button"
                onClick={() => setActiveTopic(topic.id)}
                aria-current={selected ? 'page' : undefined}
                className={`flex w-full items-center gap-3 rounded-xl border px-3 py-3 text-left transition ${
                  selected
                    ? 'border-[#d4af37]/60 bg-[#d4af37]/10 text-[#f5d77f]'
                    : 'border-zinc-800 bg-[#12151c] text-zinc-300 hover:border-zinc-600 hover:bg-zinc-900'
                }`}
              >
                <Icon size={18} className={selected ? 'text-[#f5d77f]' : 'text-zinc-400'} />
                <span className="min-w-0">
                  <span className="block text-sm font-semibold">{topic.label}</span>
                  <span className="block text-xs text-zinc-500">{topic.description}</span>
                </span>
              </button>
            );
          })}
        </nav>

        <section aria-live="polite" className="min-w-0">
          {activeTopic === 'account' ? (
            <div className="space-y-5">
              <header>
                <h2 className="text-xl font-bold text-[#f5d77f]">Conta</h2>
                <p className="mt-1 text-sm text-zinc-400">Seus dados pessoais e segurança de acesso.</p>
              </header>

              {error && <p role="alert" className="rounded border border-red-500/50 p-3 text-sm text-red-200">{error}</p>}
              {success && <p role="status" className="rounded border border-emerald-500/50 p-3 text-sm text-emerald-200">{success}</p>}

              <form onSubmit={handleSubmit} className="space-y-5">
                <section className="rounded-xl border border-[#d4af37]/30 bg-[#12151c] p-5 space-y-4">
                  <h3 className="font-bold text-[#f5d77f]">Dados pessoais</h3>
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
                        type="tel"
                        value={telefone}
                        onChange={(event) => setTelefone(formatPhone(event.target.value))}
                        maxLength={15}
                        inputMode="numeric"
                        autoComplete="tel"
                        placeholder="(00) 00000-0000"
                        className="w-full rounded-lg border border-zinc-700 bg-zinc-900 px-3 py-2 text-white outline-none focus:border-[#d4af37]"
                      />
                    </label>
                    <label className="block space-y-1.5">
                      <span className="text-sm text-zinc-300">Data de nascimento</span>
                      <input
                        value={dataNascimento}
                        onChange={(event) => setDataNascimento(formatBirthDate(event.target.value))}
                        maxLength={10}
                        inputMode="numeric"
                        placeholder="DD/MM/AAAA"
                        autoComplete="bday"
                        className="w-full rounded-lg border border-zinc-700 bg-zinc-900 px-3 py-2 text-white outline-none focus:border-[#d4af37]"
                      />
                    </label>
                  </div>
                  {locationError && <p role="alert" className="rounded border border-red-500/50 p-3 text-sm text-red-200">{locationError}</p>}
                  <div className="grid gap-4 sm:grid-cols-2">
                    <label className="block space-y-1.5">
                      <span className="text-sm text-zinc-300">Estado</span>
                      <select
                        value={selectedState}
                        onChange={(event) => {
                          setSelectedState(event.target.value);
                          setSelectedCity('');
                          setCities([]);
                          setLoadingCities(Boolean(event.target.value));
                          setLocationError('');
                        }}
                        disabled={loadingStates}
                        className="w-full rounded-lg border border-zinc-700 bg-zinc-900 px-3 py-2 text-white outline-none focus:border-[#d4af37] disabled:opacity-60"
                      >
                        <option value="">{loadingStates ? 'Carregando estados...' : 'Selecione um estado'}</option>
                        {states.map((state) => (
                          <option key={state.id} value={state.sigla}>{state.nome}</option>
                        ))}
                      </select>
                    </label>
                    <label className="block space-y-1.5">
                      <span className="text-sm text-zinc-300">Cidade</span>
                      <select
                        value={selectedCity}
                        onChange={(event) => setSelectedCity(event.target.value)}
                        disabled={!selectedState || loadingCities || cities.length === 0}
                        className="w-full rounded-lg border border-zinc-700 bg-zinc-900 px-3 py-2 text-white outline-none focus:border-[#d4af37] disabled:opacity-60"
                      >
                        <option value="">
                          {!selectedState ? 'Selecione primeiro o estado' : loadingCities ? 'Carregando cidades...' : 'Selecione uma cidade'}
                        </option>
                        {cities.map((city) => (
                          <option key={city.id} value={city.nome}>{city.nome}</option>
                        ))}
                      </select>
                    </label>
                  </div>
                </section>

                <section className="rounded-xl border border-zinc-800 bg-zinc-900/60 p-5 space-y-3">
                  <h3 className="font-bold text-zinc-200">Identificação da conta</h3>
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

              <form onSubmit={handlePasswordSubmit} className="space-y-5">
                <section className="rounded-xl border border-[#d4af37]/30 bg-[#12151c] p-5 space-y-4">
                  <header className="flex items-center gap-2">
                    <KeyRound size={18} className="text-[#f5d77f]" />
                    <h3 className="font-bold text-[#f5d77f]">Trocar senha</h3>
                  </header>
                  <p className="text-xs text-zinc-400">A nova senha deve ter entre 8 e 72 bytes.</p>
                  {passwordError && <p role="alert" className="rounded border border-red-500/50 p-3 text-sm text-red-200">{passwordError}</p>}
                  {passwordSuccess && <p role="status" className="rounded border border-emerald-500/50 p-3 text-sm text-emerald-200">{passwordSuccess}</p>}
                  <label className="block space-y-1.5">
                    <span className="text-sm text-zinc-300">Senha atual</span>
                    <input
                      type="password"
                      value={senhaAtual}
                      onChange={(event) => setSenhaAtual(event.target.value)}
                      autoComplete="current-password"
                      required
                      className="w-full rounded-lg border border-zinc-700 bg-zinc-900 px-3 py-2 text-white outline-none focus:border-[#d4af37]"
                    />
                  </label>
                  <div className="grid gap-4 sm:grid-cols-2">
                    <label className="block space-y-1.5">
                      <span className="text-sm text-zinc-300">Nova senha</span>
                      <input
                        type="password"
                        value={novaSenha}
                        onChange={(event) => setNovaSenha(event.target.value)}
                        autoComplete="new-password"
                        minLength={8}
                        maxLength={72}
                        required
                        className="w-full rounded-lg border border-zinc-700 bg-zinc-900 px-3 py-2 text-white outline-none focus:border-[#d4af37]"
                      />
                    </label>
                    <label className="block space-y-1.5">
                      <span className="text-sm text-zinc-300">Confirme a nova senha</span>
                      <input
                        type="password"
                        value={confirmarSenha}
                        onChange={(event) => setConfirmarSenha(event.target.value)}
                        autoComplete="new-password"
                        minLength={8}
                        maxLength={72}
                        required
                        className="w-full rounded-lg border border-zinc-700 bg-zinc-900 px-3 py-2 text-white outline-none focus:border-[#d4af37]"
                      />
                    </label>
                  </div>
                </section>
                <button
                  type="submit"
                  disabled={changingPassword}
                  className="flex items-center justify-center gap-2 rounded-lg bg-[#d4af37] px-5 py-2.5 font-bold text-black transition hover:bg-[#f5d77f] disabled:cursor-not-allowed disabled:opacity-60"
                >
                  <KeyRound size={16} />
                  {changingPassword ? 'Alterando senha...' : 'Alterar senha'}
                </button>
              </form>
            </div>
          ) : (
            <SettingsTopicContent topic={activeTopic} />
          )}
        </section>
      </div>
    </main>
  );
};

const topicContent: Record<string, { title: string; description: string; options: string[] }> = {
  sounds: {
    title: 'Sons',
    description: 'Preferências de áudio para ações e eventos durante as partidas.',
    options: ['Sons de ações e fichas', 'Avisos sonoros da sua vez', 'Volume dos efeitos'],
  },
  themes: {
    title: 'Temas',
    description: 'Personalize a aparência da aplicação de acordo com sua preferência.',
    options: ['Tema escuro', 'Paleta de cores da mesa', 'Contraste e legibilidade'],
  },
  table: {
    title: 'Mesa de jogo',
    description: 'Opções para organizar a experiência visual durante o jogo.',
    options: ['Animações das cartas', 'Exibição de confirmações de ação', 'Layout da mesa'],
  },
  notifications: {
    title: 'Notificações',
    description: 'Escolha como deseja receber avisos sobre atividades da mesa e do clube.',
    options: ['Avisos de torneios', 'Lembretes de partida', 'Comunicados do clube'],
  },
  privacy: {
    title: 'Privacidade',
    description: 'Informações e controles relacionados à segurança da sua conta.',
    options: ['Sua sessão é protegida por autenticação', 'Nunca compartilhe sua senha', 'Encerre a sessão em dispositivos compartilhados'],
  },
};

const SettingsTopicContent: React.FC<{ topic: string }> = ({ topic }) => {
  const content = topicContent[topic];
  if (!content) return null;

  return (
    <div className="space-y-5">
      <header>
        <h2 className="text-xl font-bold text-[#f5d77f]">{content.title}</h2>
        <p className="mt-1 text-sm text-zinc-400">{content.description}</p>
      </header>
      <section className="rounded-xl border border-[#d4af37]/30 bg-[#12151c] p-5">
        <p className="mb-4 text-sm font-semibold text-zinc-200">Opções disponíveis</p>
        <div className="space-y-3">
          {content.options.map((option) => (
            <div key={option} className="rounded-lg border border-zinc-800 bg-zinc-900/70 p-4">
              <p className="text-sm text-zinc-300">{option}</p>
            </div>
          ))}
        </div>
        {topic !== 'privacy' && (
          <p className="mt-4 text-xs text-zinc-500">Os controles destas preferências serão disponibilizados em uma próxima etapa.</p>
        )}
      </section>
    </div>
  );
};

import type { AuthResponse, User, Tournament, RankingEntry, PokerTable, Announcement } from '../types';

const API_BASE = (import.meta.env.VITE_API_BASE_URL || '').replace(/\/$/, '') + '/api';

export const api = {
  async login(username: string, senha: string, lembrarMe: boolean): Promise<AuthResponse> {
    const res = await fetch(`${API_BASE}/auth/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username, senha, lembrar_me: lembrarMe }),
    });
    const data = await res.json();
    if (!res.ok) throw new Error(data.error || 'Falha no login');
    return data.data;
  },

  async register(params: {
    username: string;
    nome_completo?: string;
    telefone?: string;
    email?: string;
    senha: string;
    data_nascimento?: string;
    cidade_estado?: string;
    aceitou_termos?: boolean;
  }): Promise<AuthResponse> {
    const res = await fetch(`${API_BASE}/auth/register`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(params),
    });
    const data = await res.json();
    if (!res.ok) throw new Error(data.error || 'Falha no cadastro');
    return data.data;
  },

  async getMe(token: string): Promise<User> {
    const res = await fetch(`${API_BASE}/auth/me`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    const data = await res.json();
    if (!res.ok) throw new Error(data.error || 'Sessão inválida');
    return data.data;
  },

  async getTournaments(token: string): Promise<Tournament[]> {
    const res = await fetch(`${API_BASE}/tournaments`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    const data = await res.json();
    if (!res.ok) throw new Error(data.error || 'Falha ao buscar torneios');
    return data.data || [];
  },

  async createTournament(token: string, params: {
    nome: string;
    buy_in: number;
    max_inscritos: number;
    starting_stack: number;
    blind_interval_min: number;
    small_blind: number;
    big_blind: number;
  }): Promise<Tournament> {
    const res = await fetch(`${API_BASE}/tournaments`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: ['Bearer', token].join(' '),
      },
      body: JSON.stringify(params),
    });
    const data = await res.json();
    if (!res.ok) throw new Error(data.error || 'Falha ao criar torneio');
    return data.data;
  },

  async registerTournament(token: string, tournamentId: string): Promise<{
    tournament: Tournament;
    seat_number: number;
    table_id: string;
  }> {
    const res = await fetch(`${API_BASE}/tournaments/${tournamentId}/register`, {
      method: 'POST',
      headers: { Authorization: ['Bearer', token].join(' ') },
    });
    const data = await res.json();
    if (!res.ok) throw new Error(data.error || 'Falha na inscrição do torneio');
    return data.data;
  },

  async startTournament(token: string, tournamentId: string): Promise<void> {
    const res = await fetch(`${API_BASE}/tournaments/${tournamentId}/start`, {
      method: 'POST',
      headers: { Authorization: ['Bearer', token].join(' ') },
    });
    const data = await res.json();
    if (!res.ok) throw new Error(data.error || 'Falha ao iniciar torneio');
  },

  async getRankings(token: string): Promise<RankingEntry[]> {
    const res = await fetch(`${API_BASE}/rankings`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    const data = await res.json();
    return data.data || [];
  },

  async getTables(token: string): Promise<PokerTable[]> {
    const res = await fetch(`${API_BASE}/tables`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    const data = await res.json();
    if (!res.ok) throw new Error(data.error || 'Falha ao buscar mesas');
    return data.data || [];
  },

  async createTable(
    token: string,
    params: {
      nome: string;
      small_blind: number;
      big_blind: number;
      buy_in_min: number;
      buy_in_max: number;
      max_seats: number;
      occupied_seats: number[];
      password?: string;
    }
  ): Promise<PokerTable> {
    const res = await fetch(`${API_BASE}/tables`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${token}`,
      },
      body: JSON.stringify(params),
    });
    const data = await res.json();
    if (!res.ok) throw new Error(data.error || 'Falha ao criar mesa');
    return data.data;
  },

  async occupySeat(token: string, tableId: string, seatNumber: number): Promise<PokerTable> {
    const res = await fetch(`${API_BASE}/tables/occupy`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${token}`,
      },
      body: JSON.stringify({ table_id: tableId, seat_number: seatNumber }),
    });
    const data = await res.json();
    if (!res.ok) throw new Error(data.error || 'Falha ao ocupar assento');
    return data.data;
  },

  async leaveSeat(token: string, tableId: string, seatNumber: number): Promise<void> {
    const res = await fetch(`${API_BASE}/tables/leave`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${token}`,
      },
      body: JSON.stringify({ table_id: tableId, seat_number: seatNumber }),
    });
    const data = await res.json();
    if (!res.ok) throw new Error(data.error || 'Falha ao desocupar assento');
  },

  async buyIn(token: string, amount: number, tableId?: string, seatNumber?: number): Promise<User> {
    const controller = new AbortController();
    const timeout = window.setTimeout(() => controller.abort(), 15000);
    try {
      const res = await fetch(`${API_BASE}/chips/buy-in`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${token}`,
        'Idempotency-Key': crypto.randomUUID(),
      },
      body: JSON.stringify({ amount, table_id: tableId, seat_number: seatNumber }),
      signal: controller.signal,
      });
      const data = await res.json();
      if (!res.ok) throw new Error(data.error || 'Saldo insuficiente ou falha no buy-in');
      return data.data;
    } catch (error) {
      if (error instanceof DOMException && error.name === 'AbortError') {
        throw new Error('O buy-in demorou mais de 15 segundos. Verifique se o servidor está disponível.');
      }
      throw error;
    } finally {
      window.clearTimeout(timeout);
    }
  },

  async rebuy(token: string, amount: number, tableId: string): Promise<User> {
    const res = await fetch(`${API_BASE}/chips/rebuy`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: ['Bearer', token].join(' '),
        'Idempotency-Key': crypto.randomUUID(),
      },
      body: JSON.stringify({ amount, table_id: tableId }),
    });
    const data = await res.json();
    if (!res.ok) throw new Error(data.error || 'Falha ao recarregar fichas');
    return data.data;
  },

  async getWallet(token: string): Promise<{ wallet: { balance_cents: number; available_cents: number; reserved_cents: number }; ledger: Array<{ type: string; amount_cents: number; created_at: string }> }> {
    const res = await fetch(`${API_BASE}/wallet`, { headers: { Authorization: `Bearer ${token}` } });
    const data = await res.json();
    if (!res.ok) throw new Error(data.error || 'Falha ao buscar carteira');
    return data.data;
  },

  async devDeposit(token: string, amountCents: number): Promise<User> {
    const res = await fetch(`${API_BASE}/dev/wallet/deposit`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
      body: JSON.stringify({ amount_cents: amountCents }),
    });
    const data = await res.json();
    if (!res.ok) throw new Error(data.error || 'Falha no depósito MOCK/DEV');
    return data.data;
  },

  async getAnnouncements(token: string): Promise<Announcement[]> {
    const res = await fetch(`${API_BASE}/announcements`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    const data = await res.json();
    return data.data || [];
  },
};

import type { AuthResponse, User, Tournament, RankingEntry, PokerTable, Announcement } from '../types';

const API_BASE = (import.meta.env.VITE_API_BASE_URL || '').replace(/\/$/, '') + '/api';

export const api = {
  async login(email: string, senha: string, lembrarMe: boolean): Promise<AuthResponse> {
    const res = await fetch(`${API_BASE}/auth/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email, senha, lembrar_me: lembrarMe }),
    });
    const data = await res.json();
    if (!res.ok) throw new Error(data.error || 'Falha no login');
    return data.data;
  },

  async register(params: {
    nome_completo: string;
    telefone: string;
    email: string;
    senha?: string;
    data_nascimento: string;
    cidade_estado: string;
    aceitou_termos: boolean;
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
    return data.data || [];
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
      bot_seats: number[];
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

  async buyIn(token: string, amount: number): Promise<User> {
    const res = await fetch(`${API_BASE}/chips/buy-in`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${token}`,
      },
      body: JSON.stringify({ amount }),
    });
    const data = await res.json();
    if (!res.ok) throw new Error(data.error || 'Saldo insuficiente ou falha no buy-in');
    return data.data;
  },

  async cashOut(token: string, amount: number): Promise<User> {
    const res = await fetch(`${API_BASE}/chips/cash-out`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${token}`,
      },
      body: JSON.stringify({ amount }),
    });
    const data = await res.json();
    if (!res.ok) throw new Error(data.error || 'Falha no cash-out');
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

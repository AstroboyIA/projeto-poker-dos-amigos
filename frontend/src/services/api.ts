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

  async getAnnouncements(token: string): Promise<Announcement[]> {
    const res = await fetch(`${API_BASE}/announcements`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    const data = await res.json();
    return data.data || [];
  },
};

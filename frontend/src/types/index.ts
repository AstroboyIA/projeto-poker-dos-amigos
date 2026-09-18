export type UserRole = 'admin_gerente' | 'gerente' | 'jogador';

export interface User {
  id: string;
  nome_completo: string;
  telefone: string;
  email: string;
  data_nascimento: string;
  cidade_estado: string;
  aceitou_termos: boolean;
  role: UserRole;
  status: string;
  saldo_fichas: number;
}

export interface AuthResponse {
  token: string;
  expires_at: number;
  user: User;
}

export type CardSuit = 'S' | 'H' | 'D' | 'C';

export interface Card {
  value: number;
  suit: CardSuit;
  code: string;
}

export interface PokerTable {
  id: string;
  nome: string;
  tipo: 'cash_game' | 'torneio';
  small_blind: number;
  big_blind: number;
  buy_in_min: number;
  buy_in_max: number;
  max_seats: number;
  status: string;
  current_pot: number;
  bot_seats?: number[];
  occupied_seats?: number[];
  password?: string;
  created_by?: string;
  community_cards?: Card[];
}

export interface Tournament {
  id: string;
  nome: string;
  buy_in: number;
  garantido: number;
  inscritos: number;
  max_inscritos: number;
  data_inicio: string;
  status: string;
  blind_interval_min: number;
}

export interface RankingEntry {
  posicao: number;
  user_id: string;
  nome: string;
  pontos: number;
  torneios_ganhos: number;
  lucro_total: number;
}

export interface Announcement {
  id: string;
  titulo: string;
  conteudo: string;
  autor: string;
  publicado: boolean;
  created_at: string;
}

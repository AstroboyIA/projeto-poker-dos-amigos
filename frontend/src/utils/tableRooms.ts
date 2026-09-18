import type { PokerTable } from '../types';
import { api } from '../services/api';

export type SeatMode = 'bot' | 'open';

export interface TableRoom {
  id: string;
  name: string;
  smallBlind: number;
  bigBlind: number;
  buyInMin: number;
  buyInMax: number;
  maxSeats: number;
  botSeats: number[];
  occupiedSeats: number[];
  password?: string;
  createdBy: string;
  createdAt: string;
}

const STORAGE_KEY = 'poker-dos-amigos.tables';

export const pokerTableToRoom = (table: PokerTable): TableRoom => {
  return {
    id: table.id,
    name: table.nome,
    smallBlind: table.small_blind,
    bigBlind: table.big_blind,
    buyInMin: table.buy_in_min,
    buyInMax: table.buy_in_max,
    maxSeats: table.max_seats,
    botSeats: table.bot_seats || [],
    occupiedSeats: table.occupied_seats || [],
    password: table.password,
    createdBy: table.created_by || 'Clube',
    createdAt: new Date().toISOString(),
  };
};

export const getCachedTableRooms = (): TableRoom[] => {
  try {
    const stored = localStorage.getItem(STORAGE_KEY);
    if (!stored) return [];
    const parsed = JSON.parse(stored);
    return Array.isArray(parsed) ? parsed : [];
  } catch {
    return [];
  }
};

export const saveCachedTableRooms = (tables: TableRoom[]) => {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(tables));
};

export const fetchRemoteTables = async (token?: string | null): Promise<TableRoom[]> => {
  if (!token) return getCachedTableRooms();
  try {
    const serverTables = await api.getTables(token);
    const rooms = serverTables.map(pokerTableToRoom);
    saveCachedTableRooms(rooms);
    return rooms;
  } catch (err) {
    console.warn('Erro ao buscar mesas remotas, usando cache local:', err);
    return getCachedTableRooms();
  }
};

export const createRemoteTable = async (
  token: string,
  table: Omit<TableRoom, 'id' | 'createdAt'>
): Promise<TableRoom> => {
  try {
    const created = await api.createTable(token, {
      nome: table.name,
      small_blind: table.smallBlind,
      big_blind: table.bigBlind,
      buy_in_min: table.buyInMin,
      buy_in_max: table.buyInMax,
      max_seats: table.maxSeats,
      bot_seats: table.botSeats,
      occupied_seats: table.occupiedSeats,
      password: table.password,
    });

    const room = pokerTableToRoom(created);
    const existing = getCachedTableRooms().filter((t) => t.id !== room.id);
    saveCachedTableRooms([room, ...existing]);
    return room;
  } catch (err) {
    console.error('Falha ao criar mesa no backend, fallback local:', err);
    const fallbackRoom: TableRoom = {
      ...table,
      id: `mesa-${Date.now()}`,
      createdAt: new Date().toISOString(),
    };
    saveCachedTableRooms([fallbackRoom, ...getCachedTableRooms()]);
    return fallbackRoom;
  }
};

export const getTableRoomById = (id: string): TableRoom | undefined => {
  return getCachedTableRooms().find((table) => table.id === id);
};

export const countAvailableSeats = (table: TableRoom) => {
  const unavailable = new Set([...table.occupiedSeats, ...table.botSeats]);
  return table.maxSeats - unavailable.size;
};

export const occupyTableSeatRemote = async (token: string | null, tableId: string, seatNumber: number) => {
  if (token) {
    try {
      await api.occupySeat(token, tableId, seatNumber);
    } catch (e) {
      console.warn('Erro ao ocupar assento remoto:', e);
    }
  }
  const tables = getCachedTableRooms();
  const updated = tables.map((table) => {
    if (table.id !== tableId || table.occupiedSeats.includes(seatNumber)) return table;
    return {
      ...table,
      occupiedSeats: [...table.occupiedSeats, seatNumber].sort((a, b) => a - b),
    };
  });
  saveCachedTableRooms(updated);
};

export const leaveTableSeatRemote = async (token: string | null, tableId: string, seatNumber: number) => {
  if (token) {
    try {
      await api.leaveSeat(token, tableId, seatNumber);
    } catch (e) {
      console.warn('Erro ao desocupar assento remoto:', e);
    }
  }
  const updated = getCachedTableRooms().map((table) => {
    if (table.id !== tableId) return table;
    return {
      ...table,
      occupiedSeats: table.occupiedSeats.filter((seat) => seat !== seatNumber),
    };
  });
  saveCachedTableRooms(updated);
};

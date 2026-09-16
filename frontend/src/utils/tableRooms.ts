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

const seedTables: TableRoom[] = [
  {
    id: 'mesa-vip-01',
    name: "Mesa VIP Ouro #01 (Texas Hold'em)",
    smallBlind: 25,
    bigBlind: 50,
    buyInMin: 1000,
    buyInMax: 5000,
    maxSeats: 9,
    botSeats: [3, 5, 7],
    occupiedSeats: [1, 2, 4, 6],
    createdBy: 'Clube',
    createdAt: new Date().toISOString(),
  },
  {
    id: 'mesa-amigos-fechada',
    name: 'Mesa dos Amigos Fechada',
    smallBlind: 10,
    bigBlind: 20,
    buyInMin: 400,
    buyInMax: 2000,
    maxSeats: 9,
    botSeats: [2, 8],
    occupiedSeats: [1, 4, 6],
    password: '1234',
    createdBy: 'Felipe',
    createdAt: new Date().toISOString(),
  },
];

export const getTableRooms = (): TableRoom[] => {
  try {
    const stored = localStorage.getItem(STORAGE_KEY);
    if (!stored) {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(seedTables));
      return seedTables;
    }

    const parsed = JSON.parse(stored);
    const tables = Array.isArray(parsed) ? parsed : seedTables;
    const activeTables = tables.filter((table) => table.occupiedSeats.length > 0);
    if (activeTables.length !== tables.length) {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(activeTables));
    }
    return activeTables;
  } catch {
    return seedTables;
  }
};

export const saveTableRooms = (tables: TableRoom[]) => {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(tables));
};

export const getTableRoomById = (id: string): TableRoom | undefined => {
  return getTableRooms().find((table) => table.id === id);
};

export const createTableRoom = (table: Omit<TableRoom, 'id' | 'createdAt'>): TableRoom => {
  const newTable: TableRoom = {
    ...table,
    id: `mesa-${Date.now()}`,
    createdAt: new Date().toISOString(),
  };

  saveTableRooms([newTable, ...getTableRooms()]);
  return newTable;
};

export const countAvailableSeats = (table: TableRoom) => {
  const unavailable = new Set([...table.occupiedSeats, ...table.botSeats]);
  return table.maxSeats - unavailable.size;
};

export const occupyTableSeat = (tableId: string, seatNumber: number) => {
  const tables = getTableRooms();
  const updated = tables.map((table) => {
    if (table.id !== tableId || table.occupiedSeats.includes(seatNumber)) return table;
    return {
      ...table,
      occupiedSeats: [...table.occupiedSeats, seatNumber].sort((a, b) => a - b),
    };
  });
  saveTableRooms(updated);
};

export const leaveTableSeat = (tableId: string, seatNumber: number) => {
  const updated = getTableRooms()
    .map((table) => {
      if (table.id !== tableId) return table;
      return {
        ...table,
        occupiedSeats: table.occupiedSeats.filter((seat) => seat !== seatNumber),
      };
    })
    .filter((table) => table.occupiedSeats.length > 0);

  saveTableRooms(updated);
};

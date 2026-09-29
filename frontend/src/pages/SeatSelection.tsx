import React, { useEffect, useRef, useState } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { ArrowLeft, UserPlus, CheckCircle2, AlertCircle, Coins, ShieldCheck, Users } from 'lucide-react';
import { HeaderLogo } from '../components/common/HeaderLogo';
import { useAuth } from '../context/AuthContext';
import { getTableRoomById } from '../utils/tableRooms';
import { api } from '../services/api';

interface SeatInfo {
  seatNumber: number;
  isOccupied: boolean;
  playerName?: string;
  playerAvatar?: string;
  playerStack?: number;
  positionName: string;
}

export const SeatSelectionPage: React.FC = () => {
  const { user, token, updateUser } = useAuth();
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const tableId = searchParams.get('tableId') || 'mesa-vip-01';
  const autoJoin = searchParams.get('autoJoin') === '1';
  const room = getTableRoomById(tableId);

  // Configurações da Mesa
  const tableName = `${room?.name || "Mesa VIP Ouro #01 (Texas Hold'em)"} (9-Max)`;
  const blinds = `${room?.smallBlind || 25} / ${room?.bigBlind || 50}`;
  const buyInMin = room?.buyInMin || 1000;
  const buyInMax = room?.buyInMax || 5000;
  const defaultBuyIn = Math.min(Math.max(2500, buyInMin), buyInMax);

  const [selectedSeat, setSelectedSeat] = useState<number | null>(autoJoin ? 1 : null);
  const [buyInAmount, setBuyInAmount] = useState<number>(defaultBuyIn);
  const [autoPostBlinds, setAutoPostBlinds] = useState<boolean>(true);
  const [autoRebuy, setAutoRebuy] = useState<boolean>(false);
  const [entering, setEntering] = useState(false);
  const autoJoinStarted = useRef(false);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);
  const occupiedNames: Record<number, string> = {
    1: 'Jonatas (Sócio)',
    2: 'Felipe (Sócio)',
    4: "Bruno 'AllIn'",
    6: 'Carlos Shark',
  };

  const positionNames = [
    'Dealer / Botão',
    'Small Blind',
    'Big Blind',
    'Under the Gun (UTG)',
    'UTG+1',
    'Middle Position (MP)',
    'Lowjack (LJ)',
    'Hijack (HJ)',
    'Cut-Off (CO)',
  ];

  const seats: SeatInfo[] = Array.from({ length: 9 }, (_, idx) => {
    const seatNumber = idx + 1;
    const isOccupied = room?.occupiedSeats.includes(seatNumber) ?? [1, 2, 4, 6].includes(seatNumber);
    return {
      seatNumber,
      isOccupied,
      playerName: isOccupied ? occupiedNames[seatNumber] || 'Jogador sentado' : undefined,
      playerStack: isOccupied ? [4850, 5200, 3400, 2100, 2800, 6100, 4200, 3900, 2600][idx] : undefined,
      positionName: `${positionNames[idx]} ${isOccupied ? '(Ocupado)' : '(Vago)'}`,
    };
  });

  const handleSeatClick = (seat: SeatInfo) => {
    if (seat.isOccupied) return;
    setSelectedSeat(seat.seatNumber);
  };

  const handleConfirmAndEnter = async () => {
    if (!selectedSeat || entering) return;
    setErrorMessage(null);

    // Validação de saldo de fichas
    if (user && user.saldo_fichas < buyInAmount) {
      setErrorMessage(`Saldo insuficiente (${user.saldo_fichas} fichas). O buy-in escolhido é de ${buyInAmount} fichas.`);
      return;
    }

    setEntering(true);
    try {
      if (token) {
        const updatedUser = await api.buyIn(token, buyInAmount, tableId, selectedSeat);
        updateUser(updatedUser);
      } else if (user) {
      }

      navigate(`/table/live?tableId=${tableId}&seat=${selectedSeat}&buyIn=${buyInAmount}`);
    } catch (err: any) {
      setErrorMessage(err.message || 'Falha ao realizar buy-in ou entrar na mesa');
      setEntering(false);
    }
  };

  useEffect(() => {
    if (autoJoin && selectedSeat === 1 && !entering && !autoJoinStarted.current) {
      autoJoinStarted.current = true;
      void handleConfirmAndEnter();
    }
  }, [autoJoin, selectedSeat]);

  const openSeatsCount = seats.filter((seat) => !seat.isOccupied).length;

  return (
    <div className="min-h-screen bg-[#07090e] text-white flex flex-col justify-between p-3 sm:p-6 max-w-6xl mx-auto">
      {/* Header Superior */}
      <div className="space-y-3">
        <div className="flex items-center justify-between border-b border-[#d4af37]/30 pb-3">
          <button
            onClick={() => navigate(-1)}
            className="flex items-center space-x-2 px-3 py-1.5 rounded-lg bg-zinc-900 border border-[#d4af37]/40 hover:bg-zinc-800 text-[#d4af37] text-xs font-bold transition cursor-pointer"
          >
            <ArrowLeft size={16} />
            <span>VOLTAR AO LOBBY</span>
          </button>

          <div className="flex items-center space-x-3 text-right">
            <div>
              <p className="text-xs text-zinc-400">Saldo Disponível:</p>
              <p className="text-sm font-bold text-[#f5d77f] font-mono">
                ${user?.saldo_fichas?.toLocaleString('pt-BR') || '10.000'} Fichas
              </p>
            </div>
            <div className="w-9 h-9 rounded-full bg-[#d4af37]/20 border border-[#d4af37] flex items-center justify-center text-[#f5d77f] font-bold text-sm">
              {user?.nome_completo?.charAt(0) || 'J'}
            </div>
          </div>
        </div>

        {/* Título e Info da Mesa */}
        <div className="text-center space-y-1">
          <HeaderLogo />
          <div className="pt-2">
            <h1 className="text-lg sm:text-2xl font-extrabold tracking-wider text-[#f5d77f] uppercase drop-shadow">
              {tableName}
            </h1>
            <p className="text-xs sm:text-sm text-zinc-400">
              Blinds: <span className="text-white font-bold">${blinds}</span> • Buy-in:{' '}
              <span className="text-white font-bold">${buyInMin} - ${buyInMax}</span> • Capacidade: <span className="text-[#f5d77f] font-bold">9 Assentos (Full Ring)</span>
            </p>
          </div>
        </div>
      </div>

      {/* Seção Central: Preview da Mesa com 9 Assentos Interativos */}
      <div className="my-4">
        <div className="text-center mb-2">
          <p className="text-xs uppercase tracking-widest text-[#d4af37] font-bold flex items-center justify-center gap-1.5">
            <Users size={15} />
            SELECIONE UM DOS {openSeatsCount} ASSENTOS LIVRES (MESA DE 9 JOGADORES)
          </p>
        </div>

        {/* Mesa Oval Visual 9-Max */}
        <div className="relative w-full max-w-4xl h-[420px] sm:h-[470px] mx-auto rounded-[200px] sm:rounded-[240px] poker-felt border-[14px] sm:border-[18px] border-[#181109] shadow-[0_0_50px_rgba(0,0,0,0.9),inset_0_0_60px_rgba(0,0,0,0.8)] flex flex-col items-center justify-center p-4 select-none">
          {/* Logo e Info no Centro do Feltro */}
          <div className="text-center opacity-70 pointer-events-none space-y-0.5">
            <div className="text-[#d4af37] text-sm sm:text-base font-black tracking-widest uppercase">
              TEXAS HOLD'EM 9-MAX
            </div>
            <div className="text-[10px] text-zinc-300 tracking-wider">
              {openSeatsCount} assentos livres
            </div>
          </div>

          {/* Topo: 3 Assentos (1, 2, 3) */}
          <div className="absolute top-2 left-[20%] -translate-x-1/2">
            <SeatButton seat={seats[0]} isSelected={selectedSeat === 1} onClick={() => handleSeatClick(seats[0])} />
          </div>
          <div className="absolute top-2 left-1/2 -translate-x-1/2">
            <SeatButton seat={seats[1]} isSelected={selectedSeat === 2} onClick={() => handleSeatClick(seats[1])} />
          </div>
          <div className="absolute top-2 right-[20%] translate-x-1/2">
            <SeatButton seat={seats[2]} isSelected={selectedSeat === 3} onClick={() => handleSeatClick(seats[2])} />
          </div>

          {/* Direita: 2 Assentos (4, 5) */}
          <div className="absolute top-[32%] right-2 -translate-y-1/2">
            <SeatButton seat={seats[3]} isSelected={selectedSeat === 4} onClick={() => handleSeatClick(seats[3])} />
          </div>
          <div className="absolute top-[68%] right-2 -translate-y-1/2">
            <SeatButton seat={seats[4]} isSelected={selectedSeat === 5} onClick={() => handleSeatClick(seats[4])} />
          </div>

          {/* Fundo: 2 Assentos (6, 7) */}
          <div className="absolute bottom-2 right-[32%] translate-x-1/2">
            <SeatButton seat={seats[5]} isSelected={selectedSeat === 6} onClick={() => handleSeatClick(seats[5])} />
          </div>
          <div className="absolute bottom-2 left-[32%] -translate-x-1/2">
            <SeatButton seat={seats[6]} isSelected={selectedSeat === 7} onClick={() => handleSeatClick(seats[6])} />
          </div>

          {/* Esquerda: 2 Assentos (8, 9) */}
          <div className="absolute top-[68%] left-2 -translate-y-1/2">
            <SeatButton seat={seats[7]} isSelected={selectedSeat === 8} onClick={() => handleSeatClick(seats[7])} />
          </div>
          <div className="absolute top-[32%] left-2 -translate-y-1/2">
            <SeatButton seat={seats[8]} isSelected={selectedSeat === 9} onClick={() => handleSeatClick(seats[8])} />
          </div>
        </div>
      </div>

      {/* Painel Inferior: Configuração de Buy-in e Confirmação de Entrada */}
      <div className="poker-card-frame rounded-2xl p-4 sm:p-6 space-y-4 max-w-4xl mx-auto w-full">
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6 items-center">
          {/* Coluna 1: Informações do Assento Escolhido */}
          <div className="space-y-3">
            <div className="flex items-center space-x-2 text-xs font-bold uppercase tracking-wider text-[#d4af37]">
              <ShieldCheck size={16} />
              <span>DETALHES DO SEU ASSENTO</span>
            </div>

            {selectedSeat ? (
              <div className="p-3.5 bg-emerald-950/40 border border-emerald-500/50 rounded-xl flex items-center justify-between">
                <div className="flex items-center space-x-3">
                  <div className="w-10 h-10 rounded-full bg-emerald-600/30 border border-emerald-400 flex items-center justify-center font-bold text-emerald-300 text-sm">
                    #{selectedSeat}
                  </div>
                  <div>
                    <p className="text-sm font-bold text-white">Assento #{selectedSeat} Selecionado (Mesa de 9)</p>
                    <p className="text-xs text-emerald-400">
                      {seats.find((s) => s.seatNumber === selectedSeat)?.positionName}
                    </p>
                  </div>
                </div>
                <CheckCircle2 className="w-6 h-6 text-emerald-400" />
              </div>
            ) : (
              <div className="p-3.5 bg-amber-950/30 border border-amber-600/40 rounded-xl flex items-center space-x-3 text-amber-200 text-xs">
                <AlertCircle className="w-5 h-5 flex-shrink-0 text-amber-400" />
                <span>Clique em um dos assentos livres (1 a 9) destacados em verde para escolher sua posição.</span>
              </div>
            )}

            {/* Opções de Jogo */}
            <div className="space-y-2 pt-1 text-xs text-zinc-300">
              <label className="flex items-center space-x-2 cursor-pointer">
                <input
                  type="checkbox"
                  checked={autoPostBlinds}
                  onChange={(e) => setAutoPostBlinds(e.target.checked)}
                  className="rounded border-zinc-700 bg-zinc-900 text-[#d4af37] focus:ring-0"
                />
                <span>Postar Blinds automaticamente ao entrar</span>
              </label>
              <label className="flex items-center space-x-2 cursor-pointer">
                <input
                  type="checkbox"
                  checked={autoRebuy}
                  onChange={(e) => setAutoRebuy(e.target.checked)}
                  className="rounded border-zinc-700 bg-zinc-900 text-[#d4af37] focus:ring-0"
                />
                <span>Auto Rebuy caso o stack caia abaixo de 20 Big Blinds</span>
              </label>
            </div>

          </div>

          {/* Coluna 2: Configuração de Buy-in */}
          <div className="space-y-3 bg-[#12151d] p-4 rounded-xl border border-[#d4af37]/30">
            <div className="flex items-center justify-between text-xs">
              <span className="font-bold text-[#f5d77f] uppercase tracking-wider flex items-center gap-1.5">
                <Coins size={14} className="text-[#d4af37]" />
                VALOR DO BUY-IN (FICHAS)
              </span>
              <span className="font-mono text-sm font-extrabold text-white">${buyInAmount}</span>
            </div>

            <input
              type="range"
              min={buyInMin}
              max={buyInMax}
              step={100}
              value={buyInAmount}
              onChange={(e) => setBuyInAmount(Number(e.target.value))}
              className="w-full accent-[#d4af37] cursor-pointer"
            />

            <div className="flex justify-between text-[11px] text-zinc-400 font-mono">
              <span>Mínimo: ${buyInMin}</span>
              <span>Médio: $3.000</span>
              <span>Máximo: ${buyInMax}</span>
            </div>

            {errorMessage && (
              <div className="p-2.5 rounded-lg bg-red-950/80 border border-red-500 text-xs text-red-200">
                {errorMessage}
              </div>
            )}

            {/* Botão de Confirmação */}
            <button
              onClick={handleConfirmAndEnter}
              disabled={!selectedSeat || entering}
              className={`w-full py-3.5 rounded-xl font-extrabold text-xs uppercase tracking-widest transition cursor-pointer flex items-center justify-center space-x-2 ${
                selectedSeat && !entering
                  ? 'gold-btn text-black'
                  : 'bg-zinc-800 text-zinc-500 border border-zinc-700 cursor-not-allowed'
              }`}
            >
              <span>
                {entering
                  ? 'PROCESSANDO BUY-IN...'
                  : selectedSeat
                  ? 'CONFIRMAR ASSENTO E ENTRAR NA MESA'
                  : 'ESCOLHA UM ASSENTO (1 A 9) PARA CONTINUAR'}
              </span>
            </button>
          </div>
        </div>
      </div>
    </div>
  );
};

// Subcomponente de Assento Visual
interface SeatButtonProps {
  seat: SeatInfo;
  isSelected: boolean;
  onClick: () => void;
}

const SeatButton: React.FC<SeatButtonProps> = ({ seat, isSelected, onClick }) => {
  if (seat.isOccupied) {
    return (
      <div className="flex flex-col items-center select-none opacity-85">
        <div className="w-11 h-11 sm:w-13 sm:h-13 rounded-full bg-gradient-to-br from-zinc-800 to-zinc-950 border-2 border-zinc-600 flex flex-col items-center justify-center shadow-lg">
          <span className="text-[9px] text-zinc-400 font-bold">#{seat.seatNumber}</span>
          <span className="text-[10px] font-bold text-zinc-200 truncate max-w-[44px]">
            {seat.playerName?.split(' ')[0]}
          </span>
        </div>
        <div className="mt-0.5 bg-black/80 px-1.5 py-0.5 rounded border border-zinc-800 text-[9px] font-mono text-[#f5d77f] font-bold">
          ${seat.playerStack}
        </div>
      </div>
    );
  }

  return (
    <button
      type="button"
      onClick={onClick}
      className={`flex flex-col items-center select-none transition-all duration-200 cursor-pointer group ${
        isSelected ? 'scale-110' : 'hover:scale-105'
      }`}
    >
      <div
        className={`w-11 h-11 sm:w-13 sm:h-13 rounded-full flex flex-col items-center justify-center shadow-lg transition ${
          isSelected
            ? 'bg-gradient-to-br from-yellow-400 to-amber-600 border-2 border-white ring-4 ring-yellow-400/40 text-black'
            : 'bg-emerald-950/90 border-2 border-emerald-400/80 hover:border-[#d4af37] text-emerald-200 group-hover:bg-emerald-900'
        }`}
      >
        <UserPlus size={15} className={isSelected ? 'text-black' : 'text-emerald-300 group-hover:text-[#f5d77f]'} />
        <span className="text-[9px] font-extrabold uppercase mt-0.5">#{seat.seatNumber}</span>
      </div>
      <div
        className={`mt-0.5 px-1.5 py-0.5 rounded text-[9px] font-bold tracking-wider uppercase transition ${
          isSelected
            ? 'bg-yellow-400 text-black font-extrabold shadow'
            : 'bg-black/80 border border-emerald-500/50 text-emerald-300 group-hover:border-[#d4af37] group-hover:text-[#f5d77f]'
        }`}
      >
        {isSelected ? 'ESCOLHIDO' : 'LIVRE'}
      </div>
    </button>
  );
};

import React from 'react';

export type ChipColor = 'white' | 'red' | 'blue' | 'green' | 'black' | 'purple' | 'gold';

interface PokerChipProps {
  value?: number;
  size?: 'sm' | 'md' | 'lg';
  color?: ChipColor;
  className?: string;
}

export const PokerChip: React.FC<PokerChipProps> = ({
  value,
  size = 'md',
  color,
  className = '',
}) => {
  // Determina cor com base no valor se não for explícita
  let chipColor = color;
  if (!chipColor && value !== undefined) {
    if (value >= 1000) chipColor = 'gold';
    else if (value >= 500) chipColor = 'purple';
    else if (value >= 100) chipColor = 'black';
    else if (value >= 25) chipColor = 'green';
    else if (value >= 10) chipColor = 'blue';
    else if (value >= 5) chipColor = 'red';
    else chipColor = 'white';
  }
  if (!chipColor) chipColor = 'gold';

  // Configurações de estilo por cor
  const colorStyles: Record<ChipColor, { bg: string; border: string; accent: string; text: string }> = {
    white: {
      bg: 'from-zinc-100 to-zinc-300',
      border: 'border-zinc-400',
      accent: 'border-red-600',
      text: 'text-zinc-900',
    },
    red: {
      bg: 'from-red-600 to-red-800',
      border: 'border-red-400',
      accent: 'border-white',
      text: 'text-white',
    },
    blue: {
      bg: 'from-blue-600 to-blue-800',
      border: 'border-blue-400',
      accent: 'border-white',
      text: 'text-white',
    },
    green: {
      bg: 'from-emerald-600 to-emerald-800',
      border: 'border-emerald-400',
      accent: 'border-yellow-300',
      text: 'text-white',
    },
    black: {
      bg: 'from-zinc-800 to-black',
      border: 'border-amber-400',
      accent: 'border-amber-300',
      text: 'text-amber-400',
    },
    purple: {
      bg: 'from-purple-700 to-indigo-900',
      border: 'border-purple-300',
      accent: 'border-yellow-400',
      text: 'text-yellow-300',
    },
    gold: {
      bg: 'from-yellow-400 via-amber-500 to-yellow-600',
      border: 'border-yellow-200',
      accent: 'border-black',
      text: 'text-black',
    },
  };

  const currentStyle = colorStyles[chipColor];

  // Dimensões ampliadas e robustas das fichas
  const sizeStyles = {
    sm: 'w-7 h-7 sm:w-8 sm:h-8 text-[9px] sm:text-[10px] border-[2px]',
    md: 'w-9 h-9 sm:w-11 sm:h-11 text-xs sm:text-sm border-[2.5px] shadow-lg',
    lg: 'w-12 h-12 sm:w-14 sm:h-14 text-sm sm:text-base border-[3px] shadow-xl',
  };

  const innerSizeStyles = {
    sm: 'w-4 h-4 sm:w-5 sm:h-5 border-[1px]',
    md: 'w-5 h-5 sm:w-7 sm:h-7 border-[1.5px]',
    lg: 'w-7 h-7 sm:w-9 sm:h-9 border-2',
  };

  return (
    <div
      className={`relative inline-flex items-center justify-center rounded-full bg-gradient-to-br ${currentStyle.bg} ${currentStyle.border} ${sizeStyles[size]} select-none font-black transition-transform duration-200 ${className}`}
      style={{
        boxShadow: '0 4px 8px rgba(0,0,0,0.6), inset 0 1px 2px rgba(255,255,255,0.7)',
      }}
    >
      {/* Detalhes de estrias da borda da ficha de cassino */}
      <div className="absolute inset-0 rounded-full border border-dashed border-white/50 pointer-events-none" />

      {/* Círculo interno com valor */}
      <div
        className={`rounded-full flex items-center justify-center ${currentStyle.accent} ${innerSizeStyles[size]} bg-black/25`}
      >
        {value !== undefined && (
          <span className={`${currentStyle.text} font-mono leading-none tracking-tighter font-extrabold`}>
            {value >= 1000 ? `${value / 1000}k` : value}
          </span>
        )}
      </div>
    </div>
  );
};

// Componente para Pilha de Fichas 3D (Estática sobre a mesa, sem animação de pulo)
interface ChipStackProps {
  amount: number;
  size?: 'sm' | 'md' | 'lg';
  showLabel?: boolean;
  className?: string;
}

export const ChipStack: React.FC<ChipStackProps> = ({
  amount,
  size = 'md',
  showLabel = true,
  className = '',
}) => {
  if (amount <= 0) return null;

  // Decompõe o valor em denominações realistas de fichas
  const getDenominations = (val: number): { value: number; count: number; color: ChipColor }[] => {
    let remaining = val;
    const denoms: { val: number; color: ChipColor }[] = [
      { val: 1000, color: 'gold' },
      { val: 500, color: 'purple' },
      { val: 100, color: 'black' },
      { val: 25, color: 'green' },
      { val: 10, color: 'blue' },
      { val: 5, color: 'red' },
      { val: 1, color: 'white' },
    ];

    const result: { value: number; count: number; color: ChipColor }[] = [];
    for (const d of denoms) {
      if (remaining >= d.val) {
        const count = Math.min(5, Math.floor(remaining / d.val));
        result.push({ value: d.val, count, color: d.color });
        remaining -= count * d.val;
      }
    }
    return result;
  };

  const stacks = getDenominations(amount);

  return (
    <div className={`flex flex-col items-center select-none ${className}`}>
      {/* Pilhas de Fichas Lado a Lado ou Sobrepostas no Feltro */}
      <div className="flex items-end space-x-1 justify-center">
        {stacks.slice(0, 3).map((stack, stackIdx) => (
          <div key={stackIdx} className="relative flex flex-col items-center">
            {Array.from({ length: stack.count }).map((_, chipIdx) => (
              <div
                key={chipIdx}
                className="transition-all duration-200"
                style={{
                  marginTop: chipIdx === 0 ? 0 : size === 'sm' ? -18 : size === 'md' ? -26 : -34,
                  zIndex: chipIdx,
                }}
              >
                <PokerChip
                  value={chipIdx === stack.count - 1 ? stack.value : undefined}
                  size={size}
                  color={stack.color}
                />
              </div>
            ))}
          </div>
        ))}
      </div>

      {/* Badge com Valor Numérico de Fichas */}
      {showLabel && (
        <div className="mt-1.5 px-2.5 py-0.5 rounded-full bg-black/90 border border-[#d4af37] text-[10px] sm:text-[11px] font-mono font-extrabold text-[#f5d77f] shadow-xl flex items-center gap-1 whitespace-nowrap">
          <span>🪙</span>
          <span>${amount.toLocaleString('pt-BR')}</span>
        </div>
      )}
    </div>
  );
};

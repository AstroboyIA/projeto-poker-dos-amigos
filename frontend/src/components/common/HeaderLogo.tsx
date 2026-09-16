import React from 'react';

export const HeaderLogo: React.FC = () => {
  return (
    <div className="flex flex-col items-center justify-center text-center select-none">
      {/* Brasão Dourado Estilizado */}
      <div className="relative w-24 h-24 mb-2 flex items-center justify-center">
        <div className="absolute inset-0 rounded-full border-2 border-[#d4af37]/60 shadow-[0_0_20px_rgba(212,175,55,0.4)] animate-pulse" />
        <div className="relative z-10 text-4xl flex items-center justify-center space-x-1">
          <span className="text-red-500 drop-shadow-[0_2px_4px_rgba(0,0,0,0.8)]">♥</span>
          <span className="text-[#d4af37] text-5xl font-black drop-shadow-[0_2px_8px_rgba(0,0,0,0.9)]">♠</span>
          <span className="text-red-500 drop-shadow-[0_2px_4px_rgba(0,0,0,0.8)]">♦</span>
        </div>
      </div>

      {/* Título do Clube */}
      <div className="space-y-0.5">
        <h1 className="text-2xl md:text-3xl font-extrabold tracking-wider text-[#f5d77f] uppercase drop-shadow-[0_2px_6px_rgba(0,0,0,0.9)]">
          CLUB POKER
        </h1>
        <p className="text-xs md:text-sm font-semibold tracking-widest text-[#d4af37] uppercase">
          DOS AMIGOS
        </p>
        <div className="flex items-center justify-center space-x-2 pt-1 text-[10px] tracking-widest text-[#a89060] uppercase">
          <span>♦</span>
          <span>TEXAS HOLD'EM</span>
          <span>♦</span>
        </div>
      </div>
    </div>
  );
};

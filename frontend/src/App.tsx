import React from 'react';
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { AuthProvider, useAuth } from './context/AuthContext';
import { LoginPage } from './pages/Login';
import { RegisterPage } from './pages/Register';
import { ManagerDashboard } from './pages/ManagerDashboard';
import { PlayerDashboard } from './pages/PlayerDashboard';
import { TableLobbyPage } from './pages/TableLobby';
import { SeatSelectionPage } from './pages/SeatSelection';
import { PokerTablePage } from './pages/PokerTable';

const ProtectedRoute: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const { loading } = useAuth();
  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-[#0b0c10] text-[#d4af37]">
        <div className="text-center space-y-2">
          <div className="w-8 h-8 border-2 border-[#d4af37] border-t-transparent rounded-full animate-spin mx-auto" />
          <p className="text-xs uppercase tracking-widest">Carregando Poker dos Amigos...</p>
        </div>
      </div>
    );
  }
  return <>{children}</>;
};

export function App() {
  return (
    <AuthProvider>
      <BrowserRouter>
        <Routes>
          <Route path="/" element={<LoginPage />} />
          <Route path="/register" element={<RegisterPage />} />
          <Route
            path="/manager"
            element={
              <ProtectedRoute>
                <ManagerDashboard />
              </ProtectedRoute>
            }
          />
          <Route
            path="/player"
            element={
              <ProtectedRoute>
                <PlayerDashboard />
              </ProtectedRoute>
            }
          />
          {/* Lobby de Mesas */}
          <Route
            path="/table/lobby"
            element={
              <ProtectedRoute>
                <TableLobbyPage />
              </ProtectedRoute>
            }
          />
          {/* Tela de Transição / Escolha de Assento */}
          <Route
            path="/table/select-seat"
            element={
              <ProtectedRoute>
                <SeatSelectionPage />
              </ProtectedRoute>
            }
          />
          {/* Mesa ao Vivo */}
          <Route
            path="/table/live"
            element={
              <ProtectedRoute>
                <PokerTablePage />
              </ProtectedRoute>
            }
          />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </BrowserRouter>
    </AuthProvider>
  );
}

export default App;

-- Schema inicial do Poker dos Amigos (PostgreSQL)

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Tabela de Usuários / Membros
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    nome_completo VARCHAR(255) NOT NULL,
    telefone VARCHAR(50) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    data_nascimento VARCHAR(20) NOT NULL,
    cidade_estado VARCHAR(100) NOT NULL,
    aceitou_termos BOOLEAN DEFAULT FALSE,
    role VARCHAR(50) DEFAULT 'jogador', -- 'admin_gerente', 'gerente', 'jogador'
    status VARCHAR(50) DEFAULT 'ativo',
    saldo_fichas BIGINT DEFAULT 1000,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Tabela de Mesas
CREATE TABLE IF NOT EXISTS poker_tables (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    nome VARCHAR(100) NOT NULL,
    tipo VARCHAR(50) NOT NULL, -- 'cash_game', 'torneio'
    small_blind BIGINT NOT NULL,
    big_blind BIGINT NOT NULL,
    buy_in_min BIGINT NOT NULL,
    buy_in_max BIGINT NOT NULL,
    max_seats INT DEFAULT 9,
    status VARCHAR(50) DEFAULT 'aguardando',
    current_deck JSONB, -- Memória física contínua do baralho serializada
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Tabela de Torneios
CREATE TABLE IF NOT EXISTS tournaments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    nome VARCHAR(150) NOT NULL,
    buy_in BIGINT NOT NULL,
    garantido BIGINT DEFAULT 0,
    max_inscritos INT DEFAULT 100,
    data_inicio TIMESTAMP WITH TIME ZONE NOT NULL,
    status VARCHAR(50) DEFAULT 'aberto',
    blind_interval_min INT DEFAULT 15,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Tabela de Histórico de Mãos
CREATE TABLE IF NOT EXISTS hands_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    table_id UUID REFERENCES poker_tables(id) ON DELETE CASCADE,
    hand_number BIGINT NOT NULL,
    community_cards JSONB,
    pot_total BIGINT NOT NULL,
    winners JSONB,
    deck_initial_snapshot JSONB,
    deck_final_snapshot JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Tabela Financeira
CREATE TABLE IF NOT EXISTS financial_transactions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    tipo VARCHAR(50) NOT NULL, -- 'deposito', 'saque', 'buyin', 'premio', 'rake'
    valor BIGINT NOT NULL,
    descricao TEXT,
    status VARCHAR(50) DEFAULT 'concluido',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Tabela de Comunicados
CREATE TABLE IF NOT EXISTS announcements (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    titulo VARCHAR(255) NOT NULL,
    conteudo TEXT NOT NULL,
    autor VARCHAR(100) NOT NULL,
    publicado BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

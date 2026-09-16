package models

import (
	"time"

	"github.com/google/uuid"
)

type UserRole string

const (
	RoleAdminGerente UserRole = "admin_gerente" // Jonatas e Felipe (direitos iguais)
	RoleGerente      UserRole = "gerente"
	RoleJogador      UserRole = "jogador"
)

type UserStatus string

const (
	StatusAtivo    UserStatus = "ativo"
	StatusPendente UserStatus = "pendente"
	StatusBloqueado UserStatus = "bloqueado"
)

type User struct {
	ID           uuid.UUID  `json:"id"`
	NomeCompleto string     `json:"nome_completo"`
	Telefone     string     `json:"telefone"`
	Email        string     `json:"email"`
	PasswordHash string     `json:"-"`
	DataNasc     string     `json:"data_nascimento"`
	CidadeEstado string     `json:"cidade_estado"`
	AceitouTermo bool       `json:"aceitou_termos"`
	Role         UserRole   `json:"role"`
	Status       UserStatus `json:"status"`
	SaldoFichas  int64      `json:"saldo_fichas"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type LoginRequest struct {
	Email      string `json:"email"`
	Senha      string `json:"senha"`
	LembrarMe  bool   `json:"lembrar_me"`
}

type RegisterRequest struct {
	NomeCompleto string `json:"nome_completo"`
	Telefone     string `json:"telefone"`
	Email        string `json:"email"`
	Senha        string `json:"senha"`
	DataNasc     string `json:"data_nascimento"`
	CidadeEstado string `json:"cidade_estado"`
	AceitouTermo bool   `json:"aceitou_termos"`
}

type AuthResponse struct {
	Token     string `json:"token"`
	ExpiresAt int64  `json:"expires_at"`
	User      User   `json:"user"`
}

package handlers

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/walissonpaulo/poker-dos-amigos-backend/internal/auth"
	"github.com/walissonpaulo/poker-dos-amigos-backend/internal/models"
	"github.com/walissonpaulo/poker-dos-amigos-backend/pkg/response"
)

type AuthHandler struct {
	tokenManager *auth.TokenManager
	users        map[string]*models.User // Em memória com fallback / demo
	mu           sync.RWMutex
}

func NewAuthHandler(tm *auth.TokenManager) *AuthHandler {
	h := &AuthHandler{
		tokenManager: tm,
		users:        make(map[string]*models.User),
	}

	// Criação dos sócios fundadores (Jonatas e Felipe) com direitos iguais de admin/gerente
	jonatasHash, _ := auth.HashPassword("poker123")
	felipeHash, _ := auth.HashPassword("poker123")

	h.users["jonatas@pokerdosamigos.com"] = &models.User{
		ID:           uuid.New(),
		NomeCompleto: "Jonatas",
		Telefone:     "(11) 99999-0001",
		Email:        "jonatas@pokerdosamigos.com",
		PasswordHash: jonatasHash,
		DataNasc:     "01/01/1990",
		CidadeEstado: "São Paulo/SP",
		AceitouTermo: true,
		Role:         models.RoleAdminGerente,
		Status:       models.StatusAtivo,
		SaldoFichas:  50000,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	h.users["felipe@pokerdosamigos.com"] = &models.User{
		ID:           uuid.New(),
		NomeCompleto: "Felipe",
		Telefone:     "(11) 99999-0002",
		Email:        "felipe@pokerdosamigos.com",
		PasswordHash: felipeHash,
		DataNasc:     "01/01/1990",
		CidadeEstado: "São Paulo/SP",
		AceitouTermo: true,
		Role:         models.RoleAdminGerente,
		Status:       models.StatusAtivo,
		SaldoFichas:  50000,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Usuário de demonstração Jogador
	jogadorHash, _ := auth.HashPassword("jogador123")
	h.users["jogador@pokerdosamigos.com"] = &models.User{
		ID:           uuid.New(),
		NomeCompleto: "Jogador VIP",
		Telefone:     "(11) 98888-7777",
		Email:        "jogador@pokerdosamigos.com",
		PasswordHash: jogadorHash,
		DataNasc:     "15/05/1995",
		CidadeEstado: "Curitiba/PR",
		AceitouTermo: true,
		Role:         models.RoleJogador,
		Status:       models.StatusAtivo,
		SaldoFichas:  10000,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	return h
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Corpo da requisição inválido")
		return
	}

	if req.Email == "" || req.Senha == "" {
		response.Error(w, http.StatusBadRequest, "E-mail e senha são obrigatórios")
		return
	}

	h.mu.RLock()
	user, exists := h.users[req.Email]
	h.mu.RUnlock()

	if !exists || !auth.CheckPasswordHash(req.Senha, user.PasswordHash) {
		response.Error(w, http.StatusUnauthorized, "Credenciais inválidas. Verifique seu e-mail e senha.")
		return
	}

	duration := 24 * time.Hour
	if req.LembrarMe {
		duration = 30 * 24 * time.Hour
	}

	token, exp, err := h.tokenManager.GenerateToken(user, duration)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Erro ao gerar token de autenticação")
		return
	}

	response.JSON(w, http.StatusOK, models.AuthResponse{
		Token:     token,
		ExpiresAt: exp,
		User:      *user,
	})
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Dados de formulário inválidos")
		return
	}

	if req.NomeCompleto == "" || req.Email == "" || req.Telefone == "" {
		response.Error(w, http.StatusBadRequest, "Todos os campos obrigatórios devem ser preenchidos")
		return
	}

	if !req.AceitouTermo {
		response.Error(w, http.StatusBadRequest, "É obrigatório aceitar o regulamento do clube e declarar ser maior de 18 anos")
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	if _, exists := h.users[req.Email]; exists {
		response.Error(w, http.StatusConflict, "Já existe uma conta cadastrada com este e-mail")
		return
	}

	passwordToHash := req.Senha
	if passwordToHash == "" {
		passwordToHash = "poker@123"
	}
	hash, err := auth.HashPassword(passwordToHash)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Erro ao processar senha")
		return
	}

	newUser := &models.User{
		ID:           uuid.New(),
		NomeCompleto: req.NomeCompleto,
		Telefone:     req.Telefone,
		Email:        req.Email,
		PasswordHash: hash,
		DataNasc:     req.DataNasc,
		CidadeEstado: req.CidadeEstado,
		AceitouTermo: req.AceitouTermo,
		Role:         models.RoleJogador,
		Status:       models.StatusAtivo,
		SaldoFichas:  1000, // Bônus inicial de boas-vindas
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	h.users[req.Email] = newUser

	token, exp, _ := h.tokenManager.GenerateToken(newUser, 24*time.Hour)

	response.JSON(w, http.StatusCreated, models.AuthResponse{
		Token:     token,
		ExpiresAt: exp,
		User:      *newUser,
	})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(auth.UserContextKey).(*auth.Claims)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "Usuário não autenticado")
		return
	}

	h.mu.RLock()
	user, exists := h.users[claims.Email]
	h.mu.RUnlock()

	if !exists {
		response.Error(w, http.StatusNotFound, "Usuário não encontrado")
		return
	}

	response.JSON(w, http.StatusOK, user)
}

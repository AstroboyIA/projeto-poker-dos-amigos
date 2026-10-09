package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/walissonpaulo/poker-dos-amigos-backend/internal/auth"
	"github.com/walissonpaulo/poker-dos-amigos-backend/internal/models"
	"github.com/walissonpaulo/poker-dos-amigos-backend/pkg/response"
)

const initialPlayerChips int64 = 1_000_000

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
		Username:     "jonatas",
		NomeCompleto: "Jonatas",
		Telefone:     "(11) 99999-0001",
		Email:        "jonatas@pokerdosamigos.com",
		PasswordHash: jonatasHash,
		DataNasc:     "01/01/1990",
		CidadeEstado: "São Paulo/SP",
		AceitouTermo: true,
		Role:         models.RoleAdminGerente,
		Status:       models.StatusAtivo,
		SaldoFichas:  initialPlayerChips,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	h.users["jonatas"] = h.users["jonatas@pokerdosamigos.com"]

	h.users["felipe@pokerdosamigos.com"] = &models.User{
		ID:           uuid.New(),
		Username:     "felipe",
		NomeCompleto: "Felipe",
		Telefone:     "(11) 99999-0002",
		Email:        "felipe@pokerdosamigos.com",
		PasswordHash: felipeHash,
		DataNasc:     "01/01/1990",
		CidadeEstado: "São Paulo/SP",
		AceitouTermo: true,
		Role:         models.RoleAdminGerente,
		Status:       models.StatusAtivo,
		SaldoFichas:  initialPlayerChips,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	h.users["felipe"] = h.users["felipe@pokerdosamigos.com"]

	// Usuário de demonstração Jogador
	jogadorHash, _ := auth.HashPassword("jogador123")
	h.users["jogador@pokerdosamigos.com"] = &models.User{
		ID:           uuid.New(),
		Username:     "jogador",
		NomeCompleto: "Jogador VIP",
		Telefone:     "(11) 98888-7777",
		Email:        "jogador@pokerdosamigos.com",
		PasswordHash: jogadorHash,
		DataNasc:     "15/05/1995",
		CidadeEstado: "Curitiba/PR",
		AceitouTermo: true,
		Role:         models.RoleJogador,
		Status:       models.StatusAtivo,
		SaldoFichas:  initialPlayerChips,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	h.users["jogador"] = h.users["jogador@pokerdosamigos.com"]

	return h
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Corpo da requisição inválido")
		return
	}

	identifier := strings.TrimSpace(req.Username)
	if identifier == "" {
		identifier = strings.TrimSpace(req.Email)
	}
	if identifier == "" || req.Senha == "" {
		response.Error(w, http.StatusBadRequest, "Username/e-mail e senha são obrigatórios")
		return
	}

	h.mu.RLock()
	user, exists := h.users[strings.ToLower(identifier)]
	h.mu.RUnlock()

	if !exists || !auth.CheckPasswordHash(req.Senha, user.PasswordHash) {
		response.Error(w, http.StatusUnauthorized, "Credenciais inválidas. Verifique seu username/e-mail e senha.")
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

	req.Username = strings.TrimSpace(req.Username)
	req.Email = strings.TrimSpace(req.Email)
	req.NomeCompleto = strings.TrimSpace(req.NomeCompleto)
	req.Telefone = strings.TrimSpace(req.Telefone)
	usernameOnly := req.Username != "" && req.Email == "" && req.NomeCompleto == "" && req.Telefone == "" && req.DataNasc == "" && req.CidadeEstado == ""
	if req.Senha == "" {
		response.Error(w, http.StatusBadRequest, "A senha é obrigatória")
		return
	}
	if req.Username == "" && req.Email == "" {
		response.Error(w, http.StatusBadRequest, "Username ou e-mail é obrigatório")
		return
	}
	if !usernameOnly && (req.NomeCompleto == "" || req.Email == "" || req.Telefone == "") {
		response.Error(w, http.StatusBadRequest, "Nome, telefone e e-mail são obrigatórios para o cadastro completo")
		return
	}

	if !req.AceitouTermo {
		response.Error(w, http.StatusBadRequest, "É obrigatório aceitar o regulamento do clube e declarar ser maior de 18 anos")
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	usernameKey := strings.ToLower(req.Username)
	emailKey := strings.ToLower(req.Email)
	if req.Username != "" {
		if _, exists := h.users[usernameKey]; exists {
			response.Error(w, http.StatusConflict, "Já existe uma conta cadastrada com este username")
			return
		}
	}
	if emailKey != "" {
		if _, exists := h.users[emailKey]; exists {
			response.Error(w, http.StatusConflict, "Já existe uma conta cadastrada com este e-mail")
			return
		}
	}
	if usernameKey != "" && usernameKey == emailKey {
		response.Error(w, http.StatusConflict, "Username e e-mail já estão em uso")
		return
	}

	hash, err := auth.HashPassword(req.Senha)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Erro ao processar senha")
		return
	}

	newUser := &models.User{
		ID:           uuid.New(),
		Username:     req.Username,
		NomeCompleto: req.NomeCompleto,
		Telefone:     req.Telefone,
		Email:        req.Email,
		PasswordHash: hash,
		DataNasc:     req.DataNasc,
		CidadeEstado: req.CidadeEstado,
		AceitouTermo: req.AceitouTermo,
		Role:         models.RoleJogador,
		Status:       models.StatusAtivo,
		SaldoFichas:  initialPlayerChips,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	token, exp, err := h.tokenManager.GenerateToken(newUser, 24*time.Hour)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Erro ao gerar token de autenticação")
		return
	}

	if usernameKey != "" {
		h.users[usernameKey] = newUser
	}
	if emailKey != "" {
		h.users[emailKey] = newUser
	}

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
	user, exists := h.getUserByIdentityLocked(claims.Username, claims.Email)
	h.mu.RUnlock()

	if !exists {
		response.Error(w, http.StatusNotFound, "Usuário não encontrado")
		return
	}

	response.JSON(w, http.StatusOK, user)
}

func (h *AuthHandler) GetUserByEmail(email string) (*models.User, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	u, exists := h.users[strings.ToLower(strings.TrimSpace(email))]
	return u, exists
}

func (h *AuthHandler) GetUserByIdentity(username, email string) (*models.User, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.getUserByIdentityLocked(username, email)
}

func (h *AuthHandler) getUserByIdentityLocked(username, email string) (*models.User, bool) {
	identity := strings.TrimSpace(username)
	if identity == "" {
		identity = strings.TrimSpace(email)
	}
	u, exists := h.users[strings.ToLower(identity)]
	return u, exists
}

func (h *AuthHandler) UpdateChips(email string, delta int64) (*models.User, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	u, exists := h.users[strings.ToLower(strings.TrimSpace(email))]
	if !exists {
		return nil, false
	}
	u.SaldoFichas += delta
	if u.SaldoFichas < 0 {
		u.SaldoFichas = 0
	}
	u.UpdatedAt = time.Now()
	return u, true
}

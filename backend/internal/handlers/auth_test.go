package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/walissonpaulo/poker-dos-amigos-backend/internal/auth"
	"github.com/walissonpaulo/poker-dos-amigos-backend/internal/models"
)

func TestUpdateProfileChangesOnlyAuthenticatedUserProfile(t *testing.T) {
	tokenManager := auth.NewTokenManager("test-secret")
	handler := NewAuthHandler(tokenManager)
	user := handler.users["jogador"]
	token, _, err := tokenManager.GenerateToken(user, 24*time.Hour)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	request := httptest.NewRequest(http.MethodPut, "/api/auth/profile", strings.NewReader(
		`{"nome_completo":"Jogador Atualizado","telefone":"(41) 99999-1111","data_nascimento":"20/06/1996","cidade_estado":"Curitiba/PR","role":"admin_gerente","saldo_fichas":0}`,
	))
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	tokenManager.AuthMiddleware(http.HandlerFunc(handler.UpdateProfile)).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("update profile status = %d, want %d: %s", response.Code, http.StatusOK, response.Body.String())
	}
	if user.NomeCompleto != "Jogador Atualizado" || user.Telefone != "(41) 99999-1111" ||
		user.DataNasc != "20/06/1996" || user.CidadeEstado != "Curitiba/PR" {
		t.Fatalf("profile fields were not updated: %+v", user)
	}
	if user.Role != models.RoleJogador || user.SaldoFichas != initialPlayerChips {
		t.Fatalf("protected fields changed: role=%s balance=%d", user.Role, user.SaldoFichas)
	}

	var result struct {
		Data models.AuthResponse `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatalf("decode update response: %v", err)
	}
	if result.Data.Token == "" || result.Data.User.NomeCompleto != "Jogador Atualizado" || result.Data.User.Username != user.Username {
		t.Fatalf("unexpected updated user response: %+v", result.Data)
	}
}

func TestUpdateProfileRejectsOversizedFields(t *testing.T) {
	tokenManager := auth.NewTokenManager("test-secret")
	handler := NewAuthHandler(tokenManager)
	user := handler.users["jogador"]
	token, _, err := tokenManager.GenerateToken(user, 24*time.Hour)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	request := httptest.NewRequest(http.MethodPut, "/api/auth/profile", strings.NewReader(
		`{"nome_completo":"`+strings.Repeat("a", 256)+`"}`,
	))
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	tokenManager.AuthMiddleware(http.HandlerFunc(handler.UpdateProfile)).ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("oversized profile status = %d, want %d: %s", response.Code, http.StatusBadRequest, response.Body.String())
	}
	if user.NomeCompleto != "Jogador VIP" {
		t.Fatalf("profile changed after invalid request: %q", user.NomeCompleto)
	}
}

func TestUpdateProfileValidatesPhoneAndBirthDate(t *testing.T) {
	tests := []struct {
		name      string
		phone     string
		birthDate string
	}{
		{name: "phone without formatting", phone: "81981818181", birthDate: "15/05/1995"},
		{name: "date without formatting", phone: "(41) 99999-1111", birthDate: "01012001"},
		{name: "impossible date", phone: "(41) 99999-1111", birthDate: "31/02/2001"},
		{name: "future date", phone: "(41) 99999-1111", birthDate: time.Now().AddDate(1, 0, 0).Format("02/01/2006")},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tokenManager := auth.NewTokenManager("test-secret")
			handler := NewAuthHandler(tokenManager)
			user := handler.users["jogador"]
			token, _, err := tokenManager.GenerateToken(user, 24*time.Hour)
			if err != nil {
				t.Fatalf("generate token: %v", err)
			}
			body, err := json.Marshal(models.UserProfileUpdate{
				NomeCompleto: "Nome não deve ser salvo",
				Telefone:     test.phone,
				DataNasc:     test.birthDate,
				CidadeEstado: "São Paulo/SP",
			})
			if err != nil {
				t.Fatalf("marshal request: %v", err)
			}
			request := httptest.NewRequest(http.MethodPut, "/api/auth/profile", strings.NewReader(string(body)))
			request.Header.Set("Authorization", "Bearer "+token)
			response := httptest.NewRecorder()
			tokenManager.AuthMiddleware(http.HandlerFunc(handler.UpdateProfile)).ServeHTTP(response, request)

			if response.Code != http.StatusBadRequest {
				t.Fatalf("invalid profile status = %d, want %d: %s", response.Code, http.StatusBadRequest, response.Body.String())
			}
			if user.NomeCompleto != "Jogador VIP" {
				t.Fatalf("profile changed despite invalid phone/date: %q", user.NomeCompleto)
			}
		})
	}
}

func TestChangePasswordRequiresCurrentPasswordAndUpdatesCredentials(t *testing.T) {
	tokenManager := auth.NewTokenManager("test-secret")
	handler := NewAuthHandler(tokenManager)
	user := handler.users["jogador"]
	token, _, err := tokenManager.GenerateToken(user, 24*time.Hour)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	change := func(current, next string) *httptest.ResponseRecorder {
		body, err := json.Marshal(models.ChangePasswordRequest{SenhaAtual: current, NovaSenha: next})
		if err != nil {
			t.Fatalf("marshal request: %v", err)
		}
		request := httptest.NewRequest(http.MethodPut, "/api/auth/password", strings.NewReader(string(body)))
		request.Header.Set("Authorization", "Bearer "+token)
		response := httptest.NewRecorder()
		tokenManager.AuthMiddleware(http.HandlerFunc(handler.ChangePassword)).ServeHTTP(response, request)
		return response
	}

	if response := change("wrong-password", "new-secure-pass"); response.Code != http.StatusUnauthorized {
		t.Fatalf("incorrect current password status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
	if response := change("jogador123", "short"); response.Code != http.StatusBadRequest {
		t.Fatalf("short new password status = %d, want %d", response.Code, http.StatusBadRequest)
	}
	if response := change("jogador123", "new-secure-pass"); response.Code != http.StatusOK {
		t.Fatalf("change password status = %d, want %d: %s", response.Code, http.StatusOK, response.Body.String())
	}
	if auth.CheckPasswordHash("jogador123", user.PasswordHash) || !auth.CheckPasswordHash("new-secure-pass", user.PasswordHash) {
		t.Fatal("password hash was not updated correctly")
	}
}

func TestUsernameOnlyRegistrationAndLogin(t *testing.T) {
	tokenManager := auth.NewTokenManager("test-secret")
	handler := NewAuthHandler(tokenManager)
	for _, username := range []string{"jonatas", "felipe", "jogador"} {
		if user := handler.users[username]; user.SaldoFichas != initialPlayerChips {
			t.Errorf("saldo inicial de %s = %d, esperado %d", username, user.SaldoFichas, initialPlayerChips)
		}
	}

	missingTermsRequest := httptest.NewRequest(http.MethodPost, "/api/auth/register", strings.NewReader(`{"username":"newplayer","senha":"secret123"}`))
	missingTermsResponse := httptest.NewRecorder()
	handler.Register(missingTermsResponse, missingTermsRequest)
	if missingTermsResponse.Code != http.StatusBadRequest {
		t.Fatalf("registration without terms status = %d, want %d", missingTermsResponse.Code, http.StatusBadRequest)
	}

	registerRequest := httptest.NewRequest(http.MethodPost, "/api/auth/register", strings.NewReader(`{"username":"newplayer","senha":"secret123","aceitou_termos":true}`))
	registerResponse := httptest.NewRecorder()
	handler.Register(registerResponse, registerRequest)
	if registerResponse.Code != http.StatusCreated {
		t.Fatalf("registration status = %d, want %d: %s", registerResponse.Code, http.StatusCreated, registerResponse.Body.String())
	}

	var registerResult struct {
		Data models.AuthResponse `json:"data"`
	}
	if err := json.NewDecoder(registerResponse.Body).Decode(&registerResult); err != nil {
		t.Fatalf("decode registration response: %v", err)
	}
	if registerResult.Data.User.Username != "newplayer" || registerResult.Data.User.Email != "" {
		t.Fatalf("registered user = %+v, expected username only", registerResult.Data.User)
	}
	if registerResult.Data.User.SaldoFichas != initialPlayerChips {
		t.Errorf("saldo do novo usuário = %d, esperado %d", registerResult.Data.User.SaldoFichas, initialPlayerChips)
	}

	meRequest := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	meRequest.Header.Set("Authorization", "Bearer "+registerResult.Data.Token)
	meResponse := httptest.NewRecorder()
	tokenManager.AuthMiddleware(http.HandlerFunc(handler.Me)).ServeHTTP(meResponse, meRequest)
	if meResponse.Code != http.StatusOK {
		t.Fatalf("me status = %d, want %d: %s", meResponse.Code, http.StatusOK, meResponse.Body.String())
	}

	loginRequest := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"username":"NEWPLAYER","senha":"secret123"}`))
	loginResponse := httptest.NewRecorder()
	handler.Login(loginResponse, loginRequest)
	if loginResponse.Code != http.StatusOK {
		t.Fatalf("login status = %d, want %d: %s", loginResponse.Code, http.StatusOK, loginResponse.Body.String())
	}

	emailLoginRequest := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"email":"jogador@pokerdosamigos.com","senha":"jogador123"}`))
	emailLoginResponse := httptest.NewRecorder()
	handler.Login(emailLoginResponse, emailLoginRequest)
	if emailLoginResponse.Code != http.StatusOK {
		t.Fatalf("legacy email login status = %d, want %d", emailLoginResponse.Code, http.StatusOK)
	}

	duplicateRequest := httptest.NewRequest(http.MethodPost, "/api/auth/register", strings.NewReader(`{"username":"NEWPLAYER","senha":"different123","aceitou_termos":true}`))
	duplicateResponse := httptest.NewRecorder()
	handler.Register(duplicateResponse, duplicateRequest)
	if duplicateResponse.Code != http.StatusConflict {
		t.Fatalf("duplicate registration status = %d, want %d", duplicateResponse.Code, http.StatusConflict)
	}
}

func TestRegistrationRequiresUsernameWithoutSpaces(t *testing.T) {
	tests := []struct {
		name     string
		username string
		wantCode int
	}{
		{name: "missing", username: "", wantCode: http.StatusBadRequest},
		{name: "contains space", username: "nome sobrenome", wantCode: http.StatusBadRequest},
		{name: "contains unsupported punctuation", username: "nome.sobrenome", wantCode: http.StatusBadRequest},
		{name: "hyphen separator", username: "nome-sobrenome", wantCode: http.StatusCreated},
		{name: "underscore separator", username: "nome_sobrenome", wantCode: http.StatusCreated},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler := NewAuthHandler(auth.NewTokenManager("test-secret"))
			body, err := json.Marshal(map[string]interface{}{
				"username":       test.username,
				"senha":          "secret123",
				"aceitou_termos": true,
			})
			if err != nil {
				t.Fatal(err)
			}
			request := httptest.NewRequest(http.MethodPost, "/api/auth/register", strings.NewReader(string(body)))
			response := httptest.NewRecorder()
			handler.Register(response, request)
			if response.Code != test.wantCode {
				t.Fatalf("registration status = %d, want %d: %s", response.Code, test.wantCode, response.Body.String())
			}
		})
	}
}

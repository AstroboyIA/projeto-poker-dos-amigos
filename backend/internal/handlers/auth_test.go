package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/walissonpaulo/poker-dos-amigos-backend/internal/auth"
	"github.com/walissonpaulo/poker-dos-amigos-backend/internal/models"
)

func TestUsernameOnlyRegistrationAndLogin(t *testing.T) {
	tokenManager := auth.NewTokenManager("test-secret")
	handler := NewAuthHandler(tokenManager)

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

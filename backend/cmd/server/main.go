package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/google/uuid"
	"github.com/walissonpaulo/poker-dos-amigos-backend/internal/auth"
	"github.com/walissonpaulo/poker-dos-amigos-backend/internal/config"
	"github.com/walissonpaulo/poker-dos-amigos-backend/internal/engine"
	"github.com/walissonpaulo/poker-dos-amigos-backend/internal/handlers"
	"github.com/walissonpaulo/poker-dos-amigos-backend/internal/ws"
)

func main() {
	cfg := config.LoadConfig()

	tokenManager := auth.NewTokenManager(cfg.JWTSecret)
	gameService := engine.NewGameService()
	hub := ws.NewHub(gameService)
	go hub.Run()

	authHandler := handlers.NewAuthHandler(tokenManager)
	modulesHandler := handlers.NewModulesHandler(hub, authHandler)
	hub.SetReleaseSeatHandler(modulesHandler.ReleaseSeat)

	r := chi.NewRouter()

	// Middlewares
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// CORS Setup
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	// Rotas Públicas
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"online","service":"Poker dos Amigos API"}`))
	})

	r.Route("/api/auth", func(r chi.Router) {
		r.Post("/login", authHandler.Login)
		r.Post("/register", authHandler.Register)
	})

	// Rotas Protegidas (JWT)
	r.Group(func(r chi.Router) {
		r.Use(tokenManager.AuthMiddleware)

		r.Get("/api/auth/me", authHandler.Me)

		// Módulos
		r.Get("/api/tournaments", modulesHandler.GetTournaments)
		r.Get("/api/rankings", modulesHandler.GetRankings)
		r.Get("/api/tables", modulesHandler.GetTables)
		r.Post("/api/tables", modulesHandler.CreateTable)
		r.Post("/api/tables/occupy", modulesHandler.OccupySeat)
		r.Post("/api/tables/leave", modulesHandler.LeaveSeat)
		r.Post("/api/chips/buy-in", modulesHandler.BuyIn)
		r.Post("/api/chips/cash-out", modulesHandler.CashOut)
		r.Get("/api/announcements", modulesHandler.GetAnnouncements)

		// Módulos Gerenciais (Admin / Gerente)
		r.Group(func(r chi.Router) {
			r.Use(auth.RequireManager)
			r.Get("/api/financial/summary", modulesHandler.GetFinancialSummary)
		})
	})

	// WebSocket Endpoint
	r.Get("/ws", func(w http.ResponseWriter, r *http.Request) {
		tokenStr := r.URL.Query().Get("token")
		var userID uuid.UUID
		userName := "Visitante"

		if tokenStr != "" {
			if claims, err := tokenManager.ValidateToken(tokenStr); err == nil {
				userID = claims.UserID
				userName = claims.Nome
			}
		}

		if userID == uuid.Nil {
			userID = uuid.New()
		}

		hub.ServeWS(w, r, userID, userName)
	})

	serverAddr := fmt.Sprintf(":%s", cfg.Port)
	log.Printf("♠️ ♥️ ♦️ ♣️ Servidor Poker dos Amigos rodando em http://localhost%s", serverAddr)
	if err := http.ListenAndServe(serverAddr, r); err != nil {
		log.Fatalf("Erro ao iniciar servidor: %v", err)
	}
}

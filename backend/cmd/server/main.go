package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/google/uuid"
	"github.com/walissonpaulo/poker-dos-amigos-backend/internal/auth"
	"github.com/walissonpaulo/poker-dos-amigos-backend/internal/config"
	"github.com/walissonpaulo/poker-dos-amigos-backend/internal/database"
	"github.com/walissonpaulo/poker-dos-amigos-backend/internal/engine"
	"github.com/walissonpaulo/poker-dos-amigos-backend/internal/handlers"
	"github.com/walissonpaulo/poker-dos-amigos-backend/internal/ws"
)

func main() {
	cfg := config.LoadConfig()
	if cfg.DatabaseURL == "" {
		log.Fatal("DATABASE_URL é obrigatória para compartilhar mesas entre instâncias")
	}
	dbCtx, cancelDB := context.WithTimeout(context.Background(), 15*time.Second)
	store, err := database.Open(dbCtx, cfg.DatabaseURL)
	cancelDB()
	if err != nil {
		log.Fatalf("Falha ao conectar ao PostgreSQL compartilhado: %v", err)
	}
	defer store.Close()
	migrationCtx, cancelMigration := context.WithTimeout(context.Background(), 15*time.Second)
	if err := store.ApplySharedTableMigration(migrationCtx); err != nil {
		cancelMigration()
		log.Fatalf("Falha ao aplicar migração do estado compartilhado: %v", err)
	}
	cancelMigration()

	tokenManager := auth.NewTokenManager(cfg.JWTSecret)
	gameService := engine.NewGameService()
	hub := ws.NewHub(gameService)
	hub.SetStore(store)
	go hub.Run()
	go hub.ListenForSharedUpdates(context.Background())

	authHandler := handlers.NewAuthHandlerWithStore(tokenManager, store)
	modulesHandler := handlers.NewModulesHandlerWithStore(hub, authHandler, store)
	tournamentHandler := handlers.NewTournamentHandler(store, hub, authHandler)
	hub.SetReleaseSeatHandler(modulesHandler.ReleaseSeat)
	go tournamentHandler.Run(context.Background())

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
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "Idempotency-Key"},
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
		r.Get("/api/tournaments", tournamentHandler.GetTournaments)
		r.Post("/api/tournaments", tournamentHandler.CreateTournament)
		r.Post("/api/tournaments/{tournamentID}/register", tournamentHandler.Register)
		r.Post("/api/tournaments/{tournamentID}/leave", tournamentHandler.Leave)
		r.Post("/api/tournaments/{tournamentID}/start", tournamentHandler.Start)
		r.Get("/api/rankings", modulesHandler.GetRankings)
		r.Get("/api/tables", modulesHandler.GetTables)
		r.Post("/api/tables", modulesHandler.CreateTable)
		r.Post("/api/tables/occupy", modulesHandler.OccupySeat)
		r.Post("/api/tables/leave", modulesHandler.LeaveSeat)
		r.Post("/api/chips/buy-in", modulesHandler.BuyIn)
		r.Post("/api/chips/rebuy", modulesHandler.Rebuy)
		r.Post("/api/chips/cash-out", modulesHandler.CashOut)
		r.Get("/api/wallet", modulesHandler.GetWallet)
		r.Post("/api/dev/wallet/deposit", modulesHandler.DevDeposit)
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
		if tokenStr == "" {
			http.Error(w, "autenticação obrigatória", http.StatusUnauthorized)
			return
		}
		claims, err := tokenManager.ValidateToken(tokenStr)
		if err != nil || claims.UserID == uuid.Nil {
			http.Error(w, "token inválido", http.StatusUnauthorized)
			return
		}
		hub.ServeWS(w, r, claims.UserID, claims.Nome)
	})

	serverAddr := fmt.Sprintf(":%s", cfg.Port)
	log.Printf("♠️ ♥️ ♦️ ♣️ Servidor Poker dos Amigos rodando em http://localhost%s", serverAddr)
	if err := http.ListenAndServe(serverAddr, r); err != nil {
		log.Fatalf("Erro ao iniciar servidor: %v", err)
	}
}

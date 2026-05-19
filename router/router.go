package router

import (
	"net/http"

	"github.com/ecaliskaner/Insider_One_Backend_Project/database"
	"github.com/ecaliskaner/Insider_One_Backend_Project/handlers"
	"github.com/ecaliskaner/Insider_One_Backend_Project/middleware"
	"github.com/gorilla/mux"
	httpSwagger "github.com/swaggo/http-swagger"
)

// NewRouter creates the HTTP router with all /api/v1 endpoints
func NewRouter(handler *handlers.LeagueHandler, db *database.DB) *mux.Router {
	r := mux.NewRouter()
	healthHandler := handlers.NewHealthHandler(db)

	r.HandleFunc("/healthz", healthHandler.Healthz).Methods(http.MethodGet)
	r.HandleFunc("/readyz", healthHandler.Readyz).Methods(http.MethodGet)

	// API v1 subrouter
	v1 := r.PathPrefix("/api/v1").Subrouter()

	// GET  /api/v1/health — API-scoped liveness probe
	v1.HandleFunc("/health", healthHandler.Healthz).Methods(http.MethodGet)

	// GET  /api/v1/league/table — Current standings
	v1.HandleFunc("/league/table", handler.GetTable).Methods(http.MethodGet)

	// GET  /api/v1/league/overview — Case-friendly league screen payload
	v1.HandleFunc("/league/overview", handler.GetOverview).Methods(http.MethodGet)

	// POST /api/v1/league/next-week — Simulate next week
	v1.HandleFunc("/league/next-week", handler.PlayNextWeek).Methods(http.MethodPost)

	// POST /api/v1/league/play-all — Play all remaining weeks
	v1.HandleFunc("/league/play-all", handler.PlayAll).Methods(http.MethodPost)

	// GET  /api/v1/matches/{id} — Get match
	v1.HandleFunc("/matches/{id}", handler.GetMatch).Methods(http.MethodGet)

	// PUT  /api/v1/matches/{id} — Edit match result
	v1.HandleFunc("/matches/{id}", handler.EditMatch).Methods(http.MethodPut)

	// GET  /api/v1/simulation/championship-probabilities — Monte Carlo predictions
	v1.HandleFunc("/simulation/championship-probabilities", handler.GetChampionshipProbabilities).Methods(http.MethodGet)

	// POST /api/v1/league/rollback/{week} — Rollback league state
	v1.HandleFunc("/league/rollback/{week}", handler.Rollback).Methods(http.MethodPost)

	// GET  /api/v1/teams/{id}/metrics — Team metrics
	v1.HandleFunc("/teams/{id}/metrics", handler.GetTeamMetrics).Methods(http.MethodGet)

	// POST /api/v1/league/reset — Reset league
	v1.HandleFunc("/league/reset", handler.Reset).Methods(http.MethodPost)

	// Enterprise Middlewares
	r.Use(middleware.PanicRecoveryMiddleware)
	r.Use(middleware.RequestIDMiddleware)
	r.Use(middleware.LoggingMiddleware)
	r.Use(middleware.RateLimiterMiddleware)
	r.Use(corsMiddleware)

	// Swagger documentation route
	r.PathPrefix("/swagger/").Handler(httpSwagger.WrapHandler)

	return r
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/insider/league-simulation/database"
	"github.com/insider/league-simulation/handlers"
	"github.com/insider/league-simulation/router"
	"github.com/insider/league-simulation/services"

	_ "github.com/insider/league-simulation/docs"
)

// @title           Football League Simulation API
// @version         1.0
// @description     This is a sophisticated league simulation API with Monte Carlo predictions and Time Machine rollback.
// @host            localhost:8080
// @BasePath        /api/v1

func main() {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./league.db"
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Initialize database
	db, err := database.NewDB(dbPath)
	if err != nil {
		log.Fatalf("❌ Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Seed data
	if err := database.SeedTeams(db); err != nil {
		log.Fatalf("❌ Failed to seed teams: %v", err)
	}
	if err := database.SeedPlayers(db); err != nil {
		log.Fatalf("❌ Failed to seed players: %v", err)
	}
	if err := database.SeedStandings(db); err != nil {
		log.Fatalf("❌ Failed to seed standings: %v", err)
	}
	if err := database.GenerateSchedule(db); err != nil {
		log.Fatalf("❌ Failed to generate schedule: %v", err)
	}

	// Initialize services (Adapter pattern — external APIs injected)
	matchEngine := services.NewMatchEngine()
	weatherAdapter := services.NewWeatherAdapter()
	leagueService := services.NewLeagueService(db, matchEngine, weatherAdapter)

	// Initialize handlers
	leagueHandler := handlers.NewLeagueHandler(leagueService)

	// Setup router
	r := router.NewRouter(leagueHandler)

	// Start server
	log.Println("╔══════════════════════════════════════════════╗")
	log.Println("║   ⚽ Football League Simulation API v1       ║")
	log.Printf("║   🌐 http://localhost:%s                    ║\n", port)
	log.Println("╚══════════════════════════════════════════════╝")
	log.Println("")
	log.Println("📡 Endpoints:")
	log.Println("  GET    /api/v1/league/table            — League standings")
	log.Println("  POST   /api/v1/league/next-week        — Simulate next week")
	log.Println("  POST   /api/v1/league/play-all         — Play all remaining")
	log.Println("  PUT    /api/v1/matches/{id}            — Edit match result")
	log.Println("  GET    /api/v1/simulation/oracle       — Monte Carlo predictions")
	log.Println("  POST   /api/v1/league/rollback/{week}  — Time Machine rollback")
	log.Println("  GET    /api/v1/teams/{id}/metrics      — Team metrics")
	log.Println("  POST   /api/v1/league/reset            — Reset league")
	log.Println("")

	// Configure HTTP Server
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	// Channel to listen for OS signals
	quit := make(chan os.Signal, 1)
	// Listen for SIGINT (Ctrl+C) and SIGTERM (Docker/Kubernetes termination)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	// Start server in a separate goroutine
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("❌ Server failed: %v", err)
		}
	}()

	// Block until a signal is received
	<-quit
	log.Println("\n⚠️ Shutting down server gracefully...")

	// Create a deadline to wait for active requests
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("❌ Server forced to shutdown: %v", err)
	}

	log.Println("✅ Server exited cleanly.")
}

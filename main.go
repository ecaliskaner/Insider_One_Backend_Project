package main

import (
	"log"
	"net/http"
	"os"

	"github.com/insider/league-simulation/database"
	"github.com/insider/league-simulation/handlers"
	"github.com/insider/league-simulation/router"
	"github.com/insider/league-simulation/services"
)

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

	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("❌ Server failed: %v", err)
	}
}

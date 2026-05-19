package cmd

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/ecaliskaner/Insider_One_Backend_Project/database"
	"github.com/ecaliskaner/Insider_One_Backend_Project/handlers"
	"github.com/ecaliskaner/Insider_One_Backend_Project/router"
	"github.com/ecaliskaner/Insider_One_Backend_Project/services"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Starts the HTTP server",
	Run: func(cmd *cobra.Command, args []string) {
		dbPath := viper.GetString("DB_PATH")
		port := viper.GetString("PORT")
		simSeed := viper.GetString("SIM_SEED")
		weatherProvider := viper.GetString("WEATHER_PROVIDER")
		strengthProviderName := viper.GetString("TEAM_STRENGTH_PROVIDER")
		transfermarktBaseURL := viper.GetString("TRANSFERMARKT_API_BASE_URL")

		// Initialize database (only connects and auto-migrates if enabled)
		db, err := database.NewDB(dbPath)
		if err != nil {
			log.Fatalf("❌ Failed to initialize database: %v", err)
		}
		defer func() {
			if err := db.Close(); err != nil {
				log.Printf("failed to close database: %v", err)
			}
		}()

		// Initialize services (Adapter pattern — external APIs injected)
		matchEngine := services.NewMatchEngine()
		weatherAdapter := services.NewWeatherAdapter()
		if simSeed != "" {
			seed, err := strconv.ParseInt(simSeed, 10, 64)
			if err != nil {
				log.Fatalf("invalid SIM_SEED %q: %v", simSeed, err)
			}
			matchEngine = services.NewMatchEngineWithSeed(seed)
			weatherAdapter = services.NewWeatherAdapterWithSeed(seed + 1)
			log.Printf("Using deterministic simulation seed: %d", seed)
		}
		weather := services.NewWeatherAdapterByProvider(weatherProvider, weatherAdapter)
		log.Printf("Using weather provider: %s", weatherProvider)
		strengthProvider := services.NewTeamStrengthProviderByProvider(strengthProviderName, transfermarktBaseURL)
		log.Printf("Using team strength provider: %s", strengthProviderName)
		leagueService := services.NewLeagueServiceWithStrengthProvider(db, matchEngine, weather, strengthProvider)

		// Initialize handlers
		leagueHandler := handlers.NewLeagueHandler(leagueService)

		// Setup router
		r := router.NewRouter(leagueHandler, db)

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
		log.Println("  GET    /api/v1/simulation/championship-probabilities — Championship probabilities")
		log.Println("  POST   /api/v1/league/rollback/{week}  — Rollback league state")
		log.Println("  GET    /api/v1/teams/{id}/metrics      — Team metrics")
		log.Println("  POST   /api/v1/league/reset            — Reset league")
		log.Println("")

		// Configure HTTP Server
		srv := &http.Server{
			Addr:              ":" + port,
			Handler:           r,
			ReadHeaderTimeout: 5 * time.Second,
			ReadTimeout:       10 * time.Second,
			WriteTimeout:      15 * time.Second,
			IdleTimeout:       60 * time.Second,
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
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}

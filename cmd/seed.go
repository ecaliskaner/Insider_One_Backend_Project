package cmd

import (
	"log"

	"github.com/ecaliskaner/Insider_One_Backend_Project/database"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var seedCmd = &cobra.Command{
	Use:   "seed",
	Short: "Seeds the database with initial teams, players, and standings",
	Run: func(cmd *cobra.Command, args []string) {
		dbPath := viper.GetString("DB_PATH")

		db, err := database.NewDB(dbPath)
		if err != nil {
			log.Fatalf("❌ Failed to initialize database: %v", err)
		}
		defer db.Close()

		log.Println("🌱 Seeding database...")

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

		log.Println("✅ Database seeded successfully!")
	},
}

func init() {
	rootCmd.AddCommand(seedCmd)
}

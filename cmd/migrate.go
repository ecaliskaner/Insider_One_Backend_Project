package cmd

import (
	"log"

	"github.com/ecaliskaner/Insider_One_Backend_Project/database"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Database migration commands",
}

var migrateUpCmd = &cobra.Command{
	Use:   "up",
	Short: "Run database migrations up",
	Run: func(cmd *cobra.Command, args []string) {
		dbPath := viper.GetString("DB_PATH")

		db, err := database.NewDB(dbPath)
		if err != nil {
			log.Fatalf("failed to connect to database: %v", err)
		}
		defer func() {
			if err := db.Close(); err != nil {
				log.Printf("failed to close database: %v", err)
			}
		}()

		log.Println("Running migrations...")

		if err := db.RunMigrations(); err != nil {
			log.Fatalf("migration failed: %v", err)
		}

		log.Println("Migrations completed successfully.")
	},
}

func init() {
	migrateCmd.AddCommand(migrateUpCmd)
	rootCmd.AddCommand(migrateCmd)
}

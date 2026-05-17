package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:   "insider",
	Short: "Insider Football League Simulation API",
	Long:  `A sophisticated REST API that simulates a football league using a Poisson-based match engine, Monte Carlo predictions, and a Time Machine rollback feature.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
}

func initConfig() {
	// Enable reading from environment variables
	viper.AutomaticEnv()

	// Default configurations
	viper.SetDefault("PORT", "8080")
	viper.SetDefault("DB_PATH", "./league.db")
	viper.SetDefault("SIM_SEED", "")
}

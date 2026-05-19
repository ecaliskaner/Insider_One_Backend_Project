package main

import (
	"github.com/ecaliskaner/Insider_One_Backend_Project/cmd"
	_ "github.com/ecaliskaner/Insider_One_Backend_Project/docs"
)

// @title           Football League Simulation API
// @version         1.0
// @description     REST API for simulating a four-team football league, editing match results, rolling back state, and calculating championship probabilities.
// @host            localhost:8080
// @BasePath        /api/v1

func main() {
	cmd.Execute()
}

package main

import (
	"github.com/insider/league-simulation/cmd"
	_ "github.com/insider/league-simulation/docs"
)

// @title           Football League Simulation API
// @version         1.0
// @description     REST API for simulating a four-team football league, editing match results, rolling back state, and calculating championship probabilities.
// @host            localhost:8080
// @BasePath        /api/v1

func main() {
	cmd.Execute()
}

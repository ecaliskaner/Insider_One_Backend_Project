package services

import "github.com/insider/league-simulation/models"

// LeagueService defines the core league operations interface.
// Interface-based design as required by the task.
type LeagueService interface {
	// GetStandings returns the current league table (PTS, W, D, L, GD)
	GetStandings() ([]models.Standing, error)

	// PlayNextWeek simulates the next week's matches and updates state
	PlayNextWeek() (*models.WeekResult, error)

	// PlayAll simulates all remaining matches in the season
	PlayAll() ([]models.WeekResult, error)

	// EditMatch edits a specific match result; recalculates standings and morale
	EditMatch(matchID int, homeScore int, awayScore int) (*models.Match, error)

	// GetPredictions runs Monte Carlo simulations for championship win %
	GetPredictions() ([]models.Prediction, error)

	// Rollback reverts database state to a specific week (Time Machine)
	Rollback(week int) error

	// GetTeamMetrics returns a team's current strength, morale, fatigue, market value
	GetTeamMetrics(teamID int) (*models.TeamMetrics, error)

	// Reset clears all data and regenerates
	Reset() error

	// GetCurrentWeek returns next unplayed week number
	GetCurrentWeek() (int, error)
}

// MatchEngine defines the interface for simulating match results.
// Implemented as a separate component (Adapter pattern) so the core logic remains pure.
type MatchEngine interface {
	// SimulateMatch generates a result based on team strengths, morale, fatigue, weather
	SimulateMatch(home, away models.Team, weather string) (homeGoals int, awayGoals int, events []models.MatchEvent)
}

// WeatherAdapter generates weather conditions for matches
type WeatherAdapter interface {
	GetWeather(city string) string
}

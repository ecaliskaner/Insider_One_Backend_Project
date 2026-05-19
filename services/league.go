package services

import (
	"context"
	"github.com/ecaliskaner/Insider_One_Backend_Project/models"
)

// LeagueService defines the core league operations interface.
// Interface-based design as required by the task.
type LeagueService interface {
	// GetStandings returns the current league table (PTS, W, D, L, GD)
	GetStandings(ctx context.Context) ([]models.Standing, error)

	// GetOverview returns the case-friendly current league screen payload
	GetOverview(ctx context.Context) (*models.LeagueOverview, error)

	// PlayNextWeek simulates the next week's matches and updates state
	PlayNextWeek(ctx context.Context) (*models.WeekResult, error)

	// PlayAll simulates all remaining matches in the season
	PlayAll(ctx context.Context) ([]models.WeekResult, error)

	// GetMatch retrieves a match and its events
	GetMatch(ctx context.Context, matchID int) (*models.Match, []models.MatchEvent, error)

	// EditMatch edits a specific match result; recalculates standings and morale
	EditMatch(ctx context.Context, matchID int, homeScore int, awayScore int) (*models.Match, error)

	// GetPredictions runs Monte Carlo simulations for championship win %
	GetPredictions(ctx context.Context) ([]models.Prediction, error)

	// Rollback reverts database state to a specific week.
	Rollback(ctx context.Context, week int) error

	// GetTeamMetrics returns a team's current strength, morale, fatigue, market value
	GetTeamMetrics(ctx context.Context, teamID int) (*models.TeamMetrics, error)

	// Reset clears all data and regenerates
	Reset(ctx context.Context) error

	// GetCurrentWeek returns next unplayed week number
	GetCurrentWeek(ctx context.Context) (int, error)
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

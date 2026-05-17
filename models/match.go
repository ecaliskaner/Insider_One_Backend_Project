package models

import "context"

// Match represents a single match between two teams
type Match struct {
	ID               int    `json:"id"`
	Week             int    `json:"week"`
	HomeTeamID       int    `json:"home_team_id"`
	AwayTeamID       int    `json:"away_team_id"`
	HomeScore        *int   `json:"home_score"`        // nil if not played
	AwayScore        *int   `json:"away_score"`        // nil if not played
	WeatherCondition string `json:"weather_condition"` // sunny, rainy, snowy, windy, foggy
	Status           string `json:"status"`            // scheduled, played, edited
	HomeTeam         string `json:"home_team,omitempty"`
	AwayTeam         string `json:"away_team,omitempty"`
}

// MatchEvent represents an event that occurred during a match
type MatchEvent struct {
	ID        int    `json:"id"`
	MatchID   int    `json:"match_id"`
	PlayerID  *int   `json:"player_id,omitempty"`
	EventType string `json:"event_type"` // Goal, Assist, Yellow Card, Red Card, Quantum VAR Decision, Injury
	Minute    int    `json:"minute"`
	Detail    string `json:"detail,omitempty"`
}

// Standing represents a team's position in the league table
type Standing struct {
	TeamID   int    `json:"team_id"`
	TeamName string `json:"team_name,omitempty"`
	Played   int    `json:"played"`
	Won      int    `json:"won"`
	Drawn    int    `json:"drawn"`
	Lost     int    `json:"lost"`
	GF       int    `json:"gf"` // goals for
	GA       int    `json:"ga"` // goals against
	GD       int    `json:"gd"` // goal difference
	Points   int    `json:"points"`
	Position int    `json:"position,omitempty"`
}

// WeekResult contains all match results for a single week
type WeekResult struct {
	Week    int     `json:"week"`
	Matches []Match `json:"matches"`
}

// LeagueOverview represents the case-friendly dashboard payload.
type LeagueOverview struct {
	CurrentWeek int          `json:"current_week"`
	Standings   []Standing   `json:"standings"`
	Weeks       []WeekResult `json:"weeks"`
	Predictions []Prediction `json:"predictions,omitempty"`
}

// Prediction represents Monte Carlo championship probability
type Prediction struct {
	TeamID   int     `json:"team_id"`
	TeamName string  `json:"team_name"`
	WinPct   float64 `json:"championship_win_pct"`
}

// MatchRepository defines the interface for match data access
type MatchRepository interface {
	GetAll(ctx context.Context) ([]Match, error)
	GetByWeek(ctx context.Context, week int) ([]Match, error)
	GetByID(ctx context.Context, id int) (*Match, error)
	Update(ctx context.Context, match *Match) error
	Create(ctx context.Context, match *Match) error
	GetCurrentWeek(ctx context.Context) (int, error)
	DeleteAll(ctx context.Context) error
	DeleteFromWeek(ctx context.Context, week int) error // For rollback
}

// MatchEventRepository defines the interface for match events
type MatchEventRepository interface {
	GetByMatchID(ctx context.Context, matchID int) ([]MatchEvent, error)
	Create(ctx context.Context, event *MatchEvent) error
	DeleteByMatchID(ctx context.Context, matchID int) error
	DeleteAll(ctx context.Context) error
	DeleteFromWeek(ctx context.Context, week int) error // For rollback
}

// StandingRepository defines the interface for standings snapshots
type StandingRepository interface {
	GetAll(ctx context.Context) ([]Standing, error)
	Upsert(ctx context.Context, standing *Standing) error
	DeleteAll(ctx context.Context) error
	RecalculateAll(ctx context.Context, matches []Match) error
}

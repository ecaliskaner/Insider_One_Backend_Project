package models

// Match represents a single match between two teams
type Match struct {
	ID               int    `json:"id"`
	Week             int    `json:"week"`
	HomeTeamID       int    `json:"home_team_id"`
	AwayTeamID       int    `json:"away_team_id"`
	HomeScore        *int   `json:"home_score"`         // nil if not played
	AwayScore        *int   `json:"away_score"`         // nil if not played
	WeatherCondition string `json:"weather_condition"`  // sunny, rainy, snowy, windy, foggy
	Status           string `json:"status"`             // scheduled, played, edited
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
	TeamID         int    `json:"team_id"`
	TeamName       string `json:"team_name,omitempty"`
	Played         int    `json:"played"`
	Won            int    `json:"won"`
	Drawn          int    `json:"drawn"`
	Lost           int    `json:"lost"`
	GF             int    `json:"gf"`  // goals for
	GA             int    `json:"ga"`  // goals against
	GD             int    `json:"gd"`  // goal difference
	Points         int    `json:"points"`
	Position       int    `json:"position,omitempty"`
}

// WeekResult contains all match results for a single week
type WeekResult struct {
	Week    int     `json:"week"`
	Matches []Match `json:"matches"`
}

// Prediction represents Monte Carlo championship probability
type Prediction struct {
	TeamID   int     `json:"team_id"`
	TeamName string  `json:"team_name"`
	WinPct   float64 `json:"championship_win_pct"`
}

// MatchRepository defines the interface for match data access
type MatchRepository interface {
	GetAll() ([]Match, error)
	GetByWeek(week int) ([]Match, error)
	GetByID(id int) (*Match, error)
	Update(match *Match) error
	Create(match *Match) error
	GetCurrentWeek() (int, error)
	DeleteAll() error
	DeleteFromWeek(week int) error // For rollback
}

// MatchEventRepository defines the interface for match events
type MatchEventRepository interface {
	GetByMatchID(matchID int) ([]MatchEvent, error)
	Create(event *MatchEvent) error
	DeleteByMatchID(matchID int) error
	DeleteAll() error
	DeleteFromWeek(week int) error // For rollback
}

// StandingRepository defines the interface for standings snapshots
type StandingRepository interface {
	GetAll() ([]Standing, error)
	Upsert(standing *Standing) error
	DeleteAll() error
	RecalculateAll(matches []Match) error
}

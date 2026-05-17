package models

import "context"

// Team represents a football team with advanced metrics
type Team struct {
	ID              int     `json:"id"`
	Name            string  `json:"name"`
	MarketValue     float64 `json:"market_value"`      // in millions
	BaseStrength    int     `json:"base_strength"`      // original strength (1-100)
	CurrentStrength int     `json:"current_strength"`   // adjusted by morale/fatigue
	Morale          float64 `json:"morale"`             // 0.0 - 1.0
	Fatigue         float64 `json:"fatigue"`            // 0.0 - 1.0 (higher = more tired)
	City            string  `json:"city"`
}

// TeamMetrics is the response for GET /api/v1/teams/{id}/metrics
type TeamMetrics struct {
	TeamID          int     `json:"team_id"`
	TeamName        string  `json:"team_name"`
	Strength        int     `json:"current_strength"`
	BaseStrength    int     `json:"base_strength"`
	Morale          float64 `json:"morale"`
	Fatigue         float64 `json:"fatigue"`
	MarketValue     float64 `json:"market_value"`
	City            string  `json:"city"`
}

// TeamRepository defines the interface for team data access
type TeamRepository interface {
	GetAll(ctx context.Context) ([]Team, error)
	GetByID(ctx context.Context, id int) (*Team, error)
	Create(ctx context.Context, team *Team) error
	Update(ctx context.Context, team *Team) error
	DeleteAll(ctx context.Context) error
}

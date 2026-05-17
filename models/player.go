package models

import "context"

// Player represents a football player on a team
type Player struct {
	ID       int    `json:"id"`
	TeamID   int    `json:"team_id"`
	Name     string `json:"name"`
	Position string `json:"position"` // GK, DEF, MID, FWD
	TeamName string `json:"team_name,omitempty"`
}

// PlayerRepository defines the interface for player data access
type PlayerRepository interface {
	GetAll(ctx context.Context) ([]Player, error)
	GetByTeamID(ctx context.Context, teamID int) ([]Player, error)
	GetByID(ctx context.Context, id int) (*Player, error)
	Create(ctx context.Context, player *Player) error
	DeleteAll(ctx context.Context) error
}

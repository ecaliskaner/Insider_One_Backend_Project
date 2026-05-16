package models

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
	GetAll() ([]Player, error)
	GetByTeamID(teamID int) ([]Player, error)
	GetByID(id int) (*Player, error)
	Create(player *Player) error
	DeleteAll() error
}

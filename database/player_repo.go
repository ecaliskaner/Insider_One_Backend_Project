package database

import (
	"database/sql"

	"github.com/insider/league-simulation/models"
)

// PlayerRepo implements models.PlayerRepository
type PlayerRepo struct {
	db *sql.DB
}

func NewPlayerRepo(db *sql.DB) *PlayerRepo {
	return &PlayerRepo{db: db}
}

func (r *PlayerRepo) GetAll() ([]models.Player, error) {
	rows, err := r.db.Query(`
		SELECT p.id, p.team_id, p.name, p.position, t.name 
		FROM players p JOIN teams t ON p.team_id = t.id ORDER BY p.team_id, p.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var players []models.Player
	for rows.Next() {
		var p models.Player
		rows.Scan(&p.ID, &p.TeamID, &p.Name, &p.Position, &p.TeamName)
		players = append(players, p)
	}
	return players, nil
}

func (r *PlayerRepo) GetByTeamID(teamID int) ([]models.Player, error) {
	rows, err := r.db.Query(`
		SELECT p.id, p.team_id, p.name, p.position, t.name 
		FROM players p JOIN teams t ON p.team_id = t.id 
		WHERE p.team_id = ? ORDER BY p.id`, teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var players []models.Player
	for rows.Next() {
		var p models.Player
		rows.Scan(&p.ID, &p.TeamID, &p.Name, &p.Position, &p.TeamName)
		players = append(players, p)
	}
	return players, nil
}

func (r *PlayerRepo) GetByID(id int) (*models.Player, error) {
	var p models.Player
	err := r.db.QueryRow(`
		SELECT p.id, p.team_id, p.name, p.position, t.name 
		FROM players p JOIN teams t ON p.team_id = t.id 
		WHERE p.id = ?`, id).Scan(&p.ID, &p.TeamID, &p.Name, &p.Position, &p.TeamName)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *PlayerRepo) Create(player *models.Player) error {
	result, err := r.db.Exec(
		"INSERT INTO players (team_id, name, position) VALUES (?, ?, ?)",
		player.TeamID, player.Name, player.Position,
	)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	player.ID = int(id)
	return nil
}

func (r *PlayerRepo) DeleteAll() error {
	_, err := r.db.Exec("DELETE FROM players")
	return err
}

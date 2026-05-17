package database

import (
	"context"

	"github.com/insider/league-simulation/models"
)

// PlayerRepo implements models.PlayerRepository
type PlayerRepo struct {
	db DBTX
}

func NewPlayerRepo(db DBTX) *PlayerRepo {
	return &PlayerRepo{db: db}
}

func (r *PlayerRepo) GetAll(ctx context.Context) ([]models.Player, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT p.id, p.team_id, p.name, p.position, t.name 
		FROM players p JOIN teams t ON p.team_id = t.id ORDER BY p.team_id, p.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var players []models.Player
	for rows.Next() {
		var p models.Player
		if err := rows.Scan(&p.ID, &p.TeamID, &p.Name, &p.Position, &p.TeamName); err != nil {
			return nil, err
		}
		players = append(players, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return players, nil
}

func (r *PlayerRepo) GetByTeamID(ctx context.Context, teamID int) ([]models.Player, error) {
	rows, err := r.db.QueryContext(ctx, `
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
		if err := rows.Scan(&p.ID, &p.TeamID, &p.Name, &p.Position, &p.TeamName); err != nil {
			return nil, err
		}
		players = append(players, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return players, nil
}

func (r *PlayerRepo) GetByID(ctx context.Context, id int) (*models.Player, error) {
	var p models.Player
	err := r.db.QueryRowContext(ctx, `
		SELECT p.id, p.team_id, p.name, p.position, t.name 
		FROM players p JOIN teams t ON p.team_id = t.id 
		WHERE p.id = ?`, id).Scan(&p.ID, &p.TeamID, &p.Name, &p.Position, &p.TeamName)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *PlayerRepo) Create(ctx context.Context, player *models.Player) error {
	result, err := r.db.ExecContext(ctx,
		"INSERT INTO players (team_id, name, position) VALUES (?, ?, ?)",
		player.TeamID, player.Name, player.Position,
	)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	player.ID = int(id)
	return nil
}

func (r *PlayerRepo) DeleteAll(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM players")
	return err
}

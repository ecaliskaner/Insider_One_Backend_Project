package database

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/insider/league-simulation/models"
)

// TeamRepo implements models.TeamRepository
type TeamRepo struct {
	db *sql.DB
}

func NewTeamRepo(db *sql.DB) *TeamRepo {
	return &TeamRepo{db: db}
}

func (r *TeamRepo) GetAll(ctx context.Context) ([]models.Team, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, name, market_value, base_strength, current_strength, morale, fatigue, city FROM teams ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var teams []models.Team
	for rows.Next() {
		var t models.Team
		if err := rows.Scan(&t.ID, &t.Name, &t.MarketValue, &t.BaseStrength, &t.CurrentStrength, &t.Morale, &t.Fatigue, &t.City); err != nil {
			return nil, err
		}
		teams = append(teams, t)
	}
	return teams, nil
}

func (r *TeamRepo) GetByID(ctx context.Context, id int) (*models.Team, error) {
	var t models.Team
	err := r.db.QueryRowContext(ctx,
		`SELECT id, name, market_value, base_strength, current_strength, morale, fatigue, city FROM teams WHERE id = ?`, id,
	).Scan(&t.ID, &t.Name, &t.MarketValue, &t.BaseStrength, &t.CurrentStrength, &t.Morale, &t.Fatigue, &t.City)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("team with id %d not found", id)
		}
		return nil, err
	}
	return &t, nil
}

func (r *TeamRepo) Create(ctx context.Context, team *models.Team) error {
	result, err := r.db.ExecContext(ctx,
		`INSERT INTO teams (name, market_value, base_strength, current_strength, morale, fatigue, city) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		team.Name, team.MarketValue, team.BaseStrength, team.CurrentStrength, team.Morale, team.Fatigue, team.City,
	)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	team.ID = int(id)
	return nil
}

func (r *TeamRepo) Update(ctx context.Context, team *models.Team) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE teams SET name=?, market_value=?, base_strength=?, current_strength=?, morale=?, fatigue=?, city=? WHERE id=?`,
		team.Name, team.MarketValue, team.BaseStrength, team.CurrentStrength, team.Morale, team.Fatigue, team.City, team.ID,
	)
	return err
}

func (r *TeamRepo) DeleteAll(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM teams")
	return err
}

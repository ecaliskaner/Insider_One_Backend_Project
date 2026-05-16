package database

import (
	"database/sql"
	"fmt"

	"github.com/insider/league-simulation/models"
)

// MatchRepo implements models.MatchRepository
type MatchRepo struct {
	db *sql.DB
}

func NewMatchRepo(db *sql.DB) *MatchRepo {
	return &MatchRepo{db: db}
}

func (r *MatchRepo) GetAll() ([]models.Match, error) {
	return r.queryMatches(`
		SELECT m.id, m.week, m.home_team_id, m.away_team_id, m.home_score, m.away_score,
		       m.weather_condition, m.status, ht.name, at.name
		FROM matches m
		JOIN teams ht ON m.home_team_id = ht.id
		JOIN teams at ON m.away_team_id = at.id
		ORDER BY m.week, m.id`)
}

func (r *MatchRepo) GetByWeek(week int) ([]models.Match, error) {
	return r.queryMatches(`
		SELECT m.id, m.week, m.home_team_id, m.away_team_id, m.home_score, m.away_score,
		       m.weather_condition, m.status, ht.name, at.name
		FROM matches m
		JOIN teams ht ON m.home_team_id = ht.id
		JOIN teams at ON m.away_team_id = at.id
		WHERE m.week = ?
		ORDER BY m.id`, week)
}

func (r *MatchRepo) GetByID(id int) (*models.Match, error) {
	matches, err := r.queryMatches(`
		SELECT m.id, m.week, m.home_team_id, m.away_team_id, m.home_score, m.away_score,
		       m.weather_condition, m.status, ht.name, at.name
		FROM matches m
		JOIN teams ht ON m.home_team_id = ht.id
		JOIN teams at ON m.away_team_id = at.id
		WHERE m.id = ?`, id)
	if err != nil {
		return nil, err
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("match with id %d not found", id)
	}
	return &matches[0], nil
}

func (r *MatchRepo) Update(match *models.Match) error {
	_, err := r.db.Exec(
		`UPDATE matches SET home_score=?, away_score=?, weather_condition=?, status=? WHERE id=?`,
		match.HomeScore, match.AwayScore, match.WeatherCondition, match.Status, match.ID,
	)
	return err
}

func (r *MatchRepo) Create(match *models.Match) error {
	result, err := r.db.Exec(
		`INSERT INTO matches (week, home_team_id, away_team_id, home_score, away_score, weather_condition, status)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		match.Week, match.HomeTeamID, match.AwayTeamID, match.HomeScore, match.AwayScore,
		match.WeatherCondition, match.Status,
	)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	match.ID = int(id)
	return nil
}

func (r *MatchRepo) GetCurrentWeek() (int, error) {
	var week sql.NullInt64
	err := r.db.QueryRow("SELECT MIN(week) FROM matches WHERE status = 'scheduled'").Scan(&week)
	if err != nil {
		return 0, err
	}
	if !week.Valid {
		var maxWeek int
		r.db.QueryRow("SELECT COALESCE(MAX(week), 0) FROM matches").Scan(&maxWeek)
		return maxWeek + 1, nil
	}
	return int(week.Int64), nil
}

func (r *MatchRepo) DeleteAll() error {
	_, err := r.db.Exec("DELETE FROM matches")
	return err
}

// DeleteFromWeek removes all matches from a given week onward (for rollback)
func (r *MatchRepo) DeleteFromWeek(week int) error {
	// Reset matches from this week onward to scheduled state
	_, err := r.db.Exec(
		`UPDATE matches SET home_score = NULL, away_score = NULL, status = 'scheduled' WHERE week >= ?`, week,
	)
	return err
}

func (r *MatchRepo) queryMatches(query string, args ...interface{}) ([]models.Match, error) {
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var matches []models.Match
	for rows.Next() {
		var m models.Match
		if err := rows.Scan(
			&m.ID, &m.Week, &m.HomeTeamID, &m.AwayTeamID,
			&m.HomeScore, &m.AwayScore, &m.WeatherCondition, &m.Status,
			&m.HomeTeam, &m.AwayTeam,
		); err != nil {
			return nil, err
		}
		matches = append(matches, m)
	}
	return matches, nil
}

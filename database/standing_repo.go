package database

import (
	"database/sql"

	"github.com/insider/league-simulation/models"
)

// StandingRepo implements models.StandingRepository
type StandingRepo struct {
	db *sql.DB
}

func NewStandingRepo(db *sql.DB) *StandingRepo {
	return &StandingRepo{db: db}
}

func (r *StandingRepo) GetAll() ([]models.Standing, error) {
	rows, err := r.db.Query(`
		SELECT s.team_id, t.name, s.played, s.won, s.drawn, s.lost, s.gf, s.ga, s.gd, s.points
		FROM standings s
		JOIN teams t ON s.team_id = t.id
		ORDER BY s.points DESC, s.gd DESC, s.gf DESC, t.name ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var standings []models.Standing
	pos := 1
	for rows.Next() {
		var s models.Standing
		rows.Scan(&s.TeamID, &s.TeamName, &s.Played, &s.Won, &s.Drawn, &s.Lost,
			&s.GF, &s.GA, &s.GD, &s.Points)
		s.Position = pos
		pos++
		standings = append(standings, s)
	}
	return standings, nil
}

func (r *StandingRepo) Upsert(standing *models.Standing) error {
	_, err := r.db.Exec(`
		INSERT INTO standings (team_id, played, won, drawn, lost, gf, ga, gd, points)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(team_id) DO UPDATE SET
			played=excluded.played, won=excluded.won, drawn=excluded.drawn,
			lost=excluded.lost, gf=excluded.gf, ga=excluded.ga, gd=excluded.gd,
			points=excluded.points`,
		standing.TeamID, standing.Played, standing.Won, standing.Drawn, standing.Lost,
		standing.GF, standing.GA, standing.GD, standing.Points,
	)
	return err
}

func (r *StandingRepo) DeleteAll() error {
	_, err := r.db.Exec("DELETE FROM standings")
	return err
}

// RecalculateAll recomputes standings from all played matches
func (r *StandingRepo) RecalculateAll(matches []models.Match) error {
	// Reset all standings
	_, err := r.db.Exec("UPDATE standings SET played=0, won=0, drawn=0, lost=0, gf=0, ga=0, gd=0, points=0")
	if err != nil {
		return err
	}

	// Aggregate from played matches
	standingMap := make(map[int]*models.Standing)

	// Init from existing standings rows
	rows, err := r.db.Query("SELECT team_id FROM standings")
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var tid int
		rows.Scan(&tid)
		standingMap[tid] = &models.Standing{TeamID: tid}
	}

	for _, m := range matches {
		if m.Status != "played" && m.Status != "edited" {
			continue
		}
		if m.HomeScore == nil || m.AwayScore == nil {
			continue
		}

		home, ok := standingMap[m.HomeTeamID]
		if !ok {
			home = &models.Standing{TeamID: m.HomeTeamID}
			standingMap[m.HomeTeamID] = home
		}
		away, ok := standingMap[m.AwayTeamID]
		if !ok {
			away = &models.Standing{TeamID: m.AwayTeamID}
			standingMap[m.AwayTeamID] = away
		}

		hg, ag := *m.HomeScore, *m.AwayScore
		home.Played++
		away.Played++
		home.GF += hg
		home.GA += ag
		away.GF += ag
		away.GA += hg

		if hg > ag {
			home.Won++
			home.Points += 3
			away.Lost++
		} else if hg < ag {
			away.Won++
			away.Points += 3
			home.Lost++
		} else {
			home.Drawn++
			away.Drawn++
			home.Points++
			away.Points++
		}
	}

	// Update GD and persist
	for _, s := range standingMap {
		s.GD = s.GF - s.GA
		if err := r.Upsert(s); err != nil {
			return err
		}
	}

	return nil
}

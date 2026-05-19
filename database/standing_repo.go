package database

import (
	"context"
	"database/sql"

	"github.com/ecaliskaner/Insider_One_Backend_Project/models"
)

// StandingRepo implements models.StandingRepository
type StandingRepo struct {
	db DBTX
}

func NewStandingRepo(db DBTX) *StandingRepo {
	return &StandingRepo{db: db}
}

func (r *StandingRepo) GetAll(ctx context.Context) ([]models.Standing, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT s.team_id, t.name, s.played, s.won, s.drawn, s.lost, s.gf, s.ga, s.gd, s.points
		FROM standings s
		JOIN teams t ON s.team_id = t.id
		ORDER BY s.points DESC, s.gd DESC, s.gf DESC, t.name ASC`)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	var standings []models.Standing
	pos := 1
	for rows.Next() {
		var s models.Standing
		if err := rows.Scan(&s.TeamID, &s.TeamName, &s.Played, &s.Won, &s.Drawn, &s.Lost,
			&s.GF, &s.GA, &s.GD, &s.Points); err != nil {
			return nil, err
		}
		s.Position = pos
		pos++
		standings = append(standings, s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	matches, err := r.getAllPlayedMatches(ctx)
	if err != nil {
		return nil, err
	}

	return models.RankStandings(standings, matches), nil
}

// getAllPlayedMatches fetches matches needed for tiebreaker calculations
func (r *StandingRepo) getAllPlayedMatches(ctx context.Context) ([]models.Match, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, home_team_id, away_team_id, home_score, away_score, status 
		FROM matches 
		WHERE status='played' OR status='edited'`)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	var matches []models.Match
	for rows.Next() {
		var m models.Match
		var hs, as sql.NullInt64
		if err := rows.Scan(&m.ID, &m.HomeTeamID, &m.AwayTeamID, &hs, &as, &m.Status); err != nil {
			return nil, err
		}
		if hs.Valid && as.Valid {
			hVal := int(hs.Int64)
			aVal := int(as.Int64)
			m.HomeScore = &hVal
			m.AwayScore = &aVal
		}
		matches = append(matches, m)
	}
	return matches, nil
}

func (r *StandingRepo) Upsert(ctx context.Context, standing *models.Standing) error {
	_, err := r.db.ExecContext(ctx, `
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

func (r *StandingRepo) DeleteAll(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM standings")
	return err
}

// RecalculateAll recomputes standings from all played matches
func (r *StandingRepo) RecalculateAll(ctx context.Context, matches []models.Match) error {
	// Reset all standings
	_, err := r.db.ExecContext(ctx, "UPDATE standings SET played=0, won=0, drawn=0, lost=0, gf=0, ga=0, gd=0, points=0")
	if err != nil {
		return err
	}

	// Aggregate from played matches
	standingMap := make(map[int]*models.Standing)

	// Init from existing standings rows
	rows, err := r.db.QueryContext(ctx, "SELECT team_id FROM standings")
	if err != nil {
		return err
	}
	defer func() {
		_ = rows.Close()
	}()
	for rows.Next() {
		var tid int
		if err := rows.Scan(&tid); err != nil {
			return err
		}
		standingMap[tid] = &models.Standing{TeamID: tid}
	}
	if err := rows.Err(); err != nil {
		return err
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
		if err := r.Upsert(ctx, s); err != nil {
			return err
		}
	}

	return nil
}

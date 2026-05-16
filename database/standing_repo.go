package database

import (
	"database/sql"
	"sort"

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

	// Apply Premier League Head-to-Head Tiebreakers
	matches, err := r.getAllPlayedMatches()
	if err != nil {
		return nil, err
	}
	
	return r.applyTiebreakers(standings, matches), nil
}

// getAllPlayedMatches fetches matches needed for tiebreaker calculations
func (r *StandingRepo) getAllPlayedMatches() ([]models.Match, error) {
	rows, err := r.db.Query(`
		SELECT id, home_team_id, away_team_id, home_score, away_score, status 
		FROM matches 
		WHERE status='played' OR status='edited'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

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

// applyTiebreakers resolves ties according to official Premier League rules
func (r *StandingRepo) applyTiebreakers(standings []models.Standing, matches []models.Match) []models.Standing {
	if len(standings) == 0 {
		return standings
	}

	// 1. Group teams that are perfectly tied on Points, GD, and GF
	var groups [][]*models.Standing
	var currentGroup []*models.Standing

	for i := 0; i < len(standings); i++ {
		if len(currentGroup) == 0 {
			currentGroup = append(currentGroup, &standings[i])
			continue
		}

		last := currentGroup[0]
		curr := &standings[i]

		if curr.Points == last.Points && curr.GD == last.GD && curr.GF == last.GF {
			currentGroup = append(currentGroup, curr)
		} else {
			groups = append(groups, currentGroup)
			currentGroup = []*models.Standing{curr}
		}
	}
	if len(currentGroup) > 0 {
		groups = append(groups, currentGroup)
	}

	// 2. Sort tied groups using Head-to-Head rules
	var sortedStandings []models.Standing
	pos := 1
	for _, group := range groups {
		if len(group) > 1 {
			r.sortTiedGroup(group, matches)
		}
		for _, s := range group {
			s.Position = pos
			pos++
			sortedStandings = append(sortedStandings, *s)
		}
	}

	return sortedStandings
}

// sortTiedGroup evaluates the "mini-league" of tied teams
func (r *StandingRepo) sortTiedGroup(group []*models.Standing, matches []models.Match) {
	tiedIDs := make(map[int]bool)
	for _, s := range group {
		tiedIDs[s.TeamID] = true
	}

	type h2hStat struct {
		Points    int
		AwayGoals int
	}
	stats := make(map[int]*h2hStat)
	for id := range tiedIDs {
		stats[id] = &h2hStat{}
	}

	// Filter matches to strictly those between tied teams
	for _, m := range matches {
		if tiedIDs[m.HomeTeamID] && tiedIDs[m.AwayTeamID] && m.HomeScore != nil && m.AwayScore != nil {
			hg, ag := *m.HomeScore, *m.AwayScore
			if hg > ag {
				stats[m.HomeTeamID].Points += 3
			} else if hg < ag {
				stats[m.AwayTeamID].Points += 3
			} else {
				stats[m.HomeTeamID].Points += 1
				stats[m.AwayTeamID].Points += 1
			}
			// Away goals tiebreaker
			stats[m.AwayTeamID].AwayGoals += ag
		}
	}

	// Sort the group
	sort.Slice(group, func(i, j int) bool {
		s1 := stats[group[i].TeamID]
		s2 := stats[group[j].TeamID]

		// 1. Head-to-Head Points
		if s1.Points != s2.Points {
			return s1.Points > s2.Points
		}
		// 2. Head-to-Head Away Goals
		if s1.AwayGoals != s2.AwayGoals {
			return s1.AwayGoals > s2.AwayGoals
		}
		// 3. Fallback: Alphabetical
		return group[i].TeamName < group[j].TeamName
	})
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

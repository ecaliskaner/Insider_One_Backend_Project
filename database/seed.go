package database

import (
	"fmt"
	"log"

	"github.com/insider/league-simulation/models"
)

// DefaultTeams returns the 4 teams with realistic attributes
func DefaultTeams() []models.Team {
	return []models.Team{
		{Name: "Manchester City", MarketValue: 1200.0, BaseStrength: 90, CurrentStrength: 90, Morale: 0.7, Fatigue: 0.0, City: "Manchester"},
		{Name: "Arsenal", MarketValue: 1050.0, BaseStrength: 85, CurrentStrength: 85, Morale: 0.7, Fatigue: 0.0, City: "London"},
		{Name: "Liverpool", MarketValue: 980.0, BaseStrength: 82, CurrentStrength: 82, Morale: 0.7, Fatigue: 0.0, City: "Liverpool"},
		{Name: "Chelsea", MarketValue: 900.0, BaseStrength: 75, CurrentStrength: 75, Morale: 0.7, Fatigue: 0.0, City: "London"},
	}
}

// DefaultPlayers returns realistic players for each team
func DefaultPlayers() map[string][]models.Player {
	return map[string][]models.Player{
		"Manchester City": {
			{Name: "Ederson", Position: "GK"},
			{Name: "Rúben Dias", Position: "DEF"},
			{Name: "Kyle Walker", Position: "DEF"},
			{Name: "Rodri", Position: "MID"},
			{Name: "Kevin De Bruyne", Position: "MID"},
			{Name: "Bernardo Silva", Position: "MID"},
			{Name: "Erling Haaland", Position: "FWD"},
			{Name: "Phil Foden", Position: "FWD"},
		},
		"Arsenal": {
			{Name: "David Raya", Position: "GK"},
			{Name: "William Saliba", Position: "DEF"},
			{Name: "Ben White", Position: "DEF"},
			{Name: "Declan Rice", Position: "MID"},
			{Name: "Martin Ødegaard", Position: "MID"},
			{Name: "Bukayo Saka", Position: "FWD"},
			{Name: "Gabriel Jesus", Position: "FWD"},
			{Name: "Kai Havertz", Position: "FWD"},
		},
		"Liverpool": {
			{Name: "Alisson", Position: "GK"},
			{Name: "Virgil van Dijk", Position: "DEF"},
			{Name: "Trent Alexander-Arnold", Position: "DEF"},
			{Name: "Alexis Mac Allister", Position: "MID"},
			{Name: "Dominik Szoboszlai", Position: "MID"},
			{Name: "Mohamed Salah", Position: "FWD"},
			{Name: "Darwin Núñez", Position: "FWD"},
			{Name: "Luis Díaz", Position: "FWD"},
		},
		"Chelsea": {
			{Name: "Robert Sánchez", Position: "GK"},
			{Name: "Reece James", Position: "DEF"},
			{Name: "Thiago Silva", Position: "DEF"},
			{Name: "Enzo Fernández", Position: "MID"},
			{Name: "Moisés Caicedo", Position: "MID"},
			{Name: "Cole Palmer", Position: "FWD"},
			{Name: "Nicolas Jackson", Position: "FWD"},
			{Name: "Raheem Sterling", Position: "FWD"},
		},
	}
}

// SeedTeams inserts the default teams if they don't exist
func SeedTeams(db *DB) error {
	var count int
	err := db.Conn.QueryRow("SELECT COUNT(*) FROM teams").Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to count teams: %w", err)
	}
	if count > 0 {
		log.Println("⏭ Teams already seeded, skipping...")
		return nil
	}

	teams := DefaultTeams()
	for _, team := range teams {
		_, err := db.Conn.Exec(
			`INSERT INTO teams (name, market_value, base_strength, current_strength, morale, fatigue, city)
			 VALUES (?, ?, ?, ?, ?, ?, ?)`,
			team.Name, team.MarketValue, team.BaseStrength, team.CurrentStrength,
			team.Morale, team.Fatigue, team.City,
		)
		if err != nil {
			return fmt.Errorf("failed to seed team %s: %w", team.Name, err)
		}
	}

	log.Printf("✅ Seeded %d teams\n", len(teams))
	return nil
}

// SeedPlayers inserts the default players if they don't exist
func SeedPlayers(db *DB) error {
	var count int
	err := db.Conn.QueryRow("SELECT COUNT(*) FROM players").Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to count players: %w", err)
	}
	if count > 0 {
		log.Println("⏭ Players already seeded, skipping...")
		return nil
	}

	// Get team IDs by name
	teamIDs := make(map[string]int)
	rows, err := db.Conn.Query("SELECT id, name FROM teams")
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id int
		var name string
		rows.Scan(&id, &name)
		teamIDs[name] = id
	}

	allPlayers := DefaultPlayers()
	total := 0
	for teamName, players := range allPlayers {
		teamID := teamIDs[teamName]
		for _, p := range players {
			_, err := db.Conn.Exec(
				"INSERT INTO players (team_id, name, position) VALUES (?, ?, ?)",
				teamID, p.Name, p.Position,
			)
			if err != nil {
				return fmt.Errorf("failed to seed player %s: %w", p.Name, err)
			}
			total++
		}
	}

	log.Printf("✅ Seeded %d players\n", total)
	return nil
}

// SeedStandings initializes standings for all teams
func SeedStandings(db *DB) error {
	var count int
	err := db.Conn.QueryRow("SELECT COUNT(*) FROM standings").Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to count standings: %w", err)
	}
	if count > 0 {
		log.Println("⏭ Standings already seeded, skipping...")
		return nil
	}

	rows, err := db.Conn.Query("SELECT id FROM teams")
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		rows.Scan(&id)
		_, err := db.Conn.Exec(
			"INSERT INTO standings (team_id, played, won, drawn, lost, gf, ga, gd, points) VALUES (?, 0, 0, 0, 0, 0, 0, 0, 0)", id,
		)
		if err != nil {
			return fmt.Errorf("failed to seed standings: %w", err)
		}
	}

	log.Println("✅ Initialized standings")
	return nil
}

// GenerateSchedule creates the full round-robin schedule (home & away)
// 4 teams = 6 unique matchups × 2 = 12 matches over 6 weeks, 2 matches/week
func GenerateSchedule(db *DB) error {
	var count int
	err := db.Conn.QueryRow("SELECT COUNT(*) FROM matches").Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to count matches: %w", err)
	}
	if count > 0 {
		log.Println("⏭ Schedule already generated, skipping...")
		return nil
	}

	rows, err := db.Conn.Query("SELECT id FROM teams ORDER BY id")
	if err != nil {
		return fmt.Errorf("failed to get teams: %w", err)
	}
	defer rows.Close()

	var teamIDs []int
	for rows.Next() {
		var id int
		rows.Scan(&id)
		teamIDs = append(teamIDs, id)
	}

	if len(teamIDs) != 4 {
		return fmt.Errorf("expected 4 teams, got %d", len(teamIDs))
	}

	type matchup struct {
		home int
		away int
	}

	firstLeg := [][]matchup{
		{{teamIDs[0], teamIDs[1]}, {teamIDs[2], teamIDs[3]}},
		{{teamIDs[0], teamIDs[2]}, {teamIDs[1], teamIDs[3]}},
		{{teamIDs[0], teamIDs[3]}, {teamIDs[1], teamIDs[2]}},
	}

	tx, err := db.Conn.Begin()
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare("INSERT INTO matches (week, home_team_id, away_team_id, weather_condition, status) VALUES (?, ?, ?, 'sunny', 'scheduled')")
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmt.Close()

	// First leg (weeks 1-3)
	for week, matches := range firstLeg {
		for _, m := range matches {
			if _, err := stmt.Exec(week+1, m.home, m.away); err != nil {
				tx.Rollback()
				return err
			}
		}
	}

	// Second leg (weeks 4-6) - reversed home/away
	for week, matches := range firstLeg {
		for _, m := range matches {
			if _, err := stmt.Exec(week+4, m.away, m.home); err != nil {
				tx.Rollback()
				return err
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	log.Println("✅ Generated 12 matches across 6 weeks")
	return nil
}

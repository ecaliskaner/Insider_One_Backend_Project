package database

import (
	"context"
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

func ResetAutoIncrement(ctx context.Context, store DBTX) error {
	_, err := store.ExecContext(ctx, `
		DELETE FROM sqlite_sequence
		WHERE name IN ('teams', 'players', 'matches', 'match_events')`,
	)
	return err
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
	return SeedTeamsContext(context.Background(), db.Conn)
}

func SeedTeamsContext(ctx context.Context, store DBTX) error {
	var count int
	err := store.QueryRowContext(ctx, "SELECT COUNT(*) FROM teams").Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to count teams: %w", err)
	}
	if count > 0 {
		log.Println("⏭ Teams already seeded, skipping...")
		return nil
	}

	teams := DefaultTeams()
	for _, team := range teams {
		_, err := store.ExecContext(ctx,
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
	return SeedPlayersContext(context.Background(), db.Conn)
}

func SeedPlayersContext(ctx context.Context, store DBTX) error {
	var count int
	err := store.QueryRowContext(ctx, "SELECT COUNT(*) FROM players").Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to count players: %w", err)
	}
	if count > 0 {
		log.Println("⏭ Players already seeded, skipping...")
		return nil
	}

	// Get team IDs by name
	teamIDs := make(map[string]int)
	rows, err := store.QueryContext(ctx, "SELECT id, name FROM teams")
	if err != nil {
		return err
	}
	for rows.Next() {
		var id int
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			rows.Close()
			return err
		}
		teamIDs[name] = id
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}

	allPlayers := DefaultPlayers()
	total := 0
	for teamName, players := range allPlayers {
		teamID := teamIDs[teamName]
		for _, p := range players {
			_, err := store.ExecContext(ctx,
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
	return SeedStandingsContext(context.Background(), db.Conn)
}

func SeedStandingsContext(ctx context.Context, store DBTX) error {
	var count int
	err := store.QueryRowContext(ctx, "SELECT COUNT(*) FROM standings").Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to count standings: %w", err)
	}
	if count > 0 {
		log.Println("⏭ Standings already seeded, skipping...")
		return nil
	}

	rows, err := store.QueryContext(ctx, "SELECT id FROM teams")
	if err != nil {
		return err
	}

	var teamIDs []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		teamIDs = append(teamIDs, id)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}

	for _, id := range teamIDs {
		_, err := store.ExecContext(ctx,
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
	return GenerateScheduleContext(context.Background(), db.Conn)
}

func GenerateScheduleContext(ctx context.Context, store DBTX) error {
	var count int
	err := store.QueryRowContext(ctx, "SELECT COUNT(*) FROM matches").Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to count matches: %w", err)
	}
	if count > 0 {
		log.Println("⏭ Schedule already generated, skipping...")
		return nil
	}

	rows, err := store.QueryContext(ctx, "SELECT id FROM teams ORDER BY id")
	if err != nil {
		return fmt.Errorf("failed to get teams: %w", err)
	}

	var teamIDs []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return fmt.Errorf("scan team id: %w", err)
		}
		teamIDs = append(teamIDs, id)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("iterate team ids: %w", err)
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("close team rows: %w", err)
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

	// First leg (weeks 1-3)
	for week, matches := range firstLeg {
		for _, m := range matches {
			if _, err := store.ExecContext(ctx,
				"INSERT INTO matches (week, home_team_id, away_team_id, weather_condition, status) VALUES (?, ?, ?, 'sunny', 'scheduled')",
				week+1, m.home, m.away,
			); err != nil {
				return err
			}
		}
	}

	// Second leg (weeks 4-6) - reversed home/away
	for week, matches := range firstLeg {
		for _, m := range matches {
			if _, err := store.ExecContext(ctx,
				"INSERT INTO matches (week, home_team_id, away_team_id, weather_condition, status) VALUES (?, ?, ?, 'sunny', 'scheduled')",
				week+4, m.away, m.home,
			); err != nil {
				return err
			}
		}
	}

	log.Println("✅ Generated 12 matches across 6 weeks")
	return nil
}

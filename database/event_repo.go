package database

import (
	"database/sql"

	"github.com/insider/league-simulation/models"
)

// EventRepo implements models.MatchEventRepository
type EventRepo struct {
	db *sql.DB
}

func NewEventRepo(db *sql.DB) *EventRepo {
	return &EventRepo{db: db}
}

func (r *EventRepo) GetByMatchID(matchID int) ([]models.MatchEvent, error) {
	rows, err := r.db.Query(
		`SELECT id, match_id, player_id, event_type, minute, detail 
		 FROM match_events WHERE match_id = ? ORDER BY minute`, matchID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []models.MatchEvent
	for rows.Next() {
		var e models.MatchEvent
		rows.Scan(&e.ID, &e.MatchID, &e.PlayerID, &e.EventType, &e.Minute, &e.Detail)
		events = append(events, e)
	}
	return events, nil
}

func (r *EventRepo) Create(event *models.MatchEvent) error {
	result, err := r.db.Exec(
		`INSERT INTO match_events (match_id, player_id, event_type, minute, detail) VALUES (?, ?, ?, ?, ?)`,
		event.MatchID, event.PlayerID, event.EventType, event.Minute, event.Detail,
	)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	event.ID = int(id)
	return nil
}

func (r *EventRepo) DeleteByMatchID(matchID int) error {
	_, err := r.db.Exec("DELETE FROM match_events WHERE match_id = ?", matchID)
	return err
}

func (r *EventRepo) DeleteAll() error {
	_, err := r.db.Exec("DELETE FROM match_events")
	return err
}

// DeleteFromWeek removes events for matches from a given week onward
func (r *EventRepo) DeleteFromWeek(week int) error {
	_, err := r.db.Exec(
		`DELETE FROM match_events WHERE match_id IN (SELECT id FROM matches WHERE week >= ?)`, week,
	)
	return err
}

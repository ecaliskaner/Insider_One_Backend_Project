# SQL Schema And Queries

This project uses SQLite through Go's `database/sql` package. Migrations live in `database/migrations`, and repository queries live under `database/*_repo.go`.

## Tables

### teams

Stores team identity and dynamic simulation metrics.

```sql
CREATE TABLE IF NOT EXISTS teams (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    market_value REAL NOT NULL DEFAULT 100.0,
    base_strength INTEGER NOT NULL DEFAULT 50,
    current_strength INTEGER NOT NULL DEFAULT 50,
    morale REAL NOT NULL DEFAULT 0.5,
    fatigue REAL NOT NULL DEFAULT 0.0,
    city TEXT NOT NULL DEFAULT ''
);
```

### players

Stores players assigned to seeded teams.

```sql
CREATE TABLE IF NOT EXISTS players (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    team_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    position TEXT NOT NULL,
    FOREIGN KEY (team_id) REFERENCES teams(id)
);
```

### matches

Stores the six-week double round-robin schedule and match results.

```sql
CREATE TABLE IF NOT EXISTS matches (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    week INTEGER NOT NULL,
    home_team_id INTEGER NOT NULL,
    away_team_id INTEGER NOT NULL,
    home_score INTEGER DEFAULT NULL,
    away_score INTEGER DEFAULT NULL,
    weather_condition TEXT NOT NULL DEFAULT 'sunny',
    status TEXT NOT NULL DEFAULT 'scheduled',
    FOREIGN KEY (home_team_id) REFERENCES teams(id),
    FOREIGN KEY (away_team_id) REFERENCES teams(id)
);
```

`status` values:

- `scheduled`: not played yet
- `played`: simulated by the match engine
- `edited`: manually edited through the API

### match_events

Stores generated match events such as goals, injuries, and VAR events.

```sql
CREATE TABLE IF NOT EXISTS match_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    match_id INTEGER NOT NULL,
    player_id INTEGER,
    event_type TEXT NOT NULL,
    minute INTEGER NOT NULL DEFAULT 0,
    detail TEXT DEFAULT '',
    FOREIGN KEY (match_id) REFERENCES matches(id),
    FOREIGN KEY (player_id) REFERENCES players(id)
);
```

### standings

Stores the current league table. It is rebuilt from played/edited matches after mutations.

```sql
CREATE TABLE IF NOT EXISTS standings (
    team_id INTEGER PRIMARY KEY,
    played INTEGER NOT NULL DEFAULT 0,
    won INTEGER NOT NULL DEFAULT 0,
    drawn INTEGER NOT NULL DEFAULT 0,
    lost INTEGER NOT NULL DEFAULT 0,
    gf INTEGER NOT NULL DEFAULT 0,
    ga INTEGER NOT NULL DEFAULT 0,
    gd INTEGER NOT NULL DEFAULT 0,
    points INTEGER NOT NULL DEFAULT 0,
    FOREIGN KEY (team_id) REFERENCES teams(id)
);
```

## Indexes

```sql
CREATE INDEX IF NOT EXISTS idx_matches_week ON matches(week);
CREATE INDEX IF NOT EXISTS idx_matches_status ON matches(status);
CREATE INDEX IF NOT EXISTS idx_match_events_match ON match_events(match_id);
CREATE INDEX IF NOT EXISTS idx_players_team ON players(team_id);
```

## Key Queries

### Current Standings

Used by `StandingRepo.GetAll`.

```sql
SELECT s.team_id, t.name, s.played, s.won, s.drawn, s.lost,
       s.gf, s.ga, s.gd, s.points
FROM standings s
JOIN teams t ON s.team_id = t.id
ORDER BY s.points DESC, s.gd DESC, s.gf DESC, t.name ASC;
```

The repository applies an additional tied-team head-to-head sort in Go when teams are equal on points, goal difference, and goals for.

### Current Week

Used by `MatchRepo.GetCurrentWeek`.

```sql
SELECT MIN(week)
FROM matches
WHERE status = 'scheduled';
```

If there are no scheduled matches, the service returns `MAX(week) + 1`, which is `7` after the six-week season has completed.

### All Matches With Team Names

Used by `MatchRepo.GetAll`.

```sql
SELECT m.id, m.week, m.home_team_id, m.away_team_id,
       m.home_score, m.away_score, m.weather_condition, m.status,
       ht.name, at.name
FROM matches m
JOIN teams ht ON m.home_team_id = ht.id
JOIN teams at ON m.away_team_id = at.id
ORDER BY m.week, m.id;
```

### Matches By Week

Used by `MatchRepo.GetByWeek`.

```sql
SELECT m.id, m.week, m.home_team_id, m.away_team_id,
       m.home_score, m.away_score, m.weather_condition, m.status,
       ht.name, at.name
FROM matches m
JOIN teams ht ON m.home_team_id = ht.id
JOIN teams at ON m.away_team_id = at.id
WHERE m.week = ?
ORDER BY m.id;
```

### Update A Match Result

Used by simulation and manual edit flows.

```sql
UPDATE matches
SET home_score = ?,
    away_score = ?,
    weather_condition = ?,
    status = ?
WHERE id = ?;
```

### Delete Events For Edited Match

Used before rebuilding state after a manual edit.

```sql
DELETE FROM match_events
WHERE match_id = ?;
```

### Roll Back From A Week

Events for target/future weeks are removed, and target/future matches are reset to scheduled.

```sql
DELETE FROM match_events
WHERE match_id IN (
    SELECT id FROM matches WHERE week >= ?
);
```

```sql
UPDATE matches
SET home_score = NULL,
    away_score = NULL,
    status = 'scheduled'
WHERE week >= ?;
```

### Reset Standings Before Rebuild

Used by `StandingRepo.RecalculateAll`.

```sql
UPDATE standings
SET played = 0,
    won = 0,
    drawn = 0,
    lost = 0,
    gf = 0,
    ga = 0,
    gd = 0,
    points = 0;
```

### Upsert Recalculated Standing

Used after aggregating played/edited matches in Go.

```sql
INSERT INTO standings (team_id, played, won, drawn, lost, gf, ga, gd, points)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(team_id) DO UPDATE SET
    played = excluded.played,
    won = excluded.won,
    drawn = excluded.drawn,
    lost = excluded.lost,
    gf = excluded.gf,
    ga = excluded.ga,
    gd = excluded.gd,
    points = excluded.points;
```

## Transaction Boundaries

The following service operations run inside SQL transactions:

- playing the next week;
- editing a match result;
- rolling back to a week;
- resetting the league.

This prevents partial updates across `matches`, `match_events`, `standings`, and `teams` when one step fails.

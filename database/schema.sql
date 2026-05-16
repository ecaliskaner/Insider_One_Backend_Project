-- Football League Simulation Schema (v2)
-- SQLite compatible
-- Matches the prompt.md SQL Schema Requirements (Section 6)

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

CREATE TABLE IF NOT EXISTS players (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    team_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    position TEXT NOT NULL,
    FOREIGN KEY (team_id) REFERENCES teams(id)
);

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

-- Indexes
CREATE INDEX IF NOT EXISTS idx_matches_week ON matches(week);
CREATE INDEX IF NOT EXISTS idx_matches_status ON matches(status);
CREATE INDEX IF NOT EXISTS idx_match_events_match ON match_events(match_id);
CREATE INDEX IF NOT EXISTS idx_players_team ON players(team_id);

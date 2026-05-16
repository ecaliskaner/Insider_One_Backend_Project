# ⚽ Football League Simulation API v1

A GoLang REST API that simulates a 4-team football league with Premier League scoring rules, strength/morale/fatigue-based match simulation, weather effects, match events (including Quantum VAR Decisions), Monte Carlo championship predictions, and a Time Machine rollback feature.

## 🏗️ Architecture

Built using **strict interface-based design** and **struct composition**. External systems (Weather, Match Simulation) are implemented as **Adapters** so the core simulation logic remains pure and testable.

```
insider/
├── main.go                    # Entry point, DI wiring
├── models/
│   ├── team.go                # Team struct & TeamRepository interface
│   ├── player.go              # Player struct & PlayerRepository interface
│   └── match.go               # Match, MatchEvent, Standing & repository interfaces
├── database/
│   ├── schema.sql             # SQL schema (5 tables, embedded)
│   ├── db.go                  # DB connection & init
│   ├── seed.go                # Teams, players, standings, schedule seeding
│   ├── team_repo.go           # TeamRepository impl
│   ├── player_repo.go         # PlayerRepository impl
│   ├── match_repo.go          # MatchRepository impl (with rollback)
│   ├── event_repo.go          # MatchEventRepository impl
│   └── standing_repo.go       # StandingRepository impl (with recalculation)
├── services/
│   ├── league.go              # LeagueService, MatchEngine, WeatherAdapter interfaces
│   ├── league_impl.go         # Core league logic (play, edit, rollback, predict)
│   ├── match_engine.go        # Poisson-based match simulator (Adapter)
│   └── weather.go             # Weather condition generator (Adapter)
├── handlers/
│   └── league_handler.go      # REST API handlers
├── router/
│   └── router.go              # /api/v1 route definitions
└── README.md
```

## ⚙️ Tech Stack

| Component | Technology |
|-----------|-----------|
| Language | Go 1.26 |
| HTTP Router | gorilla/mux |
| Database | SQLite (go-sqlite3) |
| Architecture | Interface-based, Adapter pattern, DI |

## 🚀 Setup & Run

### Prerequisites
- Go 1.21+
- GCC (for CGO/SQLite)

### Quick Start
```bash
cd insider
go mod tidy
go build -o league-simulation.exe .
./league-simulation.exe
# → http://localhost:8080
```

### Environment Variables
| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | Server port |
| `DB_PATH` | `./league.db` | SQLite path |

## 📡 API Endpoints

| Method | Endpoint | Description |
| :--- | :--- | :--- |
| `GET` | `/api/v1/league/table` | Returns current standings (PTS, W, D, L, GD). |
| `POST` | `/api/v1/league/next-week` | Simulates the next week's matches and updates state. |
| `POST` | `/api/v1/league/play-all` | Simulates all remaining matches in the season. |
| `PUT` | `/api/v1/matches/{id}` | Edits a specific match result; recalculates standings and morale. |
| `GET` | `/api/v1/simulation/oracle` | Runs 1,000 Monte Carlo simulations to calculate Championship Win %. |
| `POST` | `/api/v1/league/rollback/{week}` | **Time Machine:** Reverts database state to a specific week. |
| `GET` | `/api/v1/teams/{id}/metrics` | Returns a team's current Strength, Morale, Fatigue, and Market Value. |
| `POST` | `/api/v1/league/reset` | Resets the entire league to initial state. |

## 🗄 SQL Schema

5 tables matching the schema requirements:

```sql
-- teams: id, name, market_value, base_strength, current_strength, morale, fatigue, city
-- players: id, team_id, name, position
-- matches: id, week, home_team_id, away_team_id, home_score, away_score, weather_condition, status
-- match_events: id, match_id, player_id, event_type, minute, detail
-- standings: team_id, played, won, drawn, lost, gf, ga, gd, points
```

## ⚽ Simulation Features

### Strength-Based Match Engine
- Poisson distribution for realistic goals
- Team strength (1-100) affects expected goals
- **Morale** boost/penalty (wins raise morale, losses lower it)
- **Fatigue** accumulates each match, reducing performance
- **Home advantage** (25% boost)

### Weather System
- Weather generated per match based on home city
- Conditions: ☀️ Sunny, 🌧️ Rainy, ❄️ Snowy, 💨 Windy, 🌫️ Foggy
- Manchester/Liverpool: more rain; London: more variety
- Weather affects goal expectations

### Match Events
- ⚽ Goals with random minute
- 🔴 Quantum VAR Decisions (5% chance per match)
- 🩹 Injuries (10% chance per match)

### Dynamic Team Metrics
- **Current Strength** = Base × Morale Factor × Fatigue Factor
- **Market Value** increases 2% per win
- Morale/fatigue reset on rollback

### 🔮 Oracle (Monte Carlo Predictions)
- Available after week 4
- Runs **1,000 simulations** of remaining matches
- Returns championship probability per team

### ⏪ Time Machine (Rollback)
- Revert to any week (1-6)
- Resets matches from that week onward to "scheduled"
- Deletes associated events
- Recalculates standings and team metrics

## 🏟️ Default Teams

| Team | Strength | City | Market Value (M€) |
|------|----------|------|--------------------|
| Manchester City | 90 | Manchester | 1200 |
| Arsenal | 85 | London | 1050 |
| Liverpool | 82 | Liverpool | 980 |
| Chelsea | 75 | London | 900 |

Each team has 8 real players (GK, DEF, MID, FWD) seeded into the database.

## 🧪 Testing with cURL / Postman

```bash
# 1. View standings
curl http://localhost:8080/api/v1/league/table

# 2. Play next week
curl -X POST http://localhost:8080/api/v1/league/next-week

# 3. View team metrics
curl http://localhost:8080/api/v1/teams/1/metrics

# 4. Play all remaining
curl -X POST http://localhost:8080/api/v1/league/play-all

# 5. Oracle predictions (after week 4)
curl http://localhost:8080/api/v1/simulation/oracle

# 6. Edit a match result
curl -X PUT http://localhost:8080/api/v1/matches/1 \
  -H "Content-Type: application/json" \
  -d '{"home_score": 3, "away_score": 0}'

# 7. Time Machine — rollback to week 3
curl -X POST http://localhost:8080/api/v1/league/rollback/3

# 8. Reset league
curl -X POST http://localhost:8080/api/v1/league/reset
```

## 📐 Design Decisions

1. **Interface-based design**: All repositories (`TeamRepository`, `MatchRepository`, `StandingRepository`, etc.) and services (`LeagueService`, `MatchEngine`, `WeatherAdapter`) are interfaces. This enables testability and swappability.

2. **Adapter Pattern**: `MatchEngine` and `WeatherAdapter` are implemented as adapters. The weather adapter can be swapped for a real API; the match engine can be replaced with different simulation strategies.

3. **Struct Composition**: `LeagueServiceImpl` composes multiple repository interfaces and adapter interfaces, following Go's idiomatic composition-over-inheritance approach.

4. **Transactional Rollback**: The rollback feature uses database transactions to safely revert state to a previous week, including recalculating standings and team metrics from the remaining played matches.

5. **SQLite with WAL**: Using Write-Ahead Logging for better concurrent read performance during Monte Carlo simulations.

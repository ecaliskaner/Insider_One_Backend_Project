# ⚽ Football League Simulation API v1

A GoLang REST API that simulates a 4-team football league with Premier League scoring rules, strength/morale/fatigue-based match simulation, weather effects, match events (including Quantum VAR Decisions), Monte Carlo championship predictions, and a Time Machine rollback feature.

## 🏗️ Architecture

Built using **strict interface-based design** and **struct composition**. External systems (Weather, Match Simulation) are implemented as **Adapters** so they can be swapped for real external APIs. Currently, they use internal generation to avoid rate limits, keeping the core simulation logic pure and testable.

```
insider/
├── .github/workflows/         # CI/CD GitHub Actions pipelines
├── main.go                    # Entry point, Graceful Shutdown, DI wiring
├── models/
│   ├── team.go                # Team struct & TeamRepository interface
│   ├── player.go              # Player struct & PlayerRepository interface
│   └── match.go               # Match, MatchEvent, Standing & repository interfaces
├── database/
│   ├── migrations/            # golang-migrate up/down SQL scripts
│   ├── db.go                  # DB connection & auto-migration via embed.FS
│   ├── seed.go                # Teams, players, standings, schedule seeding
│   ├── team_repo.go           # TeamRepository impl
│   ├── player_repo.go         # PlayerRepository impl
│   ├── match_repo.go          # MatchRepository impl (with rollback)
│   ├── event_repo.go          # MatchEventRepository impl
│   └── standing_repo.go       # StandingRepository impl (with recalculation)
├── services/
│   ├── event_bus.go           # Internal Pub/Sub channel-based bus
│   ├── listeners.go           # Asynchronous event consumers
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
| Language | Go 1.26+ |
| HTTP Router | gorilla/mux |
| Database | SQLite (go-sqlite3) |
| Architecture | Interface-based, Adapter pattern, Pub/Sub, DI |
| CI/CD | GitHub Actions |
| Migrations | golang-migrate |

Includes API middleware for structured JSON logging (slog), panic recovery, and rate limiting (golang.org/x/time/rate).

## 🐳 Docker Deployment (Recommended)
This project is fully containerized. Use the provided Docker Compose file to spin up the application and database volume.
```bash
make docker-run  # Wraps docker-compose up --build -d
```
Interactive OpenAPI 3.0 documentation is available at http://localhost:8080/swagger/index.html when running locally.

## 🧪 Testing & QA
- **Table-Driven Tests:** Core simulation logic is validated using Go's idiomatic table-driven test patterns.
- **Mocking:** `WeatherAdapter` and `MatchRepository` are mocked using `testify/mock` to isolate the `MatchEngine` during unit testing.
- Run tests via: `make test` or `go test ./... -v`

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

## 🗄 SQL Schema & Queries

5 tables matching the schema requirements:

```sql
-- teams: id, name, market_value, base_strength, current_strength, morale, fatigue, city
-- players: id, team_id, name, position
-- matches: id, week, home_team_id, away_team_id, home_score, away_score, weather_condition, status
-- match_events: id, match_id, player_id, event_type, minute, detail
-- standings: team_id, played, won, drawn, lost, gf, ga, gd, points
```

Complex SQL queries are located in `database/standing_repo.go` and `database/match_repo.go`. Here is an example of the Standings Upsert query which recalculates the dynamic league table:

```sql
INSERT INTO standings (team_id, played, won, drawn, lost, gf, ga, gd, points)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(team_id) DO UPDATE SET
    played=excluded.played, won=excluded.won, drawn=excluded.drawn,
    lost=excluded.lost, gf=excluded.gf, ga=excluded.ga, gd=excluded.gd,
    points=excluded.points
```

## ⚽ Simulation Features

### Strength-Based Match Engine
- Poisson distribution for realistic goals
- Team strength (1-100) affects expected goals
- **Morale** boost/penalty (wins raise morale, losses lower it)
- **Fatigue** accumulates each match, reducing performance
- **Home advantage** (25% boost)

### Weather System
- Weather generated per match based on home city (Implemented as an Adapter, easily swappable for OpenWeatherMap API)
- Conditions: ☀️ Sunny, 🌧️ Rainy, ❄️ Snowy, 💨 Windy, 🌫️ Foggy
- Manchester/Liverpool: more rain; London: more variety
- Weather affects goal expectations

### Match Events
- ⚽ Goals with random minute
- 🔴 Quantum VAR Decisions (5% chance per match, mock internal generator instead of external Quantum API to avoid rate limits)
- 🩹 Injuries (10% chance per match)

### Dynamic Team Metrics
- **Current Strength** = Base × Morale Factor × Fatigue Factor
- **Market Value** increases 2% per win
- Morale/fatigue reset on rollback

### 🔮 Oracle (Monte Carlo Predictions)
- Available after week 4
- Runs **1,000 simulations concurrently** utilizing Go worker pools and channels for millisecond-level parallel execution without race conditions.
- Returns championship probability per team

### ⏪ Time Machine (Rollback)
- Revert to any week (1-6)
- Resets matches from that week onward to "scheduled"
- Deletes associated events
- Recalculates standings and team metrics

## 🚀 Advanced Enterprise Features

- **Graceful Server Shutdown**: Intercepts `SIGTERM` and `SIGINT` signals, halts new incoming traffic, finishes active requests up to a 10-second context timeout, and safely closes the database connection to prevent WAL corruption.
- **Internal Event-Driven Architecture (Pub/Sub)**: `MatchEngine` is decoupled from persistence. It publishes `MatchFinishedEvent` to a channel-based `EventBus`. Independent background listeners consume these events to asynchronously recalculate standings and update morale/fatigue.
- **Oracle Response Caching**: The CPU-heavy `/oracle` Monte Carlo endpoint is guarded by a thread-safe `sync.RWMutex` in-memory cache, achieving 0ms latency on repeated calls. The cache is surgically invalidated upon any league state mutation (match played, rollback, reset).
- **Database Migrations (`golang-migrate`)**: Migrated away from raw startup scripts. Employs versioned `.up.sql` and `.down.sql` schemas, executed seamlessly on startup utilizing Go's `embed.FS` and `migrate/v4`.
- **Continuous Integration**: Powered by `.github/workflows/ci.yml`. On every push or PR, GitHub Actions automatically provisions Go, runs `go fmt`, `go vet`, executes table-driven unit tests with `testify/mock`, and validates the Docker build context.

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

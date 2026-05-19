Project: Insider Football League Simulation API v1
Objective: End-to-End (E2E), Concurrency, Data Integrity, and Infrastructure Validation.

This document outlines the testing strategy for the Football League Simulation API. It guarantees that the system not only meets the functional requirements of the Insider Case Study but also withstands production-level edge cases, race conditions, and state mutations.

🛠 Prerequisites
curl or Postman for API execution.

docker and docker-compose for infrastructure validation.

make for running Go test suites.

📍 Phase 1: Core Functional & Data Integrity Validation
Test 1.1: Verify Initial State & Seeding
Ensure migrations and seed data loaded correctly.

Bash
curl -s http://localhost:8080/api/v1/league/table
Validation Criteria: Returns a JSON array of 4 teams. played, won, drawn, lost, gf, ga, gd, and points must all strictly equal 0.

Test 1.2: Simulate Next Week
Bash
curl -X POST http://localhost:8080/api/v1/league/next-week
Validation Criteria: Returns HTTP 200 OK. The JSON response must contain the match results for Week 1. Verify that a weather_condition is present.

Test 1.3: Verify Premier League Sorting Logic
Bash
curl -s http://localhost:8080/api/v1/league/table
Validation Criteria: The JSON array must strictly sort by:

points (Descending)

gd (Goal Difference) (Descending)

gf (Goals For) (Descending)

Test 1.4: Match Event Data Integrity (Goal Assignment)
We must prove that goals aren't just arbitrary numbers, but are assigned to real players in the database.

Bash
curl -s http://localhost:8080/api/v1/matches/1
Validation Criteria: The JSON response must include an events array. If the match score was 2-1, there must be exactly 3 "Goal" events. The player_id for each goal must belong to the respective scoring team.

Test 1.5: Play Remaining Matches
Bash
curl -X POST http://localhost:8080/api/v1/league/play-all
Validation Criteria: Simulates all remaining weeks up to Week 6. Returns a nested JSON structure of all matches grouped by week.

📍 Phase 2: Advanced State Management & Caching
Test 2.1: Championship Probability Caching
Championship probabilities require Week 4 to be completed and utilizes an in-memory cache to prevent CPU spikes.

Trigger championship probabilities (Cache Miss - Cold Start):

Bash
time curl -s http://localhost:8080/api/v1/simulation/championship-probabilities
Validation Criteria: Takes slightly longer (e.g., 50-200ms) as goroutines run 1,000 simulations. Returns probability percentages that sum to ~100%.

Trigger championship probabilities Again (Cache Hit):

Bash
time curl -s http://localhost:8080/api/v1/simulation/championship-probabilities
Validation Criteria: Near 0ms execution time. The JSON payload must match the previous call exactly.

Test 2.2: Event-Driven Recalculation (Match Editing)
Verify that editing a match triggers the internal Pub/Sub bus, invalidates the cache, and recalculates the standings.

Edit a Week 1 Match:

Bash
curl -X PUT http://localhost:8080/api/v1/matches/1 \
  -H "Content-Type: application/json" \
  -d '{"home_score": 10, "away_score": 0}'
Verify State Mutation:

Table Check: curl -s http://localhost:8080/api/v1/league/table -> The home team's gd must immediately reflect the massive +10 shift.

Cache Invalidation: Call championship probabilities again. It must take slightly longer (Cache Miss) and return entirely new percentages, as the baseline state has changed.

Test 2.3: Transactional Rollback
Rollback to Week 3:

Bash
curl -X POST http://localhost:8080/api/v1/league/rollback/3
Verify State Reversal:

Table Check: All teams must reflect exactly 3 matches played.

Metrics Check: curl -s http://localhost:8080/api/v1/teams/1/metrics -> Fatigue and Morale metrics must be reset to their exact values at the end of Week 3. Future match events (Weeks 4-6) must be deleted from the database.

📍 Phase 3: Concurrency, Edge Cases & Error Handling
Test 3.1: Concurrent State Mutation (Race Condition Test)
We must ensure the database and application state do not corrupt if multiple users trigger state changes at the exact same millisecond.

Reset the League: curl -X POST http://localhost:8080/api/v1/league/reset

Fire 5 requests in parallel:

Bash
for i in {1..5}; do curl -s -o /dev/null -w "%{http_code}\n" -X POST http://localhost:8080/api/v1/league/next-week & done
Validation Criteria: Exactly one request should successfully simulate Week 1 (returning 200 OK). The others must either queue safely for subsequent weeks (Weeks 2, 3, 4, 5) OR return a safe 400 Bad Request depending on the locking strategy. The database must not contain duplicate Week 1 matches.

Test 3.2: Premature Championship Probability Request
Reset the league: curl -X POST http://localhost:8080/api/v1/league/reset

Call championship probabilities: curl -s http://localhost:8080/api/v1/simulation/championship-probabilities

Validation Criteria: Returns 400 Bad Request - Championship probabilities are only available after Week 4.

Test 3.3: Invalid Rollback Bound
Attempt out-of-bounds rollback: curl -X POST http://localhost:8080/api/v1/league/rollback/99

Validation Criteria: Returns 400 Bad Request - Invalid week.

📍 Phase 4: Infrastructure & DevOps Validation
Test 4.1: Docker Build & Network Success
Ensure the container compiles, starts, and successfully mounts the SQLite volume.

Bash
make docker-run
sleep 5
curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/api/v1/league/table
Validation Criteria: Returns 200. Terminal logs show successful database migrations on startup.

Test 4.2: Graceful Server Shutdown
Verify that the server does not instantly kill active database transactions when stopped.

Start the server locally (go run main.go).

Trigger the championship probabilities endpoint (curl http://localhost:8080/api/v1/simulation/championship-probabilities).

Immediately hit Ctrl+C in the terminal running the server.

Validation Criteria: 1. Terminal logs Received SIGINT, shutting down gracefully...
2. The curl command to championship probabilities must successfully complete and return the JSON payload.
3. Only after the response is sent, the terminal logs Database connection closed. Server stopped.

🤖 Appendix: Automated Unit Testing (AI Prompt)
To validate the Go interfaces automatically, run the unit test suite:

Bash
make test
# OR
go test ./... -v -cover
(For Developers) Use the following prompt in an AI assistant to generate the Table-Driven Tests for the LeagueService:

Prompt:
"I am building a GoLang REST API for a Football League Simulation. My architecture uses strict interface-based design. I need to write Table-Driven Unit Tests using testify/assert and testify/mock.

Please generate the _test.go file for my LeagueService.

Context:
The LeagueService struct relies on mocked interfaces: MatchRepository, StandingRepository, MatchEngine, and EventBus.

Write table-driven tests for the following scenarios:

TestCalculateStandings_Sorting: Manually mock a scenario where 3 teams have equal points. Mock the StandingRepository to return Team A (GD +5, GF 10), Team B (GD +5, GF 12), and Team C (GD +2, GF 15). Assert that the service sorts them correctly based on Premier League rules (Points -> GD -> GF). Therefore, Team B should be 1st, Team A 2nd, Team C 3rd.

TestChampionshipProbabilities_CacheHit: Mock the MatchRepository to return a state where 4 weeks are played. Call service.GetPredictions(). Assert that the MatchEngine.SimulateRemaining() is called exactly 1,000 times. Call it a second time and assert that MatchEngine.SimulateRemaining() is called 0 times (verifying the sync.RWMutex cache hit).

TestRollback_Transaction: Mock the MatchRepository.RollbackToWeek(3) to succeed. Assert that the service automatically triggers a cache invalidation and publishes a RecalculateStandings event to the EventBus.

Please output idiomatic, production-ready Go code.
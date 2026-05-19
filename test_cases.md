# Test Cases

This document describes the reviewer-facing test coverage for the Insider Football Simulation API. It combines automated tests, manual API checks, and edge cases tied to the original backend case requirements.

## Test Environment

| Item | Value |
| --- | --- |
| Language | Go 1.26.3 |
| Database | SQLite |
| Local base URL | `http://localhost:8080` |
| Deterministic seed | `SIM_SEED=42` |
| Test database | Temporary SQLite file |

## Verification Commands

```bash
gofmt -l .
go test ./...
go vet ./...
go build ./...
docker build -t insider-one-backend-project:review .
```

Manual API verification can be run against a local server:

```bash
DB_PATH=<temp-db> SIM_SEED=42 go run . migrate up
DB_PATH=<temp-db> SIM_SEED=42 go run . seed
DB_PATH=<temp-db> SIM_SEED=42 go run . serve
```

## Functional Test Cases

| ID | Scenario | Request | Expected Result | Coverage |
| --- | --- | --- | --- | --- |
| TC-001 | Reset league | `POST /api/v1/league/reset` | `200 OK`; teams, players, standings, and schedule are recreated | Core setup |
| TC-002 | Get standings | `GET /api/v1/league/table` | `200 OK`; returns four teams and current week metadata | League table |
| TC-003 | Get case overview before week 4 | `GET /api/v1/league/overview` | `200 OK`; returns `current_week`, `standings`, `weeks`; no `predictions` field | Case screen |
| TC-004 | Play next week | `POST /api/v1/league/next-week` | `200 OK`; next scheduled week is simulated and standings update | Week-by-week simulation |
| TC-005 | Play final week | six calls to `POST /api/v1/league/next-week` | Week 6 is playable; current week becomes 7 after season completion | Boundary correctness |
| TC-006 | Reject play after season completion | seventh call to `POST /api/v1/league/next-week` | `400 Bad Request`; title is `Season Overrun Prevented` | Season boundary |
| TC-007 | Play all remaining weeks | `POST /api/v1/league/play-all` | `200 OK`; all remaining matches are grouped by week | Extra requirement |
| TC-008 | Reject play all after completion | repeat `POST /api/v1/league/play-all` after completed season | `400 Bad Request`; detail says all weeks have already been played | Edge case |
| TC-009 | Championship probabilities before allowed week | `GET /api/v1/simulation/championship-probabilities` before week 5 | `400 Bad Request`; title is `Premature Championship Probability Request` | Prediction gating |
| TC-010 | Championship probabilities after week 4 | `GET /api/v1/simulation/championship-probabilities` after four weeks have been played | `200 OK`; returns four championship probabilities | Prediction requirement |
| TC-011 | Overview after week 4 | `GET /api/v1/league/overview` after four weeks have been played | `200 OK`; includes `predictions` with four entries | Case screen with estimation |
| TC-012 | Get match details | `GET /api/v1/matches/1` | `200 OK`; returns match and event list | Match status visibility |
| TC-013 | Edit match result | `PUT /api/v1/matches/1` with both scores | `200 OK`; match status is edited and standings rebuild | Extra requirement |
| TC-014 | Reject missing score field | `PUT /api/v1/matches/1` with only `home_score` | `400 Bad Request`; both scores are required | Request validation |
| TC-015 | Reject unknown edit field | `PUT /api/v1/matches/1` with `bonus_goal` | `400 Bad Request`; unknown fields are rejected | Strict JSON contract |
| TC-016 | Reject negative score | `PUT /api/v1/matches/1` with negative score | `400 Bad Request`; scores cannot be negative | Domain validation |
| TC-017 | Missing match | `GET /api/v1/matches/999` | `404 Not Found`; problem response returned | Not-found handling |
| TC-018 | Rollback valid week | `POST /api/v1/league/rollback/2` | `200 OK`; week 2 and future matches reset, standings rebuild | Rollback |
| TC-019 | Reject rollback out of range | `POST /api/v1/league/rollback/7` | `400 Bad Request`; valid range is week 0 through 6 | Boundary validation |
| TC-020 | Team metrics | `GET /api/v1/teams/1/metrics` | `200 OK`; returns strength, morale, fatigue, market value, city | Team state |
| TC-021 | Missing team | `GET /api/v1/teams/999/metrics` | `404 Not Found`; problem response returned | Not-found handling |
| TC-022 | Swagger UI | `GET /swagger/index.html` | `200 OK`; API docs are available | Documentation |
| TC-023 | Per-client rate limiter | exhaust one client bucket, then request from another client | first client receives `429`; second client still succeeds | Middleware isolation |
| TC-024 | Seeded simulation | run same simulation with same `SIM_SEED` | match engine and weather choices are repeatable | Demo reproducibility |
| TC-025 | Liveness probe | `GET /healthz` | `200 OK`; returns `status: ok` | Platform health |
| TC-026 | API-scoped health probe | `GET /api/v1/health` | `200 OK`; returns `status: ok` | API health |
| TC-027 | Readiness probe | `GET /readyz` | `200 OK` when database ping succeeds; `503` if storage is unavailable | Platform readiness |
| TC-028 | Preserve request ID | request with `X-Request-ID` | Response includes the same `X-Request-ID` value | Trace correlation |
| TC-029 | Generate request ID | request without `X-Request-ID` | Response includes a generated `X-Request-ID` value | Trace correlation |
| TC-030 | Malformed JSON edit body | `PUT /api/v1/matches/1` with invalid JSON | `400 Bad Request`; problem response explains malformed JSON | Request validation |
| TC-031 | Wrong edit content type | `PUT /api/v1/matches/1` with `text/plain` | `400 Bad Request`; content type is rejected | Request validation |
| TC-032 | Edit scheduled match | `PUT /api/v1/matches/1` before it has been played | `400 Bad Request`; only played matches can be edited | Domain validation |
| TC-033 | Duplicate rollback | repeat `POST /api/v1/league/rollback/2` | `200 OK`; rollback is idempotent and standings remain consistent | Rebuild consistency |
| TC-034 | Invalid team ID format | `GET /api/v1/teams/not-a-number/metrics` | `400 Bad Request`; team ID must be numeric | Request validation |

## Edge Cases Covered By Automated Tests

| Edge Case | Test |
| --- | --- |
| Week 6 is playable and week 7 is rejected | `TestPlayNextWeek_AllowsFinalWeekAndRejectsAfterSeason` |
| Overview hides predictions before week 4 | `TestLeagueOverview_ReturnsScreenPayloadWithoutPrematurePredictions` |
| Overview includes predictions after week 4 | `TestLeagueOverview_IncludesPredictionsAfterWeekFour` |
| Missing edit score is rejected | `TestEditMatch_RequiresBothScores` |
| Unknown edit field is rejected | `TestEditMatch_RejectsUnknownFields` |
| Negative edit score is rejected | `TestEditMatch_RejectsNegativeScores` |
| Malformed JSON edit body is rejected | `TestEditMatch_RejectsMalformedJSON` |
| Wrong edit content type is rejected | `TestEditMatch_RejectsWrongContentType` |
| Scheduled match edit is rejected | `TestEditMatch_RejectsScheduledMatch` |
| Premature championship probability request is rejected | `TestChampionshipProbabilities_RejectPrematureRequest` |
| Out-of-range rollback is rejected | `TestRollback_RejectsOutOfRangeWeek` |
| Duplicate rollback is idempotent and preserves standings rebuild consistency | `TestRollback_IsIdempotentAndPreservesRebuildConsistency` |
| Completed season rejects another play-all | `TestPlayAll_RejectsCompletedSeason` |
| Missing match returns 404 | `TestGetMatch_ReturnsNotFoundForMissingMatch` |
| Invalid team ID format is rejected | `TestTeamMetrics_RejectsInvalidTeamID` |
| Per-client rate limiting does not block other clients | `TestRateLimiterMiddleware_IsPerClient` |
| Health, API health, and readiness probes return success when dependencies are available | `TestHealthAndReadinessEndpoints` |
| Readiness returns 503 when the database is unavailable | `TestReadyz_ReturnsUnavailableWithoutDatabase` |
| Incoming request IDs are preserved | `TestRequestIDHeader_IsReturned`, `TestRequestIDMiddleware_PreservesInboundID` |
| Missing request IDs are generated | `TestRequestIDMiddleware_GeneratesMissingID` |
| Seeded simulation is deterministic | `TestSeededMatchEngine_IsDeterministic`, `TestSeededWeatherAdapter_IsDeterministic` |

## Latest Local Verification Result

Status: passed on local verification.

Executed checks:

```text
gofmt -l .        PASS
go test ./...     PASS
go vet ./...      PASS
go build ./...    PASS
```

Live API smoke test against a temporary SQLite database and `SIM_SEED=42`:

```text
TC-001 reset                         PASS
TC-002 standings                     PASS
TC-003 overview before predictions   PASS
TC-009 premature championship probabilities              PASS
TC-004 play weeks 1-4                PASS
TC-010 championship probabilities after week 4           PASS
TC-011 overview with predictions     PASS
TC-014 missing score                 PASS
TC-015 unknown field                 PASS
TC-016 negative score                PASS
TC-013 edit match                    PASS
TC-017 missing match                 PASS
TC-018 rollback                      PASS
TC-019 rollback out of range         PASS
TC-020 team metrics                  PASS
TC-021 missing team                  PASS
TC-022 swagger                       PASS
```

The live smoke test also caught and verified a reset-stability fix: `POST /api/v1/league/reset` now resets SQLite autoincrement counters so reviewer-friendly IDs such as match `1` and team `1` remain valid after reset.

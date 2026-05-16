# Insider Development Intern Hiring Day Task — BACKEND/Full Stack Case

## Task Description

Build a football league simulation in GoLang. The simulation contains a group of 4 teams and shows match results and the league table week by week. The goal is to estimate the final league table after week 4.

## League Rules
- 4 teams in the league (with different strengths affecting match outcomes)
- Scoring and rules follow the **Premier League** format:
  - Win: 3 points | Draw: 1 point | Loss: 0 points
  - Tiebreakers: Goal Difference → Goals Scored → Head-to-Head → H2H Away Goals

## Requirements
- [x] **GoLang only** (no Java, .NET, Ruby, etc.)
- [x] **Interface-based design** and **struct composition**
- [x] **No frontend required** — API endpoints testable via Postman
- [x] **SQL schema and queries** provided
- [x] **Documentation** (README.md)

## Extras (Strong Plus)
- [x] **Play All**: Auto-play all remaining weeks and list results
- [x] **Edit Match Results**: Modify scores and recalculate standings

## Implementation
- Language: Go 1.26
- Database: SQLite (embedded, zero-config)
- HTTP Router: gorilla/mux
- Match Simulation: Poisson distribution with strength-based probabilities
- Predictions: Monte Carlo simulation (10,000 iterations)

## Endpoints
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/league/table` | Current league standings |
| GET | `/api/league/schedule` | Full match schedule |
| GET | `/api/league/week/{week}` | Results for a specific week |
| POST | `/api/league/play-next` | Simulate next week |
| POST | `/api/league/play-all` | Play all remaining weeks |
| PUT | `/api/league/match/{id}` | Edit a match result |
| GET | `/api/league/predictions` | Championship predictions (after week 4) |
| POST | `/api/league/reset` | Reset the league |
# Project Specification: Insider Football League Simulator (GoLang)

## 📌 1. Project Overview
A backend-driven football simulation engine that models a 4-team league. The simulation calculates match results, maintains a live standings table, and generates championship winning probabilities starting from Week 4. It leverages external APIs to ground the simulation in real-world data and true randomness.

## 🛠 2. Tech Stack & Architectural Constraints
* **Language:** GoLang strictly (No Java, .NET, Ruby, etc.).
* **Design Pattern:** Interface-based design and struct composition. Use the Adapter/Decorator patterns for all external API calls.
* **Database:** SQL (Schema and queries required).
* **Interface:** RESTful API endpoints (Postman-ready).
* **Frontend:** Not required.

## ⚽ 3. Core Simulation Logic & State Metrics
* **League Rules:** Premier League scoring (3 for a win, 1 for a draw, 0 for a loss) and Goal Difference (GD) tie-breakers.
* **Core Team Metrics:**
    * `Base Strength`: Set at the start of the season.
    * `Current Strength`: Fluctuates week-to-week.
    * `Morale`: A dynamic metric (0-100) that multiplies effective strength. Drops after losses, rises after wins or positive external sentiment.
    * `Fatigue`: Drops progressively as matches are played, penalizing effective strength.
* **Home Team Bonus:** The match engine must apply a flat $+10\%$ effective strength multiplier to the home team to simulate stadium atmosphere and lack of travel fatigue.
* **Match Simulation Flow:** `Play All` and `Next Week` endpoints must calculate results factoring in all metrics, update standings, and log events. 

## 🧠 4. Advanced Mechanics & External API Integrations
The simulation must use the following external integrations to act as data providers and external state modifiers:

### A. Market Value Initialization (API-Football / Transfermarkt)
* **Purpose:** Ground the initial simulation state in real-world financial realities.
* **Implementation:** On initialization, fetch the current total squad market value for the 4 chosen teams. Map this financial value directly to the team's `Base Strength` (e.g., a £1B squad starts with a significantly higher base score than a £500M squad).

### B. Real-World Sentiment Analysis (X/Twitter API + LLM/Sentiment Tool)
* **Purpose:** Tie team `Morale` to real-world fan reactions.
* **Implementation:** Before a simulated week, fetch recent social media mentions of the teams. If the sentiment is overwhelmingly positive (e.g., following a real-life transfer or win), apply a "Fan Hype" boost, increasing that team's `Morale` and `Current Strength` by $+5\%$.

### C. The "Real-World Referee" (ANU Quantum RNG / Random.org API)
* **Purpose:** Introduce true, physical randomness for game-changing events.
* **Implementation:** Do not rely solely on Go's pseudo-random `math/rand`. When the simulation logic rolls for a controversial "VAR Decision", "Penalty", or "Red Card", call a Quantum Random Number API to decide the outcome. 

### D. Dynamic Weather Impact (OpenWeatherMap API)
* **Purpose:** Introduce environmental chaos.
* **Implementation:** Fetch real-time weather for the home team's city. Rain/Snow increases the "Chaos Factor" of the match, neutralizing strength gaps and giving underdogs a higher chance to upset.

### E. Real Player Data & Injury Sync (API-Football / News Scraper)
* **Purpose:** Use real names and reflect real-world availability.
* **Implementation:** Fetch actual squad lists. Randomly select these players as goalscorers. If a real-world star is reported injured, apply a temporary `Strength` penalty to the team.

### F. The "Newsroom" Generator (LLM API - Gemini or OpenAI)
* **Purpose:** Generate narrative summaries of match results.
* **Implementation:** Send match data to an LLM prompted to act as a sarcastic sports journalist, returning colorful commentary in the API response.

## 🗺 5. REST API Endpoints

| Method | Endpoint | Description |
| :--- | :--- | :--- |
| `GET` | `/api/v1/league/table` | Returns current standings (PTS, W, D, L, GD). |
| `POST` | `/api/v1/league/next-week` | Simulates the next week's matches and updates state. |
| `POST` | `/api/v1/league/play-all` | Simulates all remaining matches in the season. |
| `PUT` | `/api/v1/matches/{id}` | Edits a specific match result; recalculates standings and morale. |
| `GET` | `/api/v1/simulation/oracle` | Runs 1,000 background Monte Carlo simulations to calculate Championship Win %. |
| `POST` | `/api/v1/league/rollback/{week}` | **Time Machine:** Reverts database state to a specific week. |
| `GET` | `/api/v1/teams/{id}/metrics` | Returns a team's current `Strength`, `Morale`, `Fatigue`, and `Market Value`. |

## 🗄 6. SQL Schema Requirements
* **`teams`**: `id`, `name`, `market_value`, `base_strength`, `current_strength`, `morale`, `fatigue`, `city`.
* **`players`**: `id`, `team_id`, `name`, `position`.
* **`matches`**: `id`, `week`, `home_team_id`, `away_team_id`, `home_score`, `away_score`, `weather_condition`, `status`.
* **`match_events`**: `id`, `match_id`, `player_id`, `event_type` (e.g., Goal, Quantum VAR Decision).
* **`standings`**: `team_id`, `played`, `won`, `drawn`, `lost`, `gf`, `ga`, `gd`, `points`.

## 🤖 7. AI Agent Instructions
> "Act as a Senior GoLang Backend Engineer. Build this football simulation using strict interface-based design and struct composition. Start by defining the domain models (`Team`, `Match`, `Standing`) and the core `MatchEngine` interface. Implement the external APIs (Market Value, Sentiment, Quantum RNG, Weather, Injuries, LLM) as Decorators or Adapters so the core simulation logic remains pure and testable. Ensure all database interactions use a SQL driver and support transactions for the rollback/edit features."
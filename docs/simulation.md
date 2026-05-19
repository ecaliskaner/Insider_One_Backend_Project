# Championship Probability Simulation

This document explains how the prediction endpoint calculates championship probabilities:

```text
GET /api/v1/simulation/championship-probabilities
```

## Availability

Predictions are intentionally unavailable until four weeks have been played. Before that point the league table is too sparse, so the endpoint returns `400 Bad Request` with a problem+json response.

## Inputs

Each simulation starts from the current persisted league state:

- all teams with their current strength, morale, fatigue, market value, and city;
- all played or edited matches with fixed scores;
- all scheduled matches that still need simulated scores.

Played and edited matches are treated as facts. Only scheduled matches are simulated.

## Simulation Loop

The service runs 1,000 Monte Carlo simulations. In each simulation:

1. Copy the current match list in memory.
2. For every scheduled match, simulate a home and away score with the match engine.
3. Recalculate the final league table from the copied match list.
4. Rank the table with the same shared ranking function used by the real standings endpoint.
5. Count the first-place team as the winner for that simulation.

After all runs, each team's probability is:

```text
team wins / 1000 * 100
```

The result is rounded to two decimals.

## Ranking Consistency

Both real standings and simulated standings call `models.RankStandings`. This prevents the API from showing one ranking rule in `/league/table` and a different ranking rule inside the prediction engine.

The ranking order is:

1. points;
2. goal difference;
3. goals for;
4. head-to-head points for teams still perfectly tied;
5. head-to-head away goals;
6. team name as a deterministic final fallback.

## Caching

Prediction generation is CPU-heavy compared with normal API reads, so the result is cached in memory.

The cache is invalidated whenever league state changes:

- next week is played;
- all remaining weeks are played;
- a match result is edited;
- rollback is performed;
- league reset is performed.

Cached reads return the same response without rerunning the 1,000 simulations.

Cache invalidation also publishes an internal `prediction_cache_invalidated` domain event so logs can show why a cached probability result was discarded.

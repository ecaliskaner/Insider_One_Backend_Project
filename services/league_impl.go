package services

import (
	"context"
	"fmt"
	"math"
	"runtime"
	"sort"
	"sync"

	"github.com/ecaliskaner/Insider_One_Backend_Project/database"
	"github.com/ecaliskaner/Insider_One_Backend_Project/models"
)

const totalWeeks = 6

// LeagueServiceImpl implements LeagueService using struct composition.
// Composes repository interfaces and adapter interfaces.
type LeagueServiceImpl struct {
	teamRepo     models.TeamRepository
	playerRepo   models.PlayerRepository
	matchRepo    models.MatchRepository
	eventRepo    models.MatchEventRepository
	standingRepo models.StandingRepository
	engine       MatchEngine
	weather      WeatherAdapter
	db           *database.DB

	// Advanced Architecture Features
	eventBus        *EventBus
	cacheMu         sync.RWMutex
	predictionCache []models.Prediction
	stateMu         sync.RWMutex
}

// NewLeagueService creates a new LeagueServiceImpl with dependency injection
func NewLeagueService(db *database.DB, engine MatchEngine, weather WeatherAdapter) *LeagueServiceImpl {
	eb := NewEventBus()
	svc := &LeagueServiceImpl{
		teamRepo:     database.NewTeamRepo(db.Conn),
		playerRepo:   database.NewPlayerRepo(db.Conn),
		matchRepo:    database.NewMatchRepo(db.Conn),
		eventRepo:    database.NewEventRepo(db.Conn),
		standingRepo: database.NewStandingRepo(db.Conn),
		engine:       engine,
		weather:      weather,
		db:           db,
		eventBus:     eb,
	}

	StartListeners(eb, svc)
	return svc
}

func (s *LeagueServiceImpl) invalidateCache() {
	s.cacheMu.Lock()
	s.predictionCache = nil
	s.cacheMu.Unlock()
}

func (s *LeagueServiceImpl) runInTx(ctx context.Context, fn func(*LeagueServiceImpl, database.DBTX) error) error {
	tx, err := s.db.Conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	txSvc := &LeagueServiceImpl{
		teamRepo:     database.NewTeamRepo(tx),
		playerRepo:   database.NewPlayerRepo(tx),
		matchRepo:    database.NewMatchRepo(tx),
		eventRepo:    database.NewEventRepo(tx),
		standingRepo: database.NewStandingRepo(tx),
		engine:       s.engine,
		weather:      s.weather,
		db:           s.db,
		eventBus:     s.eventBus,
	}

	if err := fn(txSvc, tx); err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return fmt.Errorf("%w; rollback failed: %v", err, rollbackErr)
		}
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

// GetStandings returns the current league table
func (s *LeagueServiceImpl) GetStandings(ctx context.Context) ([]models.Standing, error) {
	s.stateMu.RLock()
	defer s.stateMu.RUnlock()
	return s.standingRepo.GetAll(ctx)
}

// GetOverview returns the full case screen payload in one request.
func (s *LeagueServiceImpl) GetOverview(ctx context.Context) (*models.LeagueOverview, error) {
	s.stateMu.RLock()
	currentWeek, err := s.matchRepo.GetCurrentWeek(ctx)
	if err != nil {
		s.stateMu.RUnlock()
		return nil, fmt.Errorf("failed to get current week: %w", err)
	}
	standings, err := s.standingRepo.GetAll(ctx)
	if err != nil {
		s.stateMu.RUnlock()
		return nil, fmt.Errorf("failed to get standings: %w", err)
	}
	matches, err := s.matchRepo.GetAll(ctx)
	if err != nil {
		s.stateMu.RUnlock()
		return nil, fmt.Errorf("failed to get matches: %w", err)
	}
	s.stateMu.RUnlock()

	overview := &models.LeagueOverview{
		CurrentWeek: currentWeek,
		Standings:   standings,
		Weeks:       groupMatchesByWeek(matches),
	}

	if currentWeek > 4 {
		predictions, err := s.GetPredictions(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get predictions: %w", err)
		}
		overview.Predictions = predictions
	}

	return overview, nil
}

// PlayNextWeek simulates the next week and updates all state
func (s *LeagueServiceImpl) PlayNextWeek(ctx context.Context) (*models.WeekResult, error) {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	return s.playNextWeekLocked(ctx)
}

func (s *LeagueServiceImpl) playNextWeekLocked(ctx context.Context) (*models.WeekResult, error) {
	var result *models.WeekResult
	if err := s.runInTx(ctx, func(txSvc *LeagueServiceImpl, _ database.DBTX) error {
		weekResult, err := txSvc.playNextWeekInStore(ctx)
		if err != nil {
			return err
		}
		result = weekResult
		return nil
	}); err != nil {
		return nil, err
	}

	s.invalidateCache()
	s.eventBus.Publish("week_finished", result.Week)
	return result, nil
}

func (s *LeagueServiceImpl) playNextWeekInStore(ctx context.Context) (*models.WeekResult, error) {
	currentWeek, err := s.matchRepo.GetCurrentWeek(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current week: %w", err)
	}
	if currentWeek > totalWeeks {
		return nil, fmt.Errorf("all %d weeks have been played", totalWeeks)
	}

	matches, err := s.matchRepo.GetByWeek(ctx, currentWeek)
	if err != nil {
		return nil, err
	}

	teams, err := s.teamRepo.GetAll(ctx)
	if err != nil {
		return nil, err
	}
	teamMap := make(map[int]models.Team)
	for _, t := range teams {
		teamMap[t.ID] = t
	}

	for i := range matches {
		if matches[i].Status != "scheduled" {
			continue
		}

		homeTeam := teamMap[matches[i].HomeTeamID]
		awayTeam := teamMap[matches[i].AwayTeamID]

		// Get weather for match
		weather := s.weather.GetWeather(homeTeam.City)
		matches[i].WeatherCondition = weather

		// Simulate
		homeGoals, awayGoals, events := s.engine.SimulateMatch(homeTeam, awayTeam, weather)
		matches[i].HomeScore = &homeGoals
		matches[i].AwayScore = &awayGoals
		matches[i].Status = "played"

		// Save match
		if err := s.matchRepo.Update(ctx, &matches[i]); err != nil {
			return nil, err
		}

		// Save events
		for _, ev := range events {
			ev.MatchID = matches[i].ID
			if err := s.eventRepo.Create(ctx, &ev); err != nil {
				return nil, fmt.Errorf("create match event: %w", err)
			}
		}
	}

	// Recalculate all standings and team metrics deterministically from DB
	if err := s.rebuildState(ctx); err != nil {
		return nil, fmt.Errorf("failed to rebuild state: %w", err)
	}

	// Re-fetch matches with team names
	matches, err = s.matchRepo.GetByWeek(ctx, currentWeek)
	if err != nil {
		return nil, fmt.Errorf("failed to load week result: %w", err)
	}

	return &models.WeekResult{
		Week:    currentWeek,
		Matches: matches,
	}, nil
}

// PlayAll simulates all remaining weeks
func (s *LeagueServiceImpl) PlayAll(ctx context.Context) ([]models.WeekResult, error) {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()

	var results []models.WeekResult
	for {
		currentWeek, err := s.matchRepo.GetCurrentWeek(ctx)
		if err != nil {
			return nil, err
		}
		if currentWeek > totalWeeks {
			break
		}
		result, err := s.playNextWeekLocked(ctx)
		if err != nil {
			return nil, err
		}
		results = append(results, *result)
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("all weeks have already been played")
	}
	return results, nil
}

// GetMatch retrieves a match and its events
func (s *LeagueServiceImpl) GetMatch(ctx context.Context, matchID int) (*models.Match, []models.MatchEvent, error) {
	s.stateMu.RLock()
	defer s.stateMu.RUnlock()

	match, err := s.matchRepo.GetByID(ctx, matchID)
	if err != nil {
		return nil, nil, err
	}
	events, err := s.eventRepo.GetByMatchID(ctx, matchID)
	if err != nil {
		return nil, nil, err
	}
	return match, events, nil
}

// EditMatch updates a match result and recalculates standings and morale
func (s *LeagueServiceImpl) EditMatch(ctx context.Context, matchID int, homeScore int, awayScore int) (*models.Match, error) {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()

	var updatedMatch *models.Match
	if err := s.runInTx(ctx, func(txSvc *LeagueServiceImpl, _ database.DBTX) error {
		match, err := txSvc.matchRepo.GetByID(ctx, matchID)
		if err != nil {
			return err
		}
		if homeScore < 0 || awayScore < 0 {
			return fmt.Errorf("scores cannot be negative")
		}

		match.HomeScore = &homeScore
		match.AwayScore = &awayScore
		match.Status = "edited"

		if err := txSvc.matchRepo.Update(ctx, match); err != nil {
			return err
		}

		if err := txSvc.eventRepo.DeleteByMatchID(ctx, matchID); err != nil {
			return fmt.Errorf("delete match events: %w", err)
		}

		if err := txSvc.rebuildState(ctx); err != nil {
			return fmt.Errorf("failed to rebuild state: %w", err)
		}

		updatedMatch, err = txSvc.matchRepo.GetByID(ctx, matchID)
		if err != nil {
			return fmt.Errorf("reload edited match: %w", err)
		}
		return nil
	}); err != nil {
		return nil, err
	}

	s.invalidateCache()
	return updatedMatch, nil
}

// GetPredictions runs 1,000 Monte Carlo simulations for championship win probabilities.
func (s *LeagueServiceImpl) GetPredictions(ctx context.Context) ([]models.Prediction, error) {
	s.cacheMu.RLock()
	if s.predictionCache != nil {
		s.cacheMu.RUnlock()
		return s.predictionCache, nil
	}
	s.cacheMu.RUnlock()

	s.stateMu.RLock()
	currentWeek, err := s.matchRepo.GetCurrentWeek(ctx)
	if err != nil {
		s.stateMu.RUnlock()
		return nil, err
	}
	if currentWeek <= 4 {
		s.stateMu.RUnlock()
		return nil, fmt.Errorf("predictions available after week 4 (current: week %d)", currentWeek)
	}

	teams, err := s.teamRepo.GetAll(ctx)
	if err != nil {
		s.stateMu.RUnlock()
		return nil, err
	}
	allMatches, err := s.matchRepo.GetAll(ctx)
	if err != nil {
		s.stateMu.RUnlock()
		return nil, err
	}
	s.stateMu.RUnlock()

	teamMap := make(map[int]models.Team)
	for _, t := range teams {
		teamMap[t.ID] = t
	}

	const simulations = 1000
	winCount := make(map[int]int)

	numWorkers := runtime.NumCPU()
	if numWorkers < 1 {
		numWorkers = 1
	}

	jobs := make(chan int, simulations)
	results := make(chan int, simulations)

	var wg sync.WaitGroup

	// Start worker pool
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range jobs {
				simMatches := make([]models.Match, len(allMatches))
				copy(simMatches, allMatches)

				for i := range simMatches {
					if simMatches[i].Status == "scheduled" {
						ht := teamMap[simMatches[i].HomeTeamID]
						at := teamMap[simMatches[i].AwayTeamID]
						hg, ag, _ := s.engine.SimulateMatch(ht, at, "sunny")
						simMatches[i].HomeScore = &hg
						simMatches[i].AwayScore = &ag
						simMatches[i].Status = "played"
					}
				}

				standings := s.calcStandingsFromMatches(teams, simMatches)
				if len(standings) > 0 {
					results <- standings[0].TeamID
				}
			}
		}()
	}

	// Send jobs
	for sim := 0; sim < simulations; sim++ {
		jobs <- sim
	}
	close(jobs)

	// Wait for workers to finish and close results channel
	go func() {
		wg.Wait()
		close(results)
	}()

	// Aggregate results without mutex bottlenecks
	for winnerID := range results {
		winCount[winnerID]++
	}

	var predictions []models.Prediction
	for _, t := range teams {
		pct := float64(winCount[t.ID]) / float64(simulations) * 100
		predictions = append(predictions, models.Prediction{
			TeamID:   t.ID,
			TeamName: t.Name,
			WinPct:   math.Round(pct*100) / 100,
		})
	}
	sort.Slice(predictions, func(i, j int) bool {
		return predictions[i].WinPct > predictions[j].WinPct
	})

	// Save to cache
	s.cacheMu.Lock()
	s.predictionCache = predictions
	s.cacheMu.Unlock()

	return predictions, nil
}

// Rollback reverts the league state to a specific week.
func (s *LeagueServiceImpl) Rollback(ctx context.Context, week int) error {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()

	if week < 0 || week > totalWeeks {
		return fmt.Errorf("invalid week %d, must be between 0 and %d", week, totalWeeks)
	}

	if err := s.runInTx(ctx, func(txSvc *LeagueServiceImpl, _ database.DBTX) error {
		if err := txSvc.eventRepo.DeleteFromWeek(ctx, week); err != nil {
			return fmt.Errorf("failed to rollback events: %w", err)
		}

		if err := txSvc.matchRepo.DeleteFromWeek(ctx, week); err != nil {
			return fmt.Errorf("failed to rollback matches: %w", err)
		}

		if err := txSvc.rebuildState(ctx); err != nil {
			return fmt.Errorf("failed to rebuild state: %w", err)
		}
		return nil
	}); err != nil {
		return err
	}

	s.invalidateCache()
	return nil
}

// GetTeamMetrics returns a team's current metrics
func (s *LeagueServiceImpl) GetTeamMetrics(ctx context.Context, teamID int) (*models.TeamMetrics, error) {
	s.stateMu.RLock()
	defer s.stateMu.RUnlock()

	team, err := s.teamRepo.GetByID(ctx, teamID)
	if err != nil {
		return nil, err
	}
	return &models.TeamMetrics{
		TeamID:       team.ID,
		TeamName:     team.Name,
		Strength:     team.CurrentStrength,
		BaseStrength: team.BaseStrength,
		Morale:       math.Round(team.Morale*1000) / 1000,
		Fatigue:      math.Round(team.Fatigue*1000) / 1000,
		MarketValue:  team.MarketValue,
		City:         team.City,
	}, nil
}

// Reset clears everything and regenerates
func (s *LeagueServiceImpl) Reset(ctx context.Context) error {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()

	if err := s.runInTx(ctx, func(txSvc *LeagueServiceImpl, txStore database.DBTX) error {
		if err := txSvc.eventRepo.DeleteAll(ctx); err != nil {
			return fmt.Errorf("delete events: %w", err)
		}
		if err := txSvc.matchRepo.DeleteAll(ctx); err != nil {
			return fmt.Errorf("delete matches: %w", err)
		}
		if err := txSvc.standingRepo.DeleteAll(ctx); err != nil {
			return fmt.Errorf("delete standings: %w", err)
		}
		if err := txSvc.playerRepo.DeleteAll(ctx); err != nil {
			return fmt.Errorf("delete players: %w", err)
		}
		if err := txSvc.teamRepo.DeleteAll(ctx); err != nil {
			return fmt.Errorf("delete teams: %w", err)
		}
		if err := database.ResetAutoIncrement(ctx, txStore); err != nil {
			return fmt.Errorf("reset autoincrement counters: %w", err)
		}

		if err := database.SeedTeamsContext(ctx, txStore); err != nil {
			return err
		}
		if err := database.SeedPlayersContext(ctx, txStore); err != nil {
			return err
		}
		if err := database.SeedStandingsContext(ctx, txStore); err != nil {
			return err
		}
		if err := database.GenerateScheduleContext(ctx, txStore); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}

	// Re-init repos to use fresh data
	s.teamRepo = database.NewTeamRepo(s.db.Conn)
	s.playerRepo = database.NewPlayerRepo(s.db.Conn)
	s.matchRepo = database.NewMatchRepo(s.db.Conn)
	s.eventRepo = database.NewEventRepo(s.db.Conn)
	s.standingRepo = database.NewStandingRepo(s.db.Conn)

	s.invalidateCache()

	return nil
}

// GetCurrentWeek returns the next unplayed week
func (s *LeagueServiceImpl) GetCurrentWeek(ctx context.Context) (int, error) {
	s.stateMu.RLock()
	defer s.stateMu.RUnlock()
	return s.matchRepo.GetCurrentWeek(ctx)
}

// applyMetricChanges modifies team metrics in-memory
func (s *LeagueServiceImpl) applyMetricChanges(home, away *models.Team, homeGoals, awayGoals int) {
	if homeGoals > awayGoals {
		home.Morale = clamp(home.Morale+0.08, 0, 1)
		away.Morale = clamp(away.Morale-0.06, 0, 1)
		// Market value boost for winner
		home.MarketValue *= 1.02
	} else if awayGoals > homeGoals {
		away.Morale = clamp(away.Morale+0.08, 0, 1)
		home.Morale = clamp(home.Morale-0.06, 0, 1)
		away.MarketValue *= 1.02
	} else {
		home.Morale = clamp(home.Morale+0.02, 0, 1)
		away.Morale = clamp(away.Morale+0.02, 0, 1)
	}

	// Fatigue increases each match
	home.Fatigue = clamp(home.Fatigue+0.075, 0, 1)
	away.Fatigue = clamp(away.Fatigue+0.075, 0, 1)

	// Recalculate current strength from base + morale - fatigue
	home.CurrentStrength = int(float64(home.BaseStrength) * (0.8 + home.Morale*0.3) * (1.0 - home.Fatigue*0.15))
	away.CurrentStrength = int(float64(away.BaseStrength) * (0.8 + away.Morale*0.3) * (1.0 - away.Fatigue*0.15))
}

// calcStandingsFromMatches is a pure function for Monte Carlo simulations
func (s *LeagueServiceImpl) calcStandingsFromMatches(teams []models.Team, matches []models.Match) []models.Standing {
	standingMap := make(map[int]*models.Standing)
	for _, t := range teams {
		standingMap[t.ID] = &models.Standing{TeamID: t.ID, TeamName: t.Name}
	}

	for _, m := range matches {
		if m.HomeScore == nil || m.AwayScore == nil {
			continue
		}
		if m.Status != "played" && m.Status != "edited" {
			continue
		}

		home := standingMap[m.HomeTeamID]
		away := standingMap[m.AwayTeamID]
		hg, ag := *m.HomeScore, *m.AwayScore

		home.Played++
		away.Played++
		home.GF += hg
		home.GA += ag
		away.GF += ag
		away.GA += hg

		if hg > ag {
			home.Won++
			home.Points += 3
			away.Lost++
		} else if hg < ag {
			away.Won++
			away.Points += 3
			home.Lost++
		} else {
			home.Drawn++
			away.Drawn++
			home.Points++
			away.Points++
		}
	}

	var standings []models.Standing
	for _, st := range standingMap {
		st.GD = st.GF - st.GA
		standings = append(standings, *st)
	}

	return models.RankStandings(standings, matches)
}

func clamp(val, min, max float64) float64 {
	if val < min {
		return min
	}
	if val > max {
		return max
	}
	return val
}

func groupMatchesByWeek(matches []models.Match) []models.WeekResult {
	weekMap := make(map[int][]models.Match)
	var weekNumbers []int
	for _, match := range matches {
		if _, ok := weekMap[match.Week]; !ok {
			weekNumbers = append(weekNumbers, match.Week)
		}
		weekMap[match.Week] = append(weekMap[match.Week], match)
	}

	sort.Ints(weekNumbers)
	weeks := make([]models.WeekResult, 0, len(weekNumbers))
	for _, week := range weekNumbers {
		weeks = append(weeks, models.WeekResult{
			Week:    week,
			Matches: weekMap[week],
		})
	}
	return weeks
}

// rebuildState deterministically rebuilds all team metrics and standings from the ground up based on played matches
func (s *LeagueServiceImpl) rebuildState(ctx context.Context) error {
	// 1. Reset team morale/fatigue/strength to defaults
	teams, err := s.teamRepo.GetAll(ctx)
	if err != nil {
		return err
	}
	for _, t := range teams {
		t.CurrentStrength = t.BaseStrength
		t.Morale = 0.5 // Default starting morale is 0.5
		t.Fatigue = 0.0
		// We don't reset MarketValue as it's not strictly deterministic right now without knowing the initial,
		// but since it's just a cosmetic multiplier we can let it be, or reset it if we wanted.
		if err := s.teamRepo.Update(ctx, &t); err != nil {
			return fmt.Errorf("reset team metrics: %w", err)
		}
	}

	// 2. Replay morale/fatigue changes from played/edited matches in chronological order
	allMatches, err := s.matchRepo.GetAll(ctx)
	if err != nil {
		return err
	}

	teamMap := make(map[int]*models.Team)
	for i := range teams {
		teamMap[teams[i].ID] = &teams[i]
	}

	for _, m := range allMatches {
		if m.Status == "played" || m.Status == "edited" {
			if m.HomeScore != nil && m.AwayScore != nil {
				ht := teamMap[m.HomeTeamID]
				at := teamMap[m.AwayTeamID]
				if ht != nil && at != nil {
					s.applyMetricChanges(ht, at, *m.HomeScore, *m.AwayScore)
				}
			}
		}
	}
	for _, t := range teamMap {
		if err := s.teamRepo.Update(ctx, t); err != nil {
			return fmt.Errorf("persist team metrics: %w", err)
		}
	}

	// 3. Recalculate standings completely
	if err := s.standingRepo.RecalculateAll(ctx, allMatches); err != nil {
		return err
	}

	return nil
}

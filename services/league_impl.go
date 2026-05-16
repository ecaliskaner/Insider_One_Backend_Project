package services

import (
	"fmt"
	"math"
	"math/rand"
	"runtime"
	"sort"
	"sync"

	"github.com/insider/league-simulation/database"
	"github.com/insider/league-simulation/models"
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
	eventBus    *EventBus
	cacheMu     sync.RWMutex
	oracleCache []models.Prediction
	stateMu     sync.Mutex
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
	s.oracleCache = nil
	s.cacheMu.Unlock()
}

// GetStandings returns the current league table
func (s *LeagueServiceImpl) GetStandings() ([]models.Standing, error) {
	return s.standingRepo.GetAll()
}

// PlayNextWeek simulates the next week and updates all state
func (s *LeagueServiceImpl) PlayNextWeek() (*models.WeekResult, error) {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()

	currentWeek, err := s.matchRepo.GetCurrentWeek()
	if err != nil {
		return nil, fmt.Errorf("failed to get current week: %w", err)
	}
	if currentWeek > totalWeeks {
		return nil, fmt.Errorf("all %d weeks have been played", totalWeeks)
	}

	matches, err := s.matchRepo.GetByWeek(currentWeek)
	if err != nil {
		return nil, err
	}

	teams, err := s.teamRepo.GetAll()
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
		if err := s.matchRepo.Update(&matches[i]); err != nil {
			return nil, err
		}

		// Save events
		for _, ev := range events {
			ev.MatchID = matches[i].ID
			s.eventRepo.Create(&ev)
		}

		// Publish MatchFinishedEvent to EventBus
		s.eventBus.Publish("match_finished", MatchFinishedEvent{
			HomeTeam:  homeTeam,
			AwayTeam:  awayTeam,
			HomeGoals: homeGoals,
			AwayGoals: awayGoals,
			MatchID:   matches[i].ID,
		})
	}

	// Invalidate Oracle Cache
	s.invalidateCache()

	// Recalculate standings from all played matches
	allMatches, _ := s.matchRepo.GetAll()
	s.standingRepo.RecalculateAll(allMatches)

	// Re-fetch matches with team names
	matches, _ = s.matchRepo.GetByWeek(currentWeek)

	return &models.WeekResult{
		Week:    currentWeek,
		Matches: matches,
	}, nil
}

// PlayAll simulates all remaining weeks
func (s *LeagueServiceImpl) PlayAll() ([]models.WeekResult, error) {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()

	var results []models.WeekResult
	for {
		currentWeek, err := s.matchRepo.GetCurrentWeek()
		if err != nil {
			return nil, err
		}
		if currentWeek > totalWeeks {
			break
		}
		result, err := s.PlayNextWeek()
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
func (s *LeagueServiceImpl) GetMatch(matchID int) (*models.Match, []models.MatchEvent, error) {
	match, err := s.matchRepo.GetByID(matchID)
	if err != nil {
		return nil, nil, err
	}
	events, err := s.eventRepo.GetByMatchID(matchID)
	if err != nil {
		return nil, nil, err
	}
	return match, events, nil
}

// EditMatch updates a match result and recalculates standings and morale
func (s *LeagueServiceImpl) EditMatch(matchID int, homeScore int, awayScore int) (*models.Match, error) {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()

	match, err := s.matchRepo.GetByID(matchID)
	if err != nil {
		return nil, err
	}
	if homeScore < 0 || awayScore < 0 {
		return nil, fmt.Errorf("scores cannot be negative")
	}

	match.HomeScore = &homeScore
	match.AwayScore = &awayScore
	match.Status = "edited"

	if err := s.matchRepo.Update(match); err != nil {
		return nil, err
	}

	// Clear old events for this match
	s.eventRepo.DeleteByMatchID(matchID)

	// Recalculate standings
	allMatches, _ := s.matchRepo.GetAll()
	s.standingRepo.RecalculateAll(allMatches)

	// Update morale based on edit
	homeTeam, _ := s.teamRepo.GetByID(match.HomeTeamID)
	awayTeam, _ := s.teamRepo.GetByID(match.AwayTeamID)
	if homeTeam != nil && awayTeam != nil {
		s.updateTeamMetrics(homeTeam, awayTeam, homeScore, awayScore)
	}

	// Invalidate cache
	s.invalidateCache()

	match, _ = s.matchRepo.GetByID(matchID)
	return match, nil
}

// GetPredictions runs 1000 Monte Carlo simulations for championship win %
// Utilizes in-memory caching for massive performance gains
func (s *LeagueServiceImpl) GetPredictions() ([]models.Prediction, error) {
	s.cacheMu.RLock()
	if s.oracleCache != nil {
		s.cacheMu.RUnlock()
		return s.oracleCache, nil
	}
	s.cacheMu.RUnlock()

	currentWeek, err := s.matchRepo.GetCurrentWeek()
	if err != nil {
		return nil, err
	}
	if currentWeek <= 4 {
		return nil, fmt.Errorf("predictions available after week 4 (current: week %d)", currentWeek)
	}

	teams, err := s.teamRepo.GetAll()
	if err != nil {
		return nil, err
	}
	allMatches, err := s.matchRepo.GetAll()
	if err != nil {
		return nil, err
	}

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
	s.oracleCache = predictions
	s.cacheMu.Unlock()
	
	return predictions, nil
}

// Rollback reverts the league state to a specific week (Time Machine)
func (s *LeagueServiceImpl) Rollback(week int) error {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()

	if week < 1 || week > totalWeeks {
		return fmt.Errorf("invalid week %d, must be between 1 and %d", week, totalWeeks)
	}

	// Delete events from this week onward
	if err := s.eventRepo.DeleteFromWeek(week); err != nil {
		return fmt.Errorf("failed to rollback events: %w", err)
	}

	// Reset matches from this week onward
	if err := s.matchRepo.DeleteFromWeek(week); err != nil {
		return fmt.Errorf("failed to rollback matches: %w", err)
	}

	// Reset team morale/fatigue to defaults
	teams, _ := s.teamRepo.GetAll()
	for _, t := range teams {
		t.CurrentStrength = t.BaseStrength
		t.Morale = 0.7
		t.Fatigue = 0.0
		s.teamRepo.Update(&t)
	}

	// Replay morale/fatigue changes from played matches
	allMatches, _ := s.matchRepo.GetAll()
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
		s.teamRepo.Update(t)
	}

	// Recalculate standings
	s.standingRepo.RecalculateAll(allMatches)

	// Invalidate Cache
	s.invalidateCache()

	return nil
}

// GetTeamMetrics returns a team's current metrics
func (s *LeagueServiceImpl) GetTeamMetrics(teamID int) (*models.TeamMetrics, error) {
	team, err := s.teamRepo.GetByID(teamID)
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
func (s *LeagueServiceImpl) Reset() error {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()

	s.eventRepo.DeleteAll()
	s.matchRepo.DeleteAll()
	s.standingRepo.DeleteAll()
	s.playerRepo.DeleteAll()
	s.teamRepo.DeleteAll()

	database.SeedTeams(s.db)
	database.SeedPlayers(s.db)
	database.SeedStandings(s.db)
	database.GenerateSchedule(s.db)

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
func (s *LeagueServiceImpl) GetCurrentWeek() (int, error) {
	return s.matchRepo.GetCurrentWeek()
}

// updateTeamMetrics adjusts morale and fatigue after a match
func (s *LeagueServiceImpl) updateTeamMetrics(home, away *models.Team, homeGoals, awayGoals int) {
	s.applyMetricChanges(home, away, homeGoals, awayGoals)
	s.teamRepo.Update(home)
	s.teamRepo.Update(away)
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
	home.Fatigue = clamp(home.Fatigue+0.05+rand.Float64()*0.05, 0, 1)
	away.Fatigue = clamp(away.Fatigue+0.05+rand.Float64()*0.05, 0, 1)

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

	sort.Slice(standings, func(i, j int) bool {
		a, b := standings[i], standings[j]
		if a.Points != b.Points {
			return a.Points > b.Points
		}
		if a.GD != b.GD {
			return a.GD > b.GD
		}
		if a.GF != b.GF {
			return a.GF > b.GF
		}
		return a.TeamName < b.TeamName
	})

	for i := range standings {
		standings[i].Position = i + 1
	}
	return standings
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

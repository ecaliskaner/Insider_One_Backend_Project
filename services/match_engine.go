package services

import (
	"math"
	"math/rand"
	"sync"

	"github.com/ecaliskaner/Insider_One_Backend_Project/models"
)

// RNG defines an interface for random number generation to allow deterministic testing
type RNG interface {
	Float64() float64
	Intn(n int) int
}

// defaultRNG uses non-security randomness for match simulation variance.
type defaultRNG struct{}

func (d *defaultRNG) Float64() float64 { return rand.Float64() } // #nosec G404 -- sports simulation randomness, not security-sensitive.
func (d *defaultRNG) Intn(n int) int   { return rand.Intn(n) }   // #nosec G404 -- sports simulation randomness, not security-sensitive.

type lockedRNG struct {
	mu  sync.Mutex
	rng *rand.Rand
}

func NewSeededRNG(seed int64) RNG {
	return &lockedRNG{rng: rand.New(rand.NewSource(seed))} // #nosec G404 -- deterministic simulation seed for tests and benchmarks.
}

func (r *lockedRNG) Float64() float64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.rng.Float64()
}

func (r *lockedRNG) Intn(n int) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.rng.Intn(n)
}

// DefaultMatchEngine implements MatchEngine with strength/morale/fatigue/weather-based simulation.
// Acts as an Adapter — the core simulation logic is pure and testable.
type DefaultMatchEngine struct {
	rng RNG
}

func NewMatchEngine() *DefaultMatchEngine {
	return &DefaultMatchEngine{
		rng: &defaultRNG{},
	}
}

func NewMatchEngineWithRNG(rng RNG) *DefaultMatchEngine {
	return &DefaultMatchEngine{
		rng: rng,
	}
}

func NewMatchEngineWithSeed(seed int64) *DefaultMatchEngine {
	return NewMatchEngineWithRNG(NewSeededRNG(seed))
}

// SimulateMatch generates realistic match results and events.
// Factors: base strength, morale boost, fatigue penalty, home advantage, weather effects.
func (e *DefaultMatchEngine) SimulateMatch(home, away models.Team, weather string) (int, int, []models.MatchEvent) {
	// Base expected goals ~1.3
	baseGoals := 1.3
	homeAdvantage := 1.25

	// Calculate effective strength with morale and fatigue
	homeEffective := float64(home.CurrentStrength) * (1.0 + (home.Morale-0.5)*0.3) * (1.0 - home.Fatigue*0.2)
	awayEffective := float64(away.CurrentStrength) * (1.0 + (away.Morale-0.5)*0.3) * (1.0 - away.Fatigue*0.2)

	// Weather modifier
	weatherMod := weatherModifier(weather)

	homeExpected := baseGoals * (homeEffective / 70.0) * homeAdvantage * weatherMod
	awayExpected := baseGoals * (awayEffective / 70.0) * weatherMod

	homeGoals := e.poissonRandom(homeExpected)
	awayGoals := e.poissonRandom(awayExpected)

	if homeGoals > 7 {
		homeGoals = 7
	}
	if awayGoals > 7 {
		awayGoals = 7
	}

	// Generate match events
	var events []models.MatchEvent

	// Goal events
	for i := 0; i < homeGoals; i++ {
		events = append(events, models.MatchEvent{
			EventType: "Goal",
			Minute:    e.rng.Intn(90) + 1,
			Detail:    home.Name + " scores",
		})
	}
	for i := 0; i < awayGoals; i++ {
		events = append(events, models.MatchEvent{
			EventType: "Goal",
			Minute:    e.rng.Intn(90) + 1,
			Detail:    away.Name + " scores",
		})
	}

	// Quantum VAR Decision (5% chance per match)
	if e.rng.Float64() < 0.05 {
		minute := e.rng.Intn(90) + 1
		decision := "Goal overturned"
		if e.rng.Float64() < 0.5 {
			decision = "Penalty awarded"
		}
		events = append(events, models.MatchEvent{
			EventType: "Quantum VAR Decision",
			Minute:    minute,
			Detail:    decision,
		})
	}

	// Injury event (10% chance per match)
	if e.rng.Float64() < 0.10 {
		events = append(events, models.MatchEvent{
			EventType: "Injury",
			Minute:    e.rng.Intn(90) + 1,
			Detail:    "Player injury during match",
		})
	}

	return homeGoals, awayGoals, events
}

// weatherModifier adjusts goal expectations based on weather
func weatherModifier(weather string) float64 {
	switch weather {
	case "rainy":
		return 0.85 // Fewer goals in rain
	case "snowy":
		return 0.75 // Significantly fewer goals
	case "windy":
		return 0.90
	case "foggy":
		return 0.92
	default: // sunny
		return 1.0
	}
}

// poissonRandom generates a Poisson-distributed random number (Knuth's algorithm)
// We pass RNG into it so it's deterministic in tests
func (e *DefaultMatchEngine) poissonRandom(lambda float64) int {
	l := math.Exp(-lambda)
	k := 0
	p := 1.0
	for {
		k++
		p *= e.rng.Float64()
		if p < l {
			break
		}
	}
	return k - 1
}

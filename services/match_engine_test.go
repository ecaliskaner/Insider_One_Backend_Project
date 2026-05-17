package services

import (
	"testing"

	"github.com/insider/league-simulation/models"
	"github.com/stretchr/testify/assert"
)

// mockRNG implements the RNG interface for deterministic testing
type mockRNG struct {
	floatVal float64
	intVal   int
}

func (m *mockRNG) Float64() float64 { return m.floatVal }
func (m *mockRNG) Intn(n int) int   { return m.intVal }

func TestMatchEngine_SimulateMatch(t *testing.T) {
	tests := []struct {
		name          string
		homeTeam      models.Team
		awayTeam      models.Team
		weather       string
		mockFloat     float64
		mockInt       int
		expectedHomeG int
		expectedAwayG int
	}{
		{
			name: "Sunny Match Equal Teams",
			homeTeam: models.Team{
				Name: "Home", BaseStrength: 80, CurrentStrength: 80, Morale: 0.5, Fatigue: 0.0,
			},
			awayTeam: models.Team{
				Name: "Away", BaseStrength: 80, CurrentStrength: 80, Morale: 0.5, Fatigue: 0.0,
			},
			weather:       "sunny",
			mockFloat:     0.5,
			mockInt:       45,
			expectedHomeG: 2, 
			expectedAwayG: 2,
		},
		{
			name: "Rainy Match Strong Home",
			homeTeam: models.Team{
				Name: "Home", BaseStrength: 100, CurrentStrength: 100, Morale: 1.0, Fatigue: 0.0,
			},
			awayTeam: models.Team{
				Name: "Away", BaseStrength: 50, CurrentStrength: 50, Morale: 0.0, Fatigue: 0.5,
			},
			weather:       "rainy",
			mockFloat:     0.1, 
			mockInt:       10,
			expectedHomeG: 0, 
			expectedAwayG: 0, 
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rng := &mockRNG{floatVal: tt.mockFloat, intVal: tt.mockInt}
			engine := NewMatchEngineWithRNG(rng)

			homeG, awayG, events := engine.SimulateMatch(tt.homeTeam, tt.awayTeam, tt.weather)

			assert.Equal(t, tt.expectedHomeG, homeG)
			assert.Equal(t, tt.expectedAwayG, awayG)

			// Check Quantum VAR decision branch
			if tt.mockFloat < 0.05 {
				var hasQuantumVAR bool
				for _, e := range events {
					if e.EventType == "Quantum VAR Decision" {
						hasQuantumVAR = true
						break
					}
				}
				assert.True(t, hasQuantumVAR, "Expected a Quantum VAR Decision event")
			}
		})
	}
}

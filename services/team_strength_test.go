package services

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ecaliskaner/Insider_One_Backend_Project/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarketValueTeamStrengthProvider_CalculatesDeterministicStrength(t *testing.T) {
	provider := NewMarketValueTeamStrengthProvider()

	strength, err := provider.BaseStrength(context.Background(), models.Team{
		Name:         "Arsenal",
		BaseStrength: 70,
		MarketValue:  1050,
	})

	require.NoError(t, err)
	assert.Equal(t, 92, strength)
}

func TestTransfermarktTeamStrengthProvider_UsesExternalMarketValueAndCaches(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		assert.Equal(t, "/clubs/search/Arsenal", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`{"clubs":[{"name":"Arsenal","marketValue":"€1.05bn"}]}`))
		require.NoError(t, err)
	}))
	defer server.Close()

	provider := NewTransfermarktTeamStrengthProvider(server.URL, NewLocalTeamStrengthProvider())
	team := models.Team{Name: "Arsenal", BaseStrength: 70, MarketValue: 800}

	strength, err := provider.BaseStrength(context.Background(), team)
	require.NoError(t, err)
	cachedStrength, err := provider.BaseStrength(context.Background(), team)
	require.NoError(t, err)

	assert.Equal(t, 92, strength)
	assert.Equal(t, strength, cachedStrength)
	assert.Equal(t, 1, requests)
}

func TestTransfermarktTeamStrengthProvider_FallsBackOnProviderError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()

	provider := NewTransfermarktTeamStrengthProvider(server.URL, NewLocalTeamStrengthProvider())

	strength, err := provider.BaseStrength(context.Background(), models.Team{
		Name:         "Chelsea",
		BaseStrength: 75,
		MarketValue:  900,
	})

	require.NoError(t, err)
	assert.Equal(t, 75, strength)
}

func TestParseMarketValueString(t *testing.T) {
	value, ok := parseMarketValueString("€1.05bn")
	assert.True(t, ok)
	assert.Equal(t, 1050.0, value)

	value, ok = parseMarketValueString("€900.00m")
	assert.True(t, ok)
	assert.Equal(t, 900.0, value)
}

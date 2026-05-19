package services

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type staticWeatherAdapter struct {
	condition string
}

func (s staticWeatherAdapter) GetWeather(string) string {
	return s.condition
}

func TestOpenMeteoWeatherAdapter_MapsCurrentWeather(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		assert.Equal(t, "51.5072", r.URL.Query().Get("latitude"))
		assert.Equal(t, "-0.1276", r.URL.Query().Get("longitude"))
		assert.Equal(t, "weather_code,wind_speed_10m", r.URL.Query().Get("current"))
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`{"current":{"weather_code":61,"wind_speed_10m":12.3}}`))
		require.NoError(t, err)
	}))
	defer server.Close()

	adapter := NewOpenMeteoWeatherAdapter(
		staticWeatherAdapter{condition: "sunny"},
		WithOpenMeteoEndpoint(server.URL),
	)

	assert.Equal(t, "rainy", adapter.GetWeather("London"))
	assert.Equal(t, "rainy", adapter.GetWeather("London"))
	assert.Equal(t, 1, requests)
}

func TestOpenMeteoWeatherAdapter_FallsBackOnProviderError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()

	adapter := NewOpenMeteoWeatherAdapter(
		staticWeatherAdapter{condition: "foggy"},
		WithOpenMeteoEndpoint(server.URL),
	)

	assert.Equal(t, "foggy", adapter.GetWeather("London"))
}

func TestNewWeatherAdapterByProvider_UsesFallbackByDefault(t *testing.T) {
	fallback := staticWeatherAdapter{condition: "windy"}

	adapter := NewWeatherAdapterByProvider("unknown", fallback)

	assert.Equal(t, "windy", adapter.GetWeather("London"))
}

func TestMapOpenMeteoCondition(t *testing.T) {
	assert.Equal(t, "windy", mapOpenMeteoCondition(0, 40))
	assert.Equal(t, "foggy", mapOpenMeteoCondition(45, 2))
	assert.Equal(t, "rainy", mapOpenMeteoCondition(61, 2))
	assert.Equal(t, "snowy", mapOpenMeteoCondition(73, 2))
	assert.Equal(t, "sunny", mapOpenMeteoCondition(1, 2))
}

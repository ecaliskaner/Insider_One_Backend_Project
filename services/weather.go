package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	WeatherProviderLocal     = "local"
	WeatherProviderOpenMeteo = "open-meteo"
)

// DefaultWeatherAdapter implements local deterministic weather generation.
type DefaultWeatherAdapter struct {
	rng RNG
}

func NewWeatherAdapter() *DefaultWeatherAdapter {
	return NewWeatherAdapterWithRNG(&defaultRNG{})
}

func NewWeatherAdapterWithRNG(rng RNG) *DefaultWeatherAdapter {
	return &DefaultWeatherAdapter{rng: rng}
}

func NewWeatherAdapterWithSeed(seed int64) *DefaultWeatherAdapter {
	return NewWeatherAdapterWithRNG(NewSeededRNG(seed))
}

// GetWeather generates a weather condition based on city.
func (w *DefaultWeatherAdapter) GetWeather(city string) string {
	conditions := []string{"sunny", "rainy", "snowy", "windy", "foggy", "sunny", "sunny", "sunny"}

	switch city {
	case "Manchester", "Liverpool":
		conditions = []string{"rainy", "rainy", "windy", "sunny", "foggy", "rainy", "sunny", "sunny"}
	case "London":
		conditions = []string{"sunny", "rainy", "foggy", "sunny", "windy", "sunny", "sunny", "rainy"}
	}

	return conditions[w.rng.Intn(len(conditions))]
}

type CityCoordinates struct {
	Latitude  float64
	Longitude float64
}

type OpenMeteoWeatherAdapter struct {
	client       *http.Client
	endpoint     string
	fallback     WeatherAdapter
	cacheTTL     time.Duration
	cacheMu      sync.RWMutex
	cache        map[string]cachedWeather
	coordinates  map[string]CityCoordinates
	requestClock func() time.Time
}

type cachedWeather struct {
	condition string
	expiresAt time.Time
}

type OpenMeteoOption func(*OpenMeteoWeatherAdapter)

func NewWeatherAdapterByProvider(provider string, fallback WeatherAdapter) WeatherAdapter {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case WeatherProviderOpenMeteo:
		return NewOpenMeteoWeatherAdapter(fallback)
	default:
		return fallback
	}
}

func NewOpenMeteoWeatherAdapter(fallback WeatherAdapter, opts ...OpenMeteoOption) *OpenMeteoWeatherAdapter {
	if fallback == nil {
		fallback = NewWeatherAdapter()
	}

	adapter := &OpenMeteoWeatherAdapter{
		client: &http.Client{
			Timeout: 2 * time.Second,
		},
		endpoint:     "https://api.open-meteo.com/v1/forecast",
		fallback:     fallback,
		cacheTTL:     30 * time.Minute,
		cache:        make(map[string]cachedWeather),
		coordinates:  defaultCityCoordinates(),
		requestClock: time.Now,
	}
	for _, opt := range opts {
		opt(adapter)
	}
	return adapter
}

func WithOpenMeteoClient(client *http.Client) OpenMeteoOption {
	return func(adapter *OpenMeteoWeatherAdapter) {
		if client != nil {
			adapter.client = client
		}
	}
}

func WithOpenMeteoEndpoint(endpoint string) OpenMeteoOption {
	return func(adapter *OpenMeteoWeatherAdapter) {
		if endpoint != "" {
			adapter.endpoint = endpoint
		}
	}
}

func WithOpenMeteoCacheTTL(ttl time.Duration) OpenMeteoOption {
	return func(adapter *OpenMeteoWeatherAdapter) {
		adapter.cacheTTL = ttl
	}
}

func (w *OpenMeteoWeatherAdapter) GetWeather(city string) string {
	normalizedCity := strings.TrimSpace(city)
	if normalizedCity == "" {
		return w.fallback.GetWeather(city)
	}

	if condition, ok := w.cached(normalizedCity); ok {
		return condition
	}

	coords, ok := w.coordinates[normalizedCity]
	if !ok {
		return w.fallback.GetWeather(city)
	}

	condition, err := w.fetchWeather(context.Background(), coords)
	if err != nil {
		return w.fallback.GetWeather(city)
	}

	w.store(normalizedCity, condition)
	return condition
}

func (w *OpenMeteoWeatherAdapter) cached(city string) (string, bool) {
	w.cacheMu.RLock()
	defer w.cacheMu.RUnlock()

	entry, ok := w.cache[city]
	if !ok || w.requestClock().After(entry.expiresAt) {
		return "", false
	}
	return entry.condition, true
}

func (w *OpenMeteoWeatherAdapter) store(city string, condition string) {
	if w.cacheTTL <= 0 {
		return
	}

	w.cacheMu.Lock()
	defer w.cacheMu.Unlock()
	w.cache[city] = cachedWeather{
		condition: condition,
		expiresAt: w.requestClock().Add(w.cacheTTL),
	}
}

func (w *OpenMeteoWeatherAdapter) fetchWeather(ctx context.Context, coords CityCoordinates) (string, error) {
	endpoint, err := url.Parse(w.endpoint)
	if err != nil {
		return "", fmt.Errorf("parse Open-Meteo endpoint: %w", err)
	}

	q := endpoint.Query()
	q.Set("latitude", fmt.Sprintf("%.4f", coords.Latitude))
	q.Set("longitude", fmt.Sprintf("%.4f", coords.Longitude))
	q.Set("current", "weather_code,wind_speed_10m")
	q.Set("timezone", "auto")
	endpoint.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return "", fmt.Errorf("create Open-Meteo request: %w", err)
	}

	resp, err := w.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("call Open-Meteo: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("Open-Meteo returned status %d", resp.StatusCode)
	}

	var payload struct {
		Current struct {
			WeatherCode int     `json:"weather_code"`
			WindSpeed   float64 `json:"wind_speed_10m"`
		} `json:"current"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", fmt.Errorf("decode Open-Meteo response: %w", err)
	}

	return mapOpenMeteoCondition(payload.Current.WeatherCode, payload.Current.WindSpeed), nil
}

func mapOpenMeteoCondition(weatherCode int, windSpeed float64) string {
	if windSpeed >= 35 {
		return "windy"
	}

	switch {
	case weatherCode == 45 || weatherCode == 48:
		return "foggy"
	case weatherCode >= 51 && weatherCode <= 67:
		return "rainy"
	case weatherCode >= 71 && weatherCode <= 77:
		return "snowy"
	case weatherCode >= 80 && weatherCode <= 82:
		return "rainy"
	case weatherCode >= 85 && weatherCode <= 86:
		return "snowy"
	case weatherCode >= 95 && weatherCode <= 99:
		return "rainy"
	default:
		return "sunny"
	}
}

func defaultCityCoordinates() map[string]CityCoordinates {
	return map[string]CityCoordinates{
		"London":     {Latitude: 51.5072, Longitude: -0.1276},
		"Manchester": {Latitude: 53.4808, Longitude: -2.2426},
		"Liverpool":  {Latitude: 53.4084, Longitude: -2.9916},
		"Chelsea":    {Latitude: 51.4875, Longitude: -0.1687},
	}
}

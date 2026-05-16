package services

import "math/rand"

// DefaultWeatherAdapter implements WeatherAdapter.
// Acts as an external API Adapter — can be swapped for a real weather API.
type DefaultWeatherAdapter struct{}

func NewWeatherAdapter() *DefaultWeatherAdapter {
	return &DefaultWeatherAdapter{}
}

// GetWeather generates a weather condition based on city.
// In production, this would call an external weather API.
func (w *DefaultWeatherAdapter) GetWeather(city string) string {
	conditions := []string{"sunny", "rainy", "snowy", "windy", "foggy", "sunny", "sunny", "sunny"}

	// Cities in England have more rain
	switch city {
	case "Manchester", "Liverpool":
		conditions = []string{"rainy", "rainy", "windy", "sunny", "foggy", "rainy", "sunny", "sunny"}
	case "London":
		conditions = []string{"sunny", "rainy", "foggy", "sunny", "windy", "sunny", "sunny", "rainy"}
	}

	return conditions[rand.Intn(len(conditions))]
}

package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockWeatherAdapter mocks the WeatherAdapter
type MockWeatherAdapter struct {
	mock.Mock
}

func (m *MockWeatherAdapter) GetWeather(city string) string {
	args := m.Called(city)
	return args.String(0)
}

func TestLeagueService_WeatherMock(t *testing.T) {
	mockWeather := new(MockWeatherAdapter)

	// Setup expectations
	mockWeather.On("GetWeather", "London").Return("rainy")
	mockWeather.On("GetWeather", "Manchester").Return("sunny")

	assert.Equal(t, "rainy", mockWeather.GetWeather("London"))
	assert.Equal(t, "sunny", mockWeather.GetWeather("Manchester"))

	mockWeather.AssertExpectations(t)
}

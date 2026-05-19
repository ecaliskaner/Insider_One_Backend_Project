package services

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ecaliskaner/Insider_One_Backend_Project/models"
)

const (
	TeamStrengthProviderLocal         = "local"
	TeamStrengthProviderMarketValue   = "market-value"
	TeamStrengthProviderTransfermarkt = "transfermarkt"
)

// TeamStrengthProvider resolves a team's base strength before simulation.
type TeamStrengthProvider interface {
	BaseStrength(ctx context.Context, team models.Team) (int, error)
}

type LocalTeamStrengthProvider struct{}

func NewLocalTeamStrengthProvider() LocalTeamStrengthProvider {
	return LocalTeamStrengthProvider{}
}

func (LocalTeamStrengthProvider) BaseStrength(_ context.Context, team models.Team) (int, error) {
	return clampStrength(team.BaseStrength), nil
}

type MarketValueTeamStrengthProvider struct{}

func NewMarketValueTeamStrengthProvider() MarketValueTeamStrengthProvider {
	return MarketValueTeamStrengthProvider{}
}

func (MarketValueTeamStrengthProvider) BaseStrength(_ context.Context, team models.Team) (int, error) {
	return strengthFromMarketValue(team.MarketValue), nil
}

type TransfermarktTeamStrengthProvider struct {
	client   *http.Client
	baseURL  string
	fallback TeamStrengthProvider
	cacheTTL time.Duration
	cacheMu  sync.RWMutex
	cache    map[string]cachedStrength
}

type cachedStrength struct {
	strength  int
	expiresAt time.Time
}

type TransfermarktOption func(*TransfermarktTeamStrengthProvider)

func NewTeamStrengthProviderByProvider(provider string, transfermarktBaseURL string) TeamStrengthProvider {
	local := NewLocalTeamStrengthProvider()
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case TeamStrengthProviderMarketValue:
		return NewMarketValueTeamStrengthProvider()
	case TeamStrengthProviderTransfermarkt:
		return NewTransfermarktTeamStrengthProvider(transfermarktBaseURL, local)
	default:
		return local
	}
}

func NewTransfermarktTeamStrengthProvider(baseURL string, fallback TeamStrengthProvider, opts ...TransfermarktOption) *TransfermarktTeamStrengthProvider {
	if fallback == nil {
		fallback = NewLocalTeamStrengthProvider()
	}
	provider := &TransfermarktTeamStrengthProvider{
		client: &http.Client{
			Timeout: 2 * time.Second,
		},
		baseURL:  strings.TrimRight(baseURL, "/"),
		fallback: fallback,
		cacheTTL: 12 * time.Hour,
		cache:    make(map[string]cachedStrength),
	}
	for _, opt := range opts {
		opt(provider)
	}
	return provider
}

func WithTransfermarktClient(client *http.Client) TransfermarktOption {
	return func(provider *TransfermarktTeamStrengthProvider) {
		if client != nil {
			provider.client = client
		}
	}
}

func WithTransfermarktCacheTTL(ttl time.Duration) TransfermarktOption {
	return func(provider *TransfermarktTeamStrengthProvider) {
		provider.cacheTTL = ttl
	}
}

func (p *TransfermarktTeamStrengthProvider) BaseStrength(ctx context.Context, team models.Team) (int, error) {
	teamName := strings.TrimSpace(team.Name)
	if teamName == "" || p.baseURL == "" {
		return p.fallback.BaseStrength(ctx, team)
	}

	if strength, ok := p.cached(teamName); ok {
		return strength, nil
	}

	marketValue, err := p.fetchMarketValue(ctx, teamName)
	if err != nil {
		return p.fallback.BaseStrength(ctx, team)
	}

	strength := strengthFromMarketValue(marketValue)
	p.store(teamName, strength)
	return strength, nil
}

func (p *TransfermarktTeamStrengthProvider) cached(teamName string) (int, bool) {
	p.cacheMu.RLock()
	defer p.cacheMu.RUnlock()

	entry, ok := p.cache[teamName]
	if !ok || time.Now().After(entry.expiresAt) {
		return 0, false
	}
	return entry.strength, true
}

func (p *TransfermarktTeamStrengthProvider) store(teamName string, strength int) {
	if p.cacheTTL <= 0 {
		return
	}
	p.cacheMu.Lock()
	defer p.cacheMu.Unlock()
	p.cache[teamName] = cachedStrength{strength: strength, expiresAt: time.Now().Add(p.cacheTTL)}
}

func (p *TransfermarktTeamStrengthProvider) fetchMarketValue(ctx context.Context, teamName string) (float64, error) {
	endpoint := fmt.Sprintf("%s/clubs/search/%s", p.baseURL, url.PathEscape(teamName))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return 0, fmt.Errorf("create Transfermarkt request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("call Transfermarkt provider: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return 0, fmt.Errorf("Transfermarkt provider returned status %d", resp.StatusCode)
	}

	var payload interface{}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return 0, fmt.Errorf("decode Transfermarkt response: %w", err)
	}

	value, ok := findMarketValue(payload)
	if !ok {
		return 0, fmt.Errorf("market value missing from Transfermarkt response")
	}
	return value, nil
}

func findMarketValue(value interface{}) (float64, bool) {
	switch typed := value.(type) {
	case map[string]interface{}:
		for _, key := range []string{"marketValue", "market_value", "marketValueNumeric", "market_value_numeric"} {
			if raw, ok := typed[key]; ok {
				if parsed, ok := parseMarketValue(raw); ok {
					return parsed, true
				}
			}
		}
		for _, nested := range typed {
			if parsed, ok := findMarketValue(nested); ok {
				return parsed, true
			}
		}
	case []interface{}:
		for _, item := range typed {
			if parsed, ok := findMarketValue(item); ok {
				return parsed, true
			}
		}
	}
	return 0, false
}

func parseMarketValue(value interface{}) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, typed > 0
	case string:
		return parseMarketValueString(typed)
	default:
		return 0, false
	}
}

var marketValueNumber = regexp.MustCompile(`[-+]?[0-9]*\.?[0-9]+`)

func parseMarketValueString(value string) (float64, bool) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	match := marketValueNumber.FindString(normalized)
	if match == "" {
		return 0, false
	}
	number, err := strconv.ParseFloat(match, 64)
	if err != nil || number <= 0 {
		return 0, false
	}
	switch {
	case strings.Contains(normalized, "bn") || strings.Contains(normalized, "billion"):
		return number * 1000, true
	case strings.Contains(normalized, "k"):
		return number / 1000, true
	default:
		return number, true
	}
}

func strengthFromMarketValue(marketValue float64) int {
	strength := 55 + int(math.Round(math.Sqrt(math.Max(marketValue, 0))*1.15))
	return clampStrength(strength)
}

func clampStrength(strength int) int {
	if strength < 40 {
		return 40
	}
	if strength > 95 {
		return 95
	}
	return strength
}

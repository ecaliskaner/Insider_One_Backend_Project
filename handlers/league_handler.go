package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/insider/league-simulation/services"
)

// LeagueHandler handles HTTP requests for the league API
type LeagueHandler struct {
	service services.LeagueService
}

func NewLeagueHandler(service services.LeagueService) *LeagueHandler {
	return &LeagueHandler{service: service}
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}

// GetTable — GET /api/v1/league/table
// Returns current standings (PTS, W, D, L, GD)
func (h *LeagueHandler) GetTable(w http.ResponseWriter, r *http.Request) {
	standings, err := h.service.GetStandings()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	currentWeek, _ := h.service.GetCurrentWeek()
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"current_week": currentWeek,
		"standings":    standings,
	})
}

// PlayNextWeek — POST /api/v1/league/next-week
// Simulates the next week's matches and updates state
func (h *LeagueHandler) PlayNextWeek(w http.ResponseWriter, r *http.Request) {
	result, err := h.service.PlayNextWeek()
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	standings, _ := h.service.GetStandings()
	currentWeek, _ := h.service.GetCurrentWeek()
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message":      "Week simulated successfully",
		"week_result":  result,
		"standings":    standings,
		"current_week": currentWeek,
	})
}

// PlayAll — POST /api/v1/league/play-all
// Simulates all remaining matches in the season
func (h *LeagueHandler) PlayAll(w http.ResponseWriter, r *http.Request) {
	results, err := h.service.PlayAll()
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	standings, _ := h.service.GetStandings()
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message":   "All remaining weeks simulated",
		"results":   results,
		"standings": standings,
	})
}

// EditMatch — PUT /api/v1/matches/{id}
// Edits a specific match result; recalculates standings and morale
func (h *LeagueHandler) EditMatch(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	matchID, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid match ID")
		return
	}

	var req struct {
		HomeScore int `json:"home_score"`
		AwayScore int `json:"away_score"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body: expected {home_score, away_score}")
		return
	}

	match, err := h.service.EditMatch(matchID, req.HomeScore, req.AwayScore)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	standings, _ := h.service.GetStandings()
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message":   "Match updated, standings and morale recalculated",
		"match":     match,
		"standings": standings,
	})
}

// GetOracle — GET /api/v1/simulation/oracle
// Runs 1,000 Monte Carlo simulations to calculate Championship Win %
func (h *LeagueHandler) GetOracle(w http.ResponseWriter, r *http.Request) {
	predictions, err := h.service.GetPredictions()
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	standings, _ := h.service.GetStandings()
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"simulation_count": 1000,
		"predictions":      predictions,
		"current_standings": standings,
	})
}

// Rollback — POST /api/v1/league/rollback/{week}
// Time Machine: Reverts database state to a specific week
func (h *LeagueHandler) Rollback(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	week, err := strconv.Atoi(vars["week"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid week number")
		return
	}

	if err := h.service.Rollback(week); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	standings, _ := h.service.GetStandings()
	currentWeek, _ := h.service.GetCurrentWeek()
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message":      fmt.Sprintf("Time machine: reverted to week %d", week),
		"current_week": currentWeek,
		"standings":    standings,
	})
}

// GetTeamMetrics — GET /api/v1/teams/{id}/metrics
// Returns a team's current Strength, Morale, Fatigue, and Market Value
func (h *LeagueHandler) GetTeamMetrics(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	teamID, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid team ID")
		return
	}

	metrics, err := h.service.GetTeamMetrics(teamID)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, metrics)
}

// Reset — POST /api/v1/league/reset
func (h *LeagueHandler) Reset(w http.ResponseWriter, r *http.Request) {
	if err := h.service.Reset(); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{
		"message": "League reset successfully",
	})
}

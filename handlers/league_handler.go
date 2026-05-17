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

// APIResponse standardizes all HTTP JSON responses
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Meta    interface{} `json:"meta,omitempty"`
}

func respondJSON(w http.ResponseWriter, status int, data interface{}, meta interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(APIResponse{
		Success: true,
		Data:    data,
		Meta:    meta,
	})
}

func respondError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(APIResponse{
		Success: false,
		Error:   message,
	})
}

// GetTable godoc
// @Summary      Get current standings
// @Description  Returns current league standings (PTS, W, D, L, GD)
// @Tags         league
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /league/table [get]
func (h *LeagueHandler) GetTable(w http.ResponseWriter, r *http.Request) {
	standings, err := h.service.GetStandings(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	currentWeek, _ := h.service.GetCurrentWeek(r.Context())
	respondJSON(w, http.StatusOK, standings, map[string]interface{}{
		"current_week": currentWeek,
	})
}

// PlayNextWeek godoc
// @Summary      Simulate next week
// @Description  Simulates the next week's matches and updates state
// @Tags         league
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /league/next-week [post]
func (h *LeagueHandler) PlayNextWeek(w http.ResponseWriter, r *http.Request) {
	result, err := h.service.PlayNextWeek(r.Context())
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	standings, _ := h.service.GetStandings(r.Context())
	currentWeek, _ := h.service.GetCurrentWeek(r.Context())
	respondJSON(w, http.StatusOK, result, map[string]interface{}{
		"message":      "Week simulated successfully",
		"standings":    standings,
		"current_week": currentWeek,
	})
}

// PlayAll godoc
// @Summary      Play all remaining weeks
// @Description  Simulates all remaining matches in the season
// @Tags         league
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /league/play-all [post]
func (h *LeagueHandler) PlayAll(w http.ResponseWriter, r *http.Request) {
	results, err := h.service.PlayAll(r.Context())
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	standings, _ := h.service.GetStandings(r.Context())
	respondJSON(w, http.StatusOK, results, map[string]interface{}{
		"message":   "All remaining weeks simulated",
		"standings": standings,
	})
}

// GetMatch godoc
// @Summary      Get match details
// @Description  Returns a specific match and its events
// @Tags         matches
// @Produce      json
// @Param        id   path      int  true  "Match ID"
// @Success      200  {object}  map[string]interface{}
// @Router       /matches/{id} [get]
func (h *LeagueHandler) GetMatch(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	matchID, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid match ID")
		return
	}

	match, events, err := h.service.GetMatch(r.Context(), matchID)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"match":  match,
		"events": events,
	}, nil)
}

// EditMatch godoc
// @Summary      Edit match result
// @Description  Edits a specific match result; recalculates standings and morale
// @Tags         matches
// @Accept       json
// @Produce      json
// @Param        id   path      int  true  "Match ID"
// @Param        body body      map[string]int true "Scores: {home_score: 1, away_score: 2}"
// @Success      200  {object}  map[string]interface{}
// @Router       /matches/{id} [put]
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

	match, err := h.service.EditMatch(r.Context(), matchID, req.HomeScore, req.AwayScore)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	standings, _ := h.service.GetStandings(r.Context())
	respondJSON(w, http.StatusOK, match, map[string]interface{}{
		"message":   "Match updated, standings and morale recalculated",
		"standings": standings,
	})
}

// GetOracle godoc
// @Summary      Monte Carlo predictions
// @Description  Runs 1,000 Monte Carlo simulations to calculate Championship Win %
// @Tags         simulation
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /simulation/oracle [get]
func (h *LeagueHandler) GetOracle(w http.ResponseWriter, r *http.Request) {
	predictions, err := h.service.GetPredictions(r.Context())
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	standings, _ := h.service.GetStandings(r.Context())
	respondJSON(w, http.StatusOK, predictions, map[string]interface{}{
		"simulation_count":  1000,
		"current_standings": standings,
	})
}

// Rollback godoc
// @Summary      Time Machine rollback
// @Description  Reverts database state to a specific week
// @Tags         league
// @Produce      json
// @Param        week path      int  true  "Week to rollback to"
// @Success      200  {object}  map[string]interface{}
// @Router       /league/rollback/{week} [post]
func (h *LeagueHandler) Rollback(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	week, err := strconv.Atoi(vars["week"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid week number")
		return
	}

	if err := h.service.Rollback(r.Context(), week); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	standings, _ := h.service.GetStandings(r.Context())
	currentWeek, _ := h.service.GetCurrentWeek(r.Context())
	respondJSON(w, http.StatusOK, nil, map[string]interface{}{
		"message":      fmt.Sprintf("Time machine: reverted to week %d", week),
		"current_week": currentWeek,
		"standings":    standings,
	})
}

// GetTeamMetrics godoc
// @Summary      Get team metrics
// @Description  Returns a team's current Strength, Morale, Fatigue, and Market Value
// @Tags         teams
// @Produce      json
// @Param        id   path      int  true  "Team ID"
// @Success      200  {object}  models.TeamMetrics
// @Router       /teams/{id}/metrics [get]
func (h *LeagueHandler) GetTeamMetrics(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	teamID, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid team ID")
		return
	}

	metrics, err := h.service.GetTeamMetrics(r.Context(), teamID)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, metrics, nil)
}

// Reset godoc
// @Summary      Reset league
// @Description  Resets the entire league to initial state
// @Tags         league
// @Produce      json
// @Success      200  {object}  map[string]string
// @Router       /league/reset [post]
func (h *LeagueHandler) Reset(w http.ResponseWriter, r *http.Request) {
	if err := h.service.Reset(r.Context()); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, nil, map[string]string{
		"message": "League reset successfully",
	})
}

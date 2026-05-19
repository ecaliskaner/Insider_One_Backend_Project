package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/ecaliskaner/Insider_One_Backend_Project/services"
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
	Meta    interface{} `json:"meta,omitempty"`
}

type EditMatchRequest struct {
	HomeScore *int `json:"home_score" example:"3"`
	AwayScore *int `json:"away_score" example:"1"`
}

func respondJSON(w http.ResponseWriter, status int, data interface{}, meta interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(APIResponse{
		Success: true,
		Data:    data,
		Meta:    meta,
	}); err != nil {
		slog.Error("failed to write JSON response", slog.Any("error", err))
	}
}

// GetTable godoc
// @Summary      Get current standings
// @Description  Returns current league standings (PTS, W, D, L, GD)
// @Tags         league
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Failure      500  {object}  ProblemDetails
// @Router       /league/table [get]
func (h *LeagueHandler) GetTable(w http.ResponseWriter, r *http.Request) {
	standings, err := h.service.GetStandings(r.Context())
	if err != nil {
		WriteProblem(w, r, http.StatusInternalServerError, "Internal Server Error", err.Error(), "https://api.insiderfootball.com/errors/internal")
		return
	}
	currentWeek, err := h.service.GetCurrentWeek(r.Context())
	if err != nil {
		WriteProblem(w, r, http.StatusInternalServerError, "Internal Server Error", err.Error(), "https://api.insiderfootball.com/errors/internal")
		return
	}
	respondJSON(w, http.StatusOK, standings, map[string]interface{}{
		"current_week": currentWeek,
	})
}

// GetOverview godoc
// @Summary      Get league overview
// @Description  Returns the current league table, weekly match status, and predictions when available
// @Tags         league
// @Produce      json
// @Success      200  {object}  models.LeagueOverview
// @Failure      500  {object}  ProblemDetails
// @Router       /league/overview [get]
func (h *LeagueHandler) GetOverview(w http.ResponseWriter, r *http.Request) {
	overview, err := h.service.GetOverview(r.Context())
	if err != nil {
		WriteProblem(w, r, http.StatusInternalServerError, "Internal Server Error", err.Error(), "https://api.insiderfootball.com/errors/internal")
		return
	}
	respondJSON(w, http.StatusOK, overview, nil)
}

// PlayNextWeek godoc
// @Summary      Simulate next week
// @Description  Simulates the next week's matches and updates state
// @Tags         league
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  ProblemDetails
// @Failure      500  {object}  ProblemDetails
// @Router       /league/next-week [post]
func (h *LeagueHandler) PlayNextWeek(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	currentWeek, err := h.service.GetCurrentWeek(ctx)
	if err != nil {
		WriteProblem(w, r, http.StatusInternalServerError, "Internal Server Error", "Failed to resolve current schedule milestone.", "https://api.insiderfootball.com/errors/internal")
		return
	}

	if currentWeek > 6 {
		WriteProblem(w, r, http.StatusBadRequest,
			"Season Overrun Prevented",
			"Cannot simulate next week. The 6-week league campaign has already concluded.",
			"https://api.insiderfootball.com/errors/season-complete",
		)
		return
	}

	result, err := h.service.PlayNextWeek(ctx)
	if err != nil {
		WriteProblem(w, r, http.StatusBadRequest, "Simulation Error", err.Error(), "https://api.insiderfootball.com/errors/simulation-failed")
		return
	}
	standings, err := h.service.GetStandings(r.Context())
	if err != nil {
		WriteProblem(w, r, http.StatusInternalServerError, "Internal Server Error", err.Error(), "https://api.insiderfootball.com/errors/internal")
		return
	}
	currentWeek, err = h.service.GetCurrentWeek(r.Context())
	if err != nil {
		WriteProblem(w, r, http.StatusInternalServerError, "Internal Server Error", err.Error(), "https://api.insiderfootball.com/errors/internal")
		return
	}
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
// @Failure      400  {object}  ProblemDetails
// @Failure      500  {object}  ProblemDetails
// @Router       /league/play-all [post]
func (h *LeagueHandler) PlayAll(w http.ResponseWriter, r *http.Request) {
	results, err := h.service.PlayAll(r.Context())
	if err != nil {
		WriteProblem(w, r, http.StatusBadRequest, "Simulation Error", err.Error(), "https://api.insiderfootball.com/errors/simulation-failed")
		return
	}
	standings, err := h.service.GetStandings(r.Context())
	if err != nil {
		WriteProblem(w, r, http.StatusInternalServerError, "Internal Server Error", err.Error(), "https://api.insiderfootball.com/errors/internal")
		return
	}
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
// @Failure      400  {object}  ProblemDetails
// @Failure      404  {object}  ProblemDetails
// @Router       /matches/{id} [get]
func (h *LeagueHandler) GetMatch(w http.ResponseWriter, r *http.Request) {
	matchID, err := parsePathInt(r, "id", "Match ID")
	if err != nil {
		WriteProblem(w, r, http.StatusBadRequest, "Invalid Match ID", err.Error()+".", "https://api.insiderfootball.com/errors/invalid-id")
		return
	}

	match, events, err := h.service.GetMatch(r.Context(), matchID)
	if err != nil {
		WriteProblem(w, r, http.StatusNotFound, "Match Not Found", err.Error(), "https://api.insiderfootball.com/errors/not-found")
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
// @Param        body body      EditMatchRequest true "Edited score"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  ProblemDetails
// @Failure      500  {object}  ProblemDetails
// @Router       /matches/{id} [put]
func (h *LeagueHandler) EditMatch(w http.ResponseWriter, r *http.Request) {
	matchID, err := parsePathInt(r, "id", "Match ID")
	if err != nil {
		WriteProblem(w, r, http.StatusBadRequest, "Invalid Match ID", err.Error()+".", "https://api.insiderfootball.com/errors/invalid-id")
		return
	}

	var req EditMatchRequest
	if err := decodeStrictJSON(w, r, &req); err != nil {
		WriteProblem(w, r, http.StatusBadRequest, "Invalid Request Body", err.Error()+".", "https://api.insiderfootball.com/errors/invalid-body")
		return
	}
	if err := validateEditMatchRequest(req); err != nil {
		WriteProblem(w, r, http.StatusBadRequest, "Invalid Request Body", err.Error()+".", "https://api.insiderfootball.com/errors/invalid-body")
		return
	}

	match, err := h.service.EditMatch(r.Context(), matchID, *req.HomeScore, *req.AwayScore)
	if err != nil {
		WriteProblem(w, r, http.StatusBadRequest, "Match Edit Failed", err.Error(), "https://api.insiderfootball.com/errors/edit-failed")
		return
	}

	standings, err := h.service.GetStandings(r.Context())
	if err != nil {
		WriteProblem(w, r, http.StatusInternalServerError, "Internal Server Error", err.Error(), "https://api.insiderfootball.com/errors/internal")
		return
	}
	respondJSON(w, http.StatusOK, match, map[string]interface{}{
		"message":   "Match updated, standings and morale recalculated",
		"standings": standings,
	})
}

// GetChampionshipProbabilities godoc
// @Summary      Championship probabilities
// @Description  Runs 1,000 Monte Carlo simulations to calculate Championship Win %
// @Tags         simulation
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  ProblemDetails
// @Failure      500  {object}  ProblemDetails
// @Router       /simulation/championship-probabilities [get]
func (h *LeagueHandler) GetChampionshipProbabilities(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	currentWeek, err := h.service.GetCurrentWeek(ctx)
	if err != nil {
		WriteProblem(w, r, http.StatusInternalServerError, "Internal Server Error", "Failed to resolve current schedule milestone.", "https://api.insiderfootball.com/errors/internal")
		return
	}

	if currentWeek <= 4 {
		WriteProblem(w, r, http.StatusBadRequest,
			"Premature Championship Probability Request",
			"Championship win probabilities are mathematically volatile and unavailable until Week 4 data constraints are met.",
			"https://api.insiderfootball.com/errors/premature-championship-probabilities",
		)
		return
	}

	predictions, err := h.service.GetPredictions(ctx)
	if err != nil {
		WriteProblem(w, r, http.StatusBadRequest, "Championship Probability Error", err.Error(), "https://api.insiderfootball.com/errors/championship-probabilities-failed")
		return
	}
	standings, err := h.service.GetStandings(r.Context())
	if err != nil {
		WriteProblem(w, r, http.StatusInternalServerError, "Internal Server Error", err.Error(), "https://api.insiderfootball.com/errors/internal")
		return
	}
	respondJSON(w, http.StatusOK, predictions, map[string]interface{}{
		"simulation_count":  1000,
		"current_standings": standings,
	})
}

// Rollback godoc
// @Summary      Rollback league state
// @Description  Reverts database state to a specific week
// @Tags         league
// @Produce      json
// @Param        week path      int  true  "Week to rollback to"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  ProblemDetails
// @Failure      500  {object}  ProblemDetails
// @Router       /league/rollback/{week} [post]
func (h *LeagueHandler) Rollback(w http.ResponseWriter, r *http.Request) {
	targetWeek, err := parseBoundedPathInt(r, "week", "Target week", 0, 6)
	if err != nil {
		WriteProblem(w, r, http.StatusBadRequest,
			"Invalid Rollback Target Bounds",
			err.Error()+".",
			"https://api.insiderfootball.com/errors/invalid-rollback-bounds",
		)
		return
	}

	if err := h.service.Rollback(r.Context(), targetWeek); err != nil {
		WriteProblem(w, r, http.StatusBadRequest, "Rollback Failed", err.Error(), "https://api.insiderfootball.com/errors/rollback-failed")
		return
	}

	standings, err := h.service.GetStandings(r.Context())
	if err != nil {
		WriteProblem(w, r, http.StatusInternalServerError, "Internal Server Error", err.Error(), "https://api.insiderfootball.com/errors/internal")
		return
	}
	currentWeek, err := h.service.GetCurrentWeek(r.Context())
	if err != nil {
		WriteProblem(w, r, http.StatusInternalServerError, "Internal Server Error", err.Error(), "https://api.insiderfootball.com/errors/internal")
		return
	}
	respondJSON(w, http.StatusOK, nil, map[string]interface{}{
		"message":      fmt.Sprintf("Rollback completed to week %d", targetWeek),
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
// @Failure      400  {object}  ProblemDetails
// @Failure      404  {object}  ProblemDetails
// @Router       /teams/{id}/metrics [get]
func (h *LeagueHandler) GetTeamMetrics(w http.ResponseWriter, r *http.Request) {
	teamID, err := parsePathInt(r, "id", "Team ID")
	if err != nil {
		WriteProblem(w, r, http.StatusBadRequest, "Invalid Team ID", err.Error()+".", "https://api.insiderfootball.com/errors/invalid-id")
		return
	}

	metrics, err := h.service.GetTeamMetrics(r.Context(), teamID)
	if err != nil {
		WriteProblem(w, r, http.StatusNotFound, "Team Not Found", err.Error(), "https://api.insiderfootball.com/errors/not-found")
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
// @Failure      500  {object}  ProblemDetails
// @Router       /league/reset [post]
func (h *LeagueHandler) Reset(w http.ResponseWriter, r *http.Request) {
	if err := h.service.Reset(r.Context()); err != nil {
		WriteProblem(w, r, http.StatusInternalServerError, "Reset Failed", err.Error(), "https://api.insiderfootball.com/errors/reset-failed")
		return
	}
	respondJSON(w, http.StatusOK, nil, map[string]string{
		"message": "League reset successfully",
	})
}

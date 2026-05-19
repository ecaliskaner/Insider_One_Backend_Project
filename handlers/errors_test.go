package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteProblem_IncludesStableErrorCode(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/matches/nope", nil)
	rec := httptest.NewRecorder()

	WriteProblem(rec, req, http.StatusBadRequest, "Invalid Match ID", "Match ID must be a positive integer.", "https://api.insiderfootball.com/errors/invalid-id")

	var problem ProblemDetails
	err := json.Unmarshal(rec.Body.Bytes(), &problem)
	require.NoError(t, err)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, "INVALID_ID", problem.Code)
	assert.Equal(t, "/api/v1/matches/nope", problem.Instance)
}

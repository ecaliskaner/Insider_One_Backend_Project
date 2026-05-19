package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ecaliskaner/Insider_One_Backend_Project/database"
	"github.com/ecaliskaner/Insider_One_Backend_Project/handlers"
	"github.com/ecaliskaner/Insider_One_Backend_Project/router"
	"github.com/ecaliskaner/Insider_One_Backend_Project/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestServer() *httptest.Server {
	// Initialize an in-memory SQLite database
	db, err := database.NewDB(":memory:")
	if err != nil {
		panic(err)
	}

	if err := db.RunMigrations(); err != nil {
		panic(err)
	}

	engine := services.NewMatchEngine()
	weather := services.NewWeatherAdapter()
	leagueService := services.NewLeagueService(db, engine, weather)
	handler := handlers.NewLeagueHandler(leagueService)

	r := router.NewRouter(handler, db)
	return httptest.NewServer(r)
}

func doRequest(t *testing.T, client *http.Client, method, url string, body interface{}) map[string]interface{} {
	respData, statusCode := doRequestWithStatus(t, client, method, url, body)
	require.Equal(t, http.StatusOK, statusCode, "Response was not successful: %v", respData)
	assert.True(t, respData["success"].(bool), "Response was not successful: %v", respData)
	return respData
}

func doRequestWithStatus(t *testing.T, client *http.Client, method, url string, body interface{}) (map[string]interface{}, int) {
	t.Helper()

	var reqBody []byte
	var err error
	if body != nil {
		reqBody, err = json.Marshal(body)
		require.NoError(t, err)
	}

	req, err := http.NewRequest(method, url, bytes.NewBuffer(reqBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	bodyBytes := new(bytes.Buffer)
	bodyBytes.ReadFrom(resp.Body)
	// fmt.Println("Response Body:", bodyBytes.String())

	var respData map[string]interface{}
	err = json.Unmarshal(bodyBytes.Bytes(), &respData)
	require.NoError(t, err, "Failed to decode response: %s", bodyBytes.String())
	return respData, resp.StatusCode
}

func doRawRequestWithStatus(t *testing.T, client *http.Client, method, url, contentType, body string) (map[string]interface{}, int) {
	t.Helper()

	req, err := http.NewRequest(method, url, bytes.NewBufferString(body))
	require.NoError(t, err)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	bodyBytes := new(bytes.Buffer)
	_, err = bodyBytes.ReadFrom(resp.Body)
	require.NoError(t, err)

	var respData map[string]interface{}
	err = json.Unmarshal(bodyBytes.Bytes(), &respData)
	require.NoError(t, err, "Failed to decode response: %s", bodyBytes.String())
	return respData, resp.StatusCode
}

func resetLeague(t *testing.T, client *http.Client, baseURL string) {
	t.Helper()
	doRequest(t, client, "POST", baseURL+"/api/v1/league/reset", nil)
}

func playWeeks(t *testing.T, client *http.Client, baseURL string, weeks int) {
	t.Helper()
	for week := 1; week <= weeks; week++ {
		resp := doRequest(t, client, "POST", baseURL+"/api/v1/league/next-week", nil)
		meta := resp["meta"].(map[string]interface{})
		assert.Equal(t, float64(week+1), meta["current_week"])
	}
}

func TestE2E_LeagueFlow(t *testing.T) {
	ts := setupTestServer()
	defer ts.Close()
	client := ts.Client()

	resetLeague(t, client, ts.URL)

	playWeeks(t, client, ts.URL, 4)

	resp := doRequest(t, client, "GET", ts.URL+"/api/v1/simulation/championship-probabilities", nil)
	meta := resp["meta"].(map[string]interface{})
	assert.Equal(t, float64(1000), meta["simulation_count"])
	data := resp["data"].([]interface{})
	assert.Len(t, data, 4) // 4 teams

	editBody := map[string]int{"home_score": 5, "away_score": 0}
	doRequest(t, client, "PUT", ts.URL+"/api/v1/matches/1", editBody)

	resp = doRequest(t, client, "POST", ts.URL+"/api/v1/league/rollback/2", nil)
	meta = resp["meta"].(map[string]interface{})
	assert.Equal(t, float64(2), meta["current_week"])

	doRequest(t, client, "POST", ts.URL+"/api/v1/league/play-all", nil)

	resp = doRequest(t, client, "GET", ts.URL+"/api/v1/league/table", nil)
	meta = resp["meta"].(map[string]interface{})
	assert.Equal(t, float64(7), meta["current_week"]) // Next week after 6 is 7
}

func TestHealthAndReadinessEndpoints(t *testing.T) {
	ts := setupTestServer()
	defer ts.Close()
	client := ts.Client()

	resp := doRequest(t, client, "GET", ts.URL+"/healthz", nil)
	assert.Equal(t, "ok", resp["data"].(map[string]interface{})["status"])

	resp = doRequest(t, client, "GET", ts.URL+"/readyz", nil)
	assert.Equal(t, "ready", resp["data"].(map[string]interface{})["status"])
}

func TestRequestIDHeader_IsReturned(t *testing.T) {
	ts := setupTestServer()
	defer ts.Close()
	client := ts.Client()

	req, err := http.NewRequest(http.MethodGet, ts.URL+"/healthz", nil)
	require.NoError(t, err)
	req.Header.Set("X-Request-ID", "case-review-123")

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "case-review-123", resp.Header.Get("X-Request-ID"))
}

func TestPlayNextWeek_AllowsFinalWeekAndRejectsAfterSeason(t *testing.T) {
	ts := setupTestServer()
	defer ts.Close()
	client := ts.Client()

	resetLeague(t, client, ts.URL)

	playWeeks(t, client, ts.URL, 6)

	resp, statusCode := doRequestWithStatus(t, client, "POST", ts.URL+"/api/v1/league/next-week", nil)
	assert.Equal(t, http.StatusBadRequest, statusCode)
	assert.Equal(t, "Season Overrun Prevented", resp["title"])
}

func TestLeagueOverview_ReturnsScreenPayloadWithoutPrematurePredictions(t *testing.T) {
	ts := setupTestServer()
	defer ts.Close()
	client := ts.Client()

	resetLeague(t, client, ts.URL)

	resp := doRequest(t, client, "GET", ts.URL+"/api/v1/league/overview", nil)
	data := resp["data"].(map[string]interface{})

	assert.Equal(t, float64(1), data["current_week"])
	assert.Len(t, data["standings"], 4)
	assert.Len(t, data["weeks"], 6)
	assert.NotContains(t, data, "predictions")
}

func TestLeagueOverview_IncludesPredictionsAfterWeekFour(t *testing.T) {
	ts := setupTestServer()
	defer ts.Close()
	client := ts.Client()

	resetLeague(t, client, ts.URL)
	playWeeks(t, client, ts.URL, 4)

	resp := doRequest(t, client, "GET", ts.URL+"/api/v1/league/overview", nil)
	data := resp["data"].(map[string]interface{})

	assert.Equal(t, float64(5), data["current_week"])
	assert.Len(t, data["standings"], 4)
	assert.Len(t, data["weeks"], 6)
	assert.Len(t, data["predictions"], 4)
}

func TestEditMatch_RequiresBothScores(t *testing.T) {
	ts := setupTestServer()
	defer ts.Close()
	client := ts.Client()

	resetLeague(t, client, ts.URL)

	resp, statusCode := doRequestWithStatus(t, client, "PUT", ts.URL+"/api/v1/matches/1", map[string]int{
		"home_score": 2,
	})

	assert.Equal(t, http.StatusBadRequest, statusCode)
	assert.Equal(t, "Invalid Request Body", resp["title"])
	assert.Equal(t, "both home_score and away_score are required.", resp["detail"])
}

func TestEditMatch_RejectsUnknownFields(t *testing.T) {
	ts := setupTestServer()
	defer ts.Close()
	client := ts.Client()

	resetLeague(t, client, ts.URL)

	resp, statusCode := doRequestWithStatus(t, client, "PUT", ts.URL+"/api/v1/matches/1", map[string]int{
		"home_score": 2,
		"away_score": 1,
		"bonus_goal": 1,
	})

	assert.Equal(t, http.StatusBadRequest, statusCode)
	assert.Equal(t, "Invalid Request Body", resp["title"])
}

func TestEditMatch_RejectsNegativeScores(t *testing.T) {
	ts := setupTestServer()
	defer ts.Close()
	client := ts.Client()

	resetLeague(t, client, ts.URL)

	resp, statusCode := doRequestWithStatus(t, client, "PUT", ts.URL+"/api/v1/matches/1", map[string]int{
		"home_score": -1,
		"away_score": 1,
	})

	assert.Equal(t, http.StatusBadRequest, statusCode)
	assert.Equal(t, "Invalid Request Body", resp["title"])
	assert.Equal(t, "scores cannot be negative.", resp["detail"])
}

func TestEditMatch_RejectsMalformedJSON(t *testing.T) {
	ts := setupTestServer()
	defer ts.Close()
	client := ts.Client()

	resetLeague(t, client, ts.URL)
	playWeeks(t, client, ts.URL, 1)

	resp, statusCode := doRawRequestWithStatus(t, client, http.MethodPut, ts.URL+"/api/v1/matches/1", "application/json", `{"home_score":`)

	assert.Equal(t, http.StatusBadRequest, statusCode)
	assert.Equal(t, "Invalid Request Body", resp["title"])
	assert.Contains(t, resp["detail"], "malformed JSON request body")
}

func TestEditMatch_RejectsWrongContentType(t *testing.T) {
	ts := setupTestServer()
	defer ts.Close()
	client := ts.Client()

	resetLeague(t, client, ts.URL)
	playWeeks(t, client, ts.URL, 1)

	resp, statusCode := doRawRequestWithStatus(t, client, http.MethodPut, ts.URL+"/api/v1/matches/1", "text/plain", `{"home_score": 2, "away_score": 1}`)

	assert.Equal(t, http.StatusBadRequest, statusCode)
	assert.Equal(t, "Invalid Request Body", resp["title"])
	assert.Equal(t, "Content-Type must be application/json.", resp["detail"])
}

func TestEditMatch_RejectsScheduledMatch(t *testing.T) {
	ts := setupTestServer()
	defer ts.Close()
	client := ts.Client()

	resetLeague(t, client, ts.URL)

	resp, statusCode := doRequestWithStatus(t, client, "PUT", ts.URL+"/api/v1/matches/1", map[string]int{
		"home_score": 2,
		"away_score": 1,
	})

	assert.Equal(t, http.StatusBadRequest, statusCode)
	assert.Equal(t, "Match Edit Failed", resp["title"])
	assert.Equal(t, "only played matches can be edited", resp["detail"])
}

func TestChampionshipProbabilities_RejectPrematureRequest(t *testing.T) {
	ts := setupTestServer()
	defer ts.Close()
	client := ts.Client()

	resetLeague(t, client, ts.URL)
	playWeeks(t, client, ts.URL, 3)

	resp, statusCode := doRequestWithStatus(t, client, "GET", ts.URL+"/api/v1/simulation/championship-probabilities", nil)

	assert.Equal(t, http.StatusBadRequest, statusCode)
	assert.Equal(t, "Premature Championship Probability Request", resp["title"])
}

func TestRollback_RejectsOutOfRangeWeek(t *testing.T) {
	ts := setupTestServer()
	defer ts.Close()
	client := ts.Client()

	resetLeague(t, client, ts.URL)

	resp, statusCode := doRequestWithStatus(t, client, "POST", ts.URL+"/api/v1/league/rollback/7", nil)

	assert.Equal(t, http.StatusBadRequest, statusCode)
	assert.Equal(t, "Invalid Rollback Target Bounds", resp["title"])
}

func TestRollback_IsIdempotentAndPreservesRebuildConsistency(t *testing.T) {
	ts := setupTestServer()
	defer ts.Close()
	client := ts.Client()

	resetLeague(t, client, ts.URL)
	playWeeks(t, client, ts.URL, 3)

	doRequest(t, client, "POST", ts.URL+"/api/v1/league/rollback/2", nil)
	resp := doRequest(t, client, "POST", ts.URL+"/api/v1/league/rollback/2", nil)
	meta := resp["meta"].(map[string]interface{})
	assert.Equal(t, float64(2), meta["current_week"])

	table := doRequest(t, client, "GET", ts.URL+"/api/v1/league/table", nil)
	standings := table["data"].([]interface{})
	totalPlayed := 0
	for _, row := range standings {
		totalPlayed += int(row.(map[string]interface{})["played"].(float64))
	}
	assert.Equal(t, 4, totalPlayed)
}

func TestTeamMetrics_RejectsInvalidTeamID(t *testing.T) {
	ts := setupTestServer()
	defer ts.Close()
	client := ts.Client()

	resp, statusCode := doRequestWithStatus(t, client, "GET", ts.URL+"/api/v1/teams/not-a-number/metrics", nil)

	assert.Equal(t, http.StatusBadRequest, statusCode)
	assert.Equal(t, "Invalid Team ID", resp["title"])
}

func TestPlayAll_RejectsCompletedSeason(t *testing.T) {
	ts := setupTestServer()
	defer ts.Close()
	client := ts.Client()

	resetLeague(t, client, ts.URL)
	doRequest(t, client, "POST", ts.URL+"/api/v1/league/play-all", nil)

	resp, statusCode := doRequestWithStatus(t, client, "POST", ts.URL+"/api/v1/league/play-all", nil)

	assert.Equal(t, http.StatusBadRequest, statusCode)
	assert.Equal(t, "Simulation Error", resp["title"])
	assert.Equal(t, "all weeks have already been played", resp["detail"])
}

func TestGetMatch_ReturnsNotFoundForMissingMatch(t *testing.T) {
	ts := setupTestServer()
	defer ts.Close()
	client := ts.Client()

	resetLeague(t, client, ts.URL)

	resp, statusCode := doRequestWithStatus(t, client, "GET", ts.URL+"/api/v1/matches/999", nil)

	assert.Equal(t, http.StatusNotFound, statusCode)
	assert.Equal(t, "Match Not Found", resp["title"])
}

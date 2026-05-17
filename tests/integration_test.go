package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/insider/league-simulation/database"
	"github.com/insider/league-simulation/handlers"
	"github.com/insider/league-simulation/router"
	"github.com/insider/league-simulation/services"
	"github.com/stretchr/testify/assert"
)

func setupTestServer() *httptest.Server {
	// Initialize an in-memory SQLite database
	db, err := database.NewDB(":memory:")
	if err != nil {
		panic(err)
	}

	engine := services.NewMatchEngine()
	weather := services.NewWeatherAdapter()
	leagueService := services.NewLeagueService(db, engine, weather)
	handler := handlers.NewLeagueHandler(leagueService)

	r := router.NewRouter(handler)
	return httptest.NewServer(r)
}

func doRequest(t *testing.T, client *http.Client, method, url string, body interface{}) map[string]interface{} {
	var reqBody []byte
	var err error
	if body != nil {
		reqBody, err = json.Marshal(body)
		assert.NoError(t, err)
	}

	req, err := http.NewRequest(method, url, bytes.NewBuffer(reqBody))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	bodyBytes := new(bytes.Buffer)
	bodyBytes.ReadFrom(resp.Body)
	// fmt.Println("Response Body:", bodyBytes.String())

	var respData map[string]interface{}
	err = json.Unmarshal(bodyBytes.Bytes(), &respData)
	assert.NoError(t, err, "Failed to decode response: %s", bodyBytes.String())
	
	if err != nil {
		t.FailNow()
	}

	// Since we standardized API responses, assert "success" is true
	assert.True(t, respData["success"].(bool), "Response was not successful: %v", respData)
	return respData
}

func TestE2E_LeagueFlow(t *testing.T) {
	ts := setupTestServer()
	defer ts.Close()
	client := ts.Client()

	// 1. Reset League
	fmt.Println("Step 1: Resetting league...")
	doRequest(t, client, "POST", ts.URL+"/api/v1/league/reset", nil)

	// 2. Play 4 weeks
	fmt.Println("Step 2: Playing 4 weeks...")
	for i := 1; i <= 4; i++ {
		resp := doRequest(t, client, "POST", ts.URL+"/api/v1/league/next-week", nil)
		meta := resp["meta"].(map[string]interface{})
		assert.Equal(t, float64(i+1), meta["current_week"])
	}

	// 3. Oracle (Predictions)
	fmt.Println("Step 3: Checking Oracle predictions...")
	resp := doRequest(t, client, "GET", ts.URL+"/api/v1/simulation/oracle", nil)
	meta := resp["meta"].(map[string]interface{})
	assert.Equal(t, float64(1000), meta["simulation_count"])
	data := resp["data"].([]interface{})
	assert.Len(t, data, 4) // 4 teams

	// 4. Edit a Match (e.g., Match 1)
	fmt.Println("Step 4: Editing match #1...")
	editBody := map[string]int{"home_score": 5, "away_score": 0}
	doRequest(t, client, "PUT", ts.URL+"/api/v1/matches/1", editBody)

	// 5. Rollback to Week 2
	fmt.Println("Step 5: Rolling back to week 2...")
	resp = doRequest(t, client, "POST", ts.URL+"/api/v1/league/rollback/2", nil)
	meta = resp["meta"].(map[string]interface{})
	assert.Equal(t, float64(2), meta["current_week"])

	// 6. Play All remaining weeks
	fmt.Println("Step 6: Playing all remaining weeks...")
	doRequest(t, client, "POST", ts.URL+"/api/v1/league/play-all", nil)

	// Verify season is over
	resp = doRequest(t, client, "GET", ts.URL+"/api/v1/league/table", nil)
	meta = resp["meta"].(map[string]interface{})
	assert.Equal(t, float64(7), meta["current_week"]) // Next week after 6 is 7

	fmt.Println("E2E Test completed successfully.")
}

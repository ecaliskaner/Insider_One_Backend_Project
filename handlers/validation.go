package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
)

const maxJSONBodyBytes = 1 << 20

func parsePathInt(r *http.Request, name, label string) (int, error) {
	raw := mux.Vars(r)[name]
	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 {
		return 0, fmt.Errorf("%s must be a positive integer", label)
	}
	return value, nil
}

func parseBoundedPathInt(r *http.Request, name, label string, min, max int) (int, error) {
	raw := mux.Vars(r)[name]
	value, err := strconv.Atoi(raw)
	if err != nil || value < min || value > max {
		return 0, fmt.Errorf("%s must be an integer between %d and %d", label, min, max)
	}
	return value, nil
}

func decodeStrictJSON(w http.ResponseWriter, r *http.Request, dst interface{}) error {
	contentType := strings.TrimSpace(r.Header.Get("Content-Type"))
	if contentType != "" {
		mediaType, _, err := mime.ParseMediaType(contentType)
		if err != nil || mediaType != "application/json" {
			return errors.New("Content-Type must be application/json")
		}
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		if errors.Is(err, io.EOF) {
			return errors.New("request body must contain a JSON object")
		}
		return fmt.Errorf("malformed JSON request body: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("request body must contain a single JSON object")
	}
	return nil
}

func validateEditMatchRequest(req EditMatchRequest) error {
	if req.HomeScore == nil || req.AwayScore == nil {
		return errors.New("both home_score and away_score are required")
	}
	if *req.HomeScore < 0 || *req.AwayScore < 0 {
		return errors.New("scores cannot be negative")
	}
	if *req.HomeScore > 30 || *req.AwayScore > 30 {
		return errors.New("scores must be realistic football scores between 0 and 30")
	}
	return nil
}

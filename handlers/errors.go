package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"path"
	"strings"
)

// ProblemDetails satisfies RFC 7807 specifications for API error responses
type ProblemDetails struct {
	Type     string `json:"type"`     // URI reference that identifies the problem type
	Title    string `json:"title"`    // Short, human-readable summary
	Status   int    `json:"status"`   // HTTP status code
	Detail   string `json:"detail"`   // Human-readable explanation specific to this occurrence
	Instance string `json:"instance"` // URI reference that identifies the specific occurrence
	Code     string `json:"code"`     // Stable machine-readable error code
}

// WriteProblem converts and streams the error payload back to the client as application/problem+json
func WriteProblem(w http.ResponseWriter, r *http.Request, status int, title, detail, errorType string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)

	prob := ProblemDetails{
		Type:     errorType,
		Title:    title,
		Status:   status,
		Detail:   detail,
		Instance: r.URL.Path,
		Code:     problemCode(errorType),
	}

	// Structured logging for external telemetry/observability
	slog.Error("API Exception Intercepted",
		slog.Int("status", status),
		slog.String("title", title),
		slog.String("detail", detail),
		slog.String("code", prob.Code),
		slog.String("path", r.URL.Path),
		slog.String("method", r.Method),
	)

	if err := json.NewEncoder(w).Encode(prob); err != nil {
		slog.Error("failed to write problem response", slog.Any("error", err))
	}
}

func problemCode(errorType string) string {
	slug := strings.Trim(path.Base(errorType), "/")
	if slug == "." || slug == "" {
		return "UNKNOWN_ERROR"
	}
	return strings.ToUpper(strings.ReplaceAll(slug, "-", "_"))
}

package httputil

import (
	"encoding/json"
	"net/http"

	"simplefrp/internal/clock"
)

type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func WriteData(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"data": data})
}

func WriteError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": ErrorBody{Code: code, Message: message},
	})
}

func StampMap(extra map[string]any) map[string]any {
	if extra == nil {
		extra = map[string]any{}
	}
	extra["time"] = clock.Stamp()
	return extra
}

func NotFound(w http.ResponseWriter, _ *http.Request) {
	WriteError(w, http.StatusNotFound, "not_found", "endpoint not found")
}

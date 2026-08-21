package health

import (
	"encoding/json"
	"net/http"
	"time"
)

type Check func() (string, bool)

type Handler struct {
	Role   string
	Checks []Check
}

func (h Handler) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	ok := true
	details := map[string]string{}
	for _, c := range h.Checks {
		name, live := c()
		if live {
			details[name] = "ok"
		} else {
			details[name] = "down"
			ok = false
		}
	}
	status := "ok"
	code := http.StatusOK
	if !ok {
		status = "degraded"
		code = http.StatusServiceUnavailable
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status": status,
		"role":   h.Role,
		"ts":     time.Now().Format(time.RFC3339),
		"checks": details,
	})
}

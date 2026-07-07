package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
)

func jsonPayloadObject(v interface{}) map[string]interface{} {
	raw, err := json.Marshal(v)
	if err != nil {
		return map[string]interface{}{}
	}

	var out map[string]interface{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return map[string]interface{}{}
	}
	return out
}

func parsePositiveIntQuery(r *http.Request, key string, fallback int) int {
	value := r.URL.Query().Get(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

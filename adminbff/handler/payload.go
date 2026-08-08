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

func (h *Handler) candidateAwarePayload(v interface{}, candidateKeys ...string) map[string]interface{} {
	payload := jsonPayloadObject(v)
	h.attachCandidateNames(payload, candidateKeys...)
	return payload
}

func (h *Handler) attachCandidateNames(value interface{}, candidateKeys ...string) {
	switch typed := value.(type) {
	case map[string]interface{}:
		for _, key := range candidateKeys {
			raw, ok := typed[key]
			if !ok {
				continue
			}
			candidateULID, ok := raw.(string)
			if !ok || candidateULID == "" {
				continue
			}
			h.attachCandidateName(typed, candidateULID)
			break
		}
		for _, child := range typed {
			h.attachCandidateNames(child, candidateKeys...)
		}
	case []interface{}:
		for _, child := range typed {
			h.attachCandidateNames(child, candidateKeys...)
		}
	}
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

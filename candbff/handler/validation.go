package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

func requireRequestField(w http.ResponseWriter, value string, name string) bool {
	if strings.TrimSpace(value) != "" {
		return true
	}
	WriteError(w, http.StatusBadRequest, ErrInvalidRequest, fmt.Sprintf("%s is required", name))
	return false
}

func requireRequestFields(w http.ResponseWriter, fields ...string) bool {
	if len(fields)%2 != 0 {
		WriteError(w, http.StatusInternalServerError, ErrInternal, "invalid validation configuration")
		return false
	}
	for i := 0; i < len(fields); i += 2 {
		if !requireRequestField(w, fields[i], fields[i+1]) {
			return false
		}
	}
	return true
}

// parsePositiveIntQuery parses a positive integer query param, returning fallback if absent or invalid.
func parsePositiveIntQuery(r *http.Request, key string, fallback int) int {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

// parseNonNegativeIntQuery parses a non-negative integer query param, returning fallback if absent or invalid.
func parseNonNegativeIntQuery(r *http.Request, key string, fallback int) int {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 0 {
		return fallback
	}
	return value
}

// totalPages calculates the total number of pages given a total count and page size.
func totalPages(total int, pageSize int) int {
	if total <= 0 || pageSize <= 0 {
		return 0
	}
	return (total + pageSize - 1) / pageSize
}

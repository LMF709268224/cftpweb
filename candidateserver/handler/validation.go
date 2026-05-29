package handler

import (
	"fmt"
	"net/http"
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

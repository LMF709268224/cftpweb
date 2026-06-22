package handler

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

func requireRequestField(w http.ResponseWriter, value, name string) bool {
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

func requiredURLParam(w http.ResponseWriter, r *http.Request, name string) (string, bool) {
	value := strings.TrimSpace(chi.URLParam(r, name))
	if value == "" {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, fmt.Sprintf("%s is required", name))
		return "", false
	}
	return value, true
}

func requirePositiveVersion(w http.ResponseWriter, version uint32) bool {
	if version > 0 {
		return true
	}
	WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "version must be greater than 0")
	return false
}

type reorderItem interface {
	GetEntityUlid() string
	GetVersion() uint32
}

func requireReorderItems[T reorderItem](w http.ResponseWriter, items []T) bool {
	if len(items) == 0 {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "items is required")
		return false
	}
	for i, item := range items {
		if strings.TrimSpace(item.GetEntityUlid()) == "" {
			WriteError(w, http.StatusBadRequest, ErrInvalidRequest, fmt.Sprintf("items[%d].entity_ulid is required", i))
			return false
		}
		if item.GetVersion() == 0 {
			WriteError(w, http.StatusBadRequest, ErrInvalidRequest, fmt.Sprintf("items[%d].version must be greater than 0", i))
			return false
		}
	}
	return true
}

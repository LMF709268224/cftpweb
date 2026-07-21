package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"unicode/utf8"
)

const (
	maxTelemetryEvents          = 100
	maxTelemetryEventNameRunes  = 128
	maxTelemetryTimestampRunes  = 128
	maxTelemetryURLRunes        = 2048
	maxTelemetryPayloadJSONSize = 16 << 10
)

// TelemetryEvent represents a single telemetry event sent from the frontend.
type TelemetryEvent struct {
	EventName string                 `json:"event_name"`
	Payload   map[string]interface{} `json:"payload,omitempty"`
	Timestamp string                 `json:"timestamp,omitempty"`
	URL       string                 `json:"url,omitempty"`
}

// TelemetryBatch represents a batch of telemetry events.
type TelemetryBatch struct {
	Events []TelemetryEvent `json:"events"`
}

// ReportTelemetry processes incoming telemetry events from the frontend.
func (h *Handler) ReportTelemetry(w http.ResponseWriter, r *http.Request) {
	var batch TelemetryBatch
	if err := ReadJSON(r, &batch); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			WriteError(w, http.StatusRequestEntityTooLarge, ErrInvalidRequest, "telemetry request body is too large")
			return
		}
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid telemetry request body")
		return
	}
	if err := validateTelemetryBatch(&batch); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, err.Error())
		return
	}

	// Extract context variables
	candidateID := ""
	if id, ok := r.Context().Value(CtxKeyCandidateID).(string); ok {
		candidateID = id
	}

	userAgent := truncateForLog(r.UserAgent(), 512)
	clientIP := truncateForLog(r.RemoteAddr, 256)
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		clientIP = truncateForLog(forwarded, 256)
	}

	for _, event := range batch.Events {
		// Prepare base attributes for the log
		attrs := []slog.Attr{
			slog.String("type", "telemetry"),
			slog.String("event_name", event.EventName),
			slog.String("user_agent", userAgent),
			slog.String("client_ip", clientIP),
		}

		if candidateID != "" {
			attrs = append(attrs, slog.String("candidate_id", candidateID))
		}
		if event.URL != "" {
			attrs = append(attrs, slog.String("url", event.URL))
		}
		if event.Timestamp != "" {
			attrs = append(attrs, slog.String("client_time", event.Timestamp))
		}

		// If there is a payload, flatten it into the log to make it easily searchable by Loki
		if len(event.Payload) > 0 {
			// Convert payload to a JSON string or attach as an Any attribute.
			// Loki can parse nested JSON, so we can attach it as Any.
			attrs = append(attrs, slog.Any("payload", event.Payload))
		}

		// Log the event as structured JSON. We use a custom level or just Info.
		slog.LogAttrs(r.Context(), slog.LevelInfo, "Frontend telemetry event", attrs...)
	}

	// Acknowledge receipt
	WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func validateTelemetryBatch(batch *TelemetryBatch) error {
	if batch == nil {
		return fmt.Errorf("telemetry batch is required")
	}
	if len(batch.Events) > maxTelemetryEvents {
		return fmt.Errorf("telemetry batch cannot contain more than %d events", maxTelemetryEvents)
	}

	for i := range batch.Events {
		event := &batch.Events[i]
		event.EventName = strings.TrimSpace(event.EventName)
		event.Timestamp = strings.TrimSpace(event.Timestamp)
		event.URL = strings.TrimSpace(event.URL)

		if event.EventName == "" {
			return fmt.Errorf("events[%d].event_name is required", i)
		}
		if utf8.RuneCountInString(event.EventName) > maxTelemetryEventNameRunes {
			return fmt.Errorf("events[%d].event_name is too long", i)
		}
		if utf8.RuneCountInString(event.Timestamp) > maxTelemetryTimestampRunes {
			return fmt.Errorf("events[%d].timestamp is too long", i)
		}
		if utf8.RuneCountInString(event.URL) > maxTelemetryURLRunes {
			return fmt.Errorf("events[%d].url is too long", i)
		}
		if len(event.Payload) == 0 {
			continue
		}
		payloadJSON, err := json.Marshal(event.Payload)
		if err != nil {
			return fmt.Errorf("events[%d].payload is invalid", i)
		}
		if len(payloadJSON) > maxTelemetryPayloadJSONSize {
			return fmt.Errorf("events[%d].payload is too large", i)
		}
	}
	return nil
}

package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
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
	if err := json.NewDecoder(r.Body).Decode(&batch); err != nil {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}

	// Extract context variables
	candidateID := ""
	if id, ok := r.Context().Value(CtxKeyCandidateID).(string); ok {
		candidateID = id
	}

	userAgent := r.UserAgent()
	clientIP := r.RemoteAddr
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		clientIP = forwarded
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

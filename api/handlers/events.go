package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"streampulse/api/kafka"
)

// Event is the shape accepted by POST /events.
type Event struct {
	EventType string                 `json:"event_type"`
	UserID    string                 `json:"user_id"`
	Source    string                 `json:"source"`
	Timestamp string                 `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata"`
}

func (e Event) Validate() []string {
	var problems []string
	if e.EventType == "" {
		problems = append(problems, "event_type is required")
	}
	if e.UserID == "" {
		problems = append(problems, "user_id is required")
	}
	if e.Timestamp == "" {
		problems = append(problems, "timestamp is required")
	} else if _, err := time.Parse(time.RFC3339, e.Timestamp); err != nil {
		problems = append(problems, "timestamp must be RFC3339")
	}
	return problems
}

type EventHandler struct {
	producer *kafka.Producer
}

func NewEventHandler(p *kafka.Producer) *EventHandler {
	return &EventHandler{producer: p}
}

func (h *EventHandler) Ingest(w http.ResponseWriter, r *http.Request) {
	var evt Event
	if err := json.NewDecoder(r.Body).Decode(&evt); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}

	if problems := evt.Validate(); len(problems) > 0 {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]interface{}{"errors": problems})
		return
	}

	payload, err := json.Marshal(evt)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to encode event"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	if err := h.producer.Publish(ctx, evt.UserID, payload); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "failed to publish event"})
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]string{"status": "accepted"})
}

func (h *EventHandler) Ready(w http.ResponseWriter, r *http.Request) {
	// In production this would check the Kafka connection health.
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func writeJSON(w http.ResponseWriter, status int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

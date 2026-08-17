package eyedux

import "time"

// Event represents an event returned by the Eyedux API.
type Event struct {
	ID            string         `json:"id"`
	Type          string         `json:"type"`
	Properties    map[string]any `json:"properties"`
	Status        string         `json:"status"`
	Timestamp     time.Time      `json:"timestamp"`
	CreatedAt     time.Time      `json:"created_at"`
	ExternalID    *string        `json:"external_id"`
	CorrelationID *string        `json:"correlation_id"`
	Metadata      map[string]any `json:"metadata"`
}

// CreateEventInput holds the parameters for creating a new event.
type CreateEventInput struct {
	ProjectID     string
	Type          string
	Properties    map[string]any
	ExternalID    *string
	CorrelationID *string
	Metadata      map[string]any
}

// ListEventsInput holds the optional filters for listing events.
type ListEventsInput struct {
	Type          *string
	CorrelationID *string
}

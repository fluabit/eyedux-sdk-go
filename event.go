package eyeduxsdk

import "time"

// EventObject is a reference to an external or correlated entity.
type EventObject struct {
	ID       string  `json:"id"`
	Property string  `json:"property"`
	Source   *string `json:"source,omitempty"`
}

// Event represents an event returned by the Eyedux API.
type Event struct {
	ID                string         `json:"id"`
	Type              string         `json:"type"`
	TypeGroup         string         `json:"type_group"`
	Properties        map[string]any `json:"properties"`
	Status            string         `json:"status"`
	Timestamp         time.Time      `json:"timestamp"`
	CreatedAt         time.Time      `json:"created_at"`
	ExternalObject    *EventObject   `json:"external_object"`
	CorrelationObject *EventObject   `json:"correlation_object"`
	Metadata          map[string]any `json:"metadata"`
}

// CreateEventInput holds the parameters for creating a new event.
type CreateEventInput struct {
	ProjectID         string
	Type              string
	TypeGroup         string
	Properties        map[string]any
	ExternalObject    *EventObject
	CorrelationObject *EventObject
	Metadata          map[string]any
}

// ListEventsInput holds the optional filters for listing events.
type ListEventsInput struct {
	Type          *string
	CorrelationID *string // filters by correlation_object.id
}

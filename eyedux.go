// Package eyedux is the official Go SDK for the Eyedux event ingestion platform.
package eyedux

import (
	"context"
	"net/http"
	"net/url"
	"time"
)

const defaultBaseURL = "https://api.eyedux.com"

// Client communicates with the Eyedux API.
// Create one with New; do not copy after first use.
type Client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

type config struct {
	timeout    time.Duration
	httpClient *http.Client
}

// Option configures a Client at construction time.
type Option func(*config)

// WithHTTPClient replaces the default HTTP client.
// When provided, WithTimeout has no effect.
func WithHTTPClient(client *http.Client) Option {
	return func(c *config) { c.httpClient = client }
}

// WithTimeout sets the timeout on the default HTTP client.
// Ignored when WithHTTPClient is also provided.
func WithTimeout(d time.Duration) Option {
	return func(c *config) { c.timeout = d }
}

// New creates a Client authenticated with apiKey.
// Returns ErrEmptyAPIKey if apiKey is empty.
func New(apiKey string, opts ...Option) (*Client, error) {
	if apiKey == "" {
		return nil, ErrEmptyAPIKey
	}

	cfg := &config{timeout: 30 * time.Second}
	for _, opt := range opts {
		opt(cfg)
	}

	httpClient := cfg.httpClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: cfg.timeout}
	}

	return &Client{
		apiKey:     apiKey,
		baseURL:    defaultBaseURL,
		httpClient: httpClient,
	}, nil
}

// createEventBody is the JSON payload for POST /public/logs.
type createEventBody struct {
	ProjectID     string         `json:"project_id"`
	Type          string         `json:"type"`
	Properties    map[string]any `json:"properties"`
	ExternalID    *string        `json:"external_id,omitempty"`
	CorrelationID *string        `json:"correlation_id,omitempty"`
	Metadata      map[string]any `json:"metadata,omitempty"`
}

// CreateEvent ingests a new event into the authenticated organization.
func (c *Client) CreateEvent(ctx context.Context, input CreateEventInput) (*Event, error) {
	body := createEventBody{
		ProjectID:     input.ProjectID,
		Type:          input.Type,
		Properties:    input.Properties,
		ExternalID:    input.ExternalID,
		CorrelationID: input.CorrelationID,
		Metadata:      input.Metadata,
	}

	var env successEnvelope[*Event]
	if err := c.do(ctx, http.MethodPost, "/public/logs", body, &env); err != nil {
		return nil, err
	}

	return env.Data, nil
}

// ListEvents retrieves events for the authenticated organization.
// Filters in input are applied cumulatively when both are set.
func (c *Client) ListEvents(ctx context.Context, input ListEventsInput) ([]Event, error) {
	params := url.Values{}
	if input.Type != nil {
		params.Set("type", *input.Type)
	}
	if input.CorrelationID != nil {
		params.Set("correlation_id", *input.CorrelationID)
	}

	path := "/public/logs"
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	var env successEnvelope[[]Event]
	if err := c.do(ctx, http.MethodGet, path, nil, &env); err != nil {
		return nil, err
	}

	if env.Data == nil {
		return []Event{}, nil
	}
	return env.Data, nil
}

// FindEventByExternalID retrieves a single event by its external ID.
// Returns ErrEmptyExternalID if externalID is empty.
func (c *Client) FindEventByExternalID(ctx context.Context, externalID string) (*Event, error) {
	if externalID == "" {
		return nil, ErrEmptyExternalID
	}

	path := "/public/logs/external/" + url.PathEscape(externalID)

	var env successEnvelope[*Event]
	if err := c.do(ctx, http.MethodGet, path, nil, &env); err != nil {
		return nil, err
	}

	return env.Data, nil
}

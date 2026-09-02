// Package eyeduxsdk is the official Go SDK for the Eyedux event ingestion platform.
package eyeduxsdk

import (
	"context"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const defaultBaseURL = "https://api.eyedux.com"

// Client communicates with the Eyedux API.
// Create one with New; do not copy after first use.
type Client struct {
	apiKey          string
	baseURL         string
	httpClient      *http.Client
	projectID       string
	defaultMetadata map[string]any
}

type config struct {
	timeout         time.Duration
	httpClient      *http.Client
	projectID       string
	defaultMetadata map[string]any
}

// Config contains the required and optional settings for a Client.
type Config struct {
	APIKey          string
	ProjectID       string
	HTTPClient      *http.Client
	Timeout         time.Duration
	DefaultMetadata map[string]any
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

// WithProjectID sets the default project used when an event does not specify one.
func WithProjectID(projectID string) Option {
	return func(c *config) { c.projectID = strings.TrimSpace(projectID) }
}

// WithDefaultMetadata adds metadata to every event created by the Client.
func WithDefaultMetadata(metadata map[string]any) Option {
	return func(c *config) { c.defaultMetadata = cloneMap(metadata) }
}

// New creates a Client authenticated with apiKey.
// Returns ErrEmptyAPIKey if apiKey is empty.
func New(apiKey string, opts ...Option) (*Client, error) {
	apiKey = strings.TrimSpace(apiKey)
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
		apiKey:          apiKey,
		baseURL:         defaultBaseURL,
		httpClient:      httpClient,
		projectID:       cfg.projectID,
		defaultMetadata: cloneMap(cfg.defaultMetadata),
	}, nil
}

// NewWithConfig creates a Client from explicit configuration.
// APIKey and ProjectID are required.
func NewWithConfig(cfg Config) (*Client, error) {
	if strings.TrimSpace(cfg.APIKey) == "" {
		return nil, ErrEmptyAPIKey
	}
	if strings.TrimSpace(cfg.ProjectID) == "" {
		return nil, ErrEmptyProjectID
	}

	opts := []Option{WithProjectID(cfg.ProjectID)}
	if cfg.HTTPClient != nil {
		opts = append(opts, WithHTTPClient(cfg.HTTPClient))
	}
	if cfg.Timeout != 0 {
		opts = append(opts, WithTimeout(cfg.Timeout))
	}
	if cfg.DefaultMetadata != nil {
		opts = append(opts, WithDefaultMetadata(cfg.DefaultMetadata))
	}

	return New(cfg.APIKey, opts...)
}

// NewFromEnv creates a Client using EYEDUX_API_KEY and explicit options.
// ProjectID must be supplied with WithProjectID or per event.
func NewFromEnv(opts ...Option) (*Client, error) {
	return New(strings.TrimSpace(os.Getenv("EYEDUX_API_KEY")), opts...)
}

// createEventBody is the JSON payload for POST /public/logs.
type createEventBody struct {
	ProjectID         string          `json:"project_id"`
	Type              string          `json:"type"`
	TypeGroup         string          `json:"type_group,omitempty"`
	EyeduxType        EventEyeduxType `json:"eyedux_type,omitempty"`
	Properties        map[string]any  `json:"properties"`
	ExternalObject    *EventObject    `json:"external_object,omitempty"`
	CorrelationObject *EventObject    `json:"correlation_object,omitempty"`
	Metadata          map[string]any  `json:"metadata,omitempty"`
}

// CreateEvent ingests a new event into the authenticated organization.
func (c *Client) CreateEvent(ctx context.Context, input CreateEventInput) (*Event, error) {
	if strings.TrimSpace(input.ProjectID) == "" {
		input.ProjectID = c.projectID
	}
	if strings.TrimSpace(input.ProjectID) == "" {
		return nil, ErrEmptyProjectID
	}
	input.ProjectID = strings.TrimSpace(input.ProjectID)
	input.Metadata = mergeMaps(c.defaultMetadata, input.Metadata)
	body := createEventBody(input)

	var env successEnvelope[*Event]
	if err := c.do(ctx, http.MethodPost, "/public/logs", body, &env); err != nil {
		return nil, err
	}

	return env.Data, nil
}

func cloneMap(values map[string]any) map[string]any {
	if values == nil {
		return nil
	}

	clone := make(map[string]any, len(values))
	for key, value := range values {
		clone[key] = value
	}
	return clone
}

func mergeMaps(defaults, values map[string]any) map[string]any {
	if len(defaults) == 0 {
		return cloneMap(values)
	}

	merged := cloneMap(defaults)
	for key, value := range values {
		merged[key] = value
	}
	return merged
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

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
	apiKey            string
	baseURL           string
	httpClient        *http.Client
	projectID         string
	defaultMetadata   map[string]any
	automaticMetadata bool
	metadataProvider  func() map[string]any
}

type config struct {
	timeout           time.Duration
	httpClient        *http.Client
	projectID         string
	defaultMetadata   map[string]any
	automaticMetadata bool
}

// Config contains the required and optional settings for a Client.
type Config struct {
	APIKey                   string
	ProjectID                string
	HTTPClient               *http.Client
	Timeout                  time.Duration
	DefaultMetadata          map[string]any
	DisableAutomaticMetadata bool
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

// WithAutomaticMetadata controls collection of runtime metadata for system,
// audit, and metric events. It is enabled by default.
func WithAutomaticMetadata(enabled bool) Option {
	return func(c *config) { c.automaticMetadata = enabled }
}

// New creates a Client authenticated with apiKey.
// Returns ErrEmptyAPIKey if apiKey is empty.
func New(apiKey string, opts ...Option) (*Client, error) {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return nil, ErrEmptyAPIKey
	}

	cfg := &config{timeout: 30 * time.Second, automaticMetadata: true}
	for _, opt := range opts {
		opt(cfg)
	}

	httpClient := cfg.httpClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: cfg.timeout}
	}

	return &Client{
		apiKey:            apiKey,
		baseURL:           defaultBaseURL,
		httpClient:        httpClient,
		projectID:         cfg.projectID,
		defaultMetadata:   cloneMap(cfg.defaultMetadata),
		automaticMetadata: cfg.automaticMetadata,
		metadataProvider:  collectRuntimeMetadata,
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
	if cfg.DisableAutomaticMetadata {
		opts = append(opts, WithAutomaticMetadata(false))
	}

	return New(cfg.APIKey, opts...)
}

// NewFromEnv creates a Client using EYEDUX_API_KEY and explicit options.
// ProjectID must be supplied with WithProjectID or per event.
func NewFromEnv(opts ...Option) (*Client, error) {
	return New(strings.TrimSpace(os.Getenv("EYEDUX_API_KEY")), opts...)
}

// CreateEvent ingests a new event into the authenticated organization.
func (c *Client) CreateEvent(ctx context.Context, input CreateEventInput) (*Event, error) {
	// Se o chamador não informou um ProjectID (ou só passou espaços), usa o
	// ProjectID padrão configurado no client.
	if strings.TrimSpace(input.ProjectID) == "" {
		input.ProjectID = c.projectID
	}
	// Se mesmo após o fallback o ProjectID continuar vazio, não há como
	// prosseguir: retorna erro específico de ProjectID ausente.
	if strings.TrimSpace(input.ProjectID) == "" {
		return nil, ErrEmptyProjectID
	}
	// Normaliza o ProjectID removendo espaços em branco nas extremidades
	// antes de usá-lo na requisição.
	input.ProjectID = strings.TrimSpace(input.ProjectID)
	// Parte dos metadados padrão configurados no client (pode ser nil).
	defaultMetadata := c.defaultMetadata
	// Quando a coleta automática de metadados está habilitada, o tipo do
	// evento é um dos que exigem metadados de runtime, e existe um
	// provider configurado, mescla os metadados retornados pelo provider
	// com os metadados padrão (o provider tem prioridade).
	if c.automaticMetadata && isRuntimeMetadataEvent(input.EyeduxType) && c.metadataProvider != nil {
		defaultMetadata = mergeMaps(c.metadataProvider(), defaultMetadata)
	}
	// Mescla os metadados padrão (já combinados com os de runtime, se houver)
	// com os metadados explicitamente informados pelo chamador, dando
	// prioridade aos valores de input.Metadata.
	input.Metadata = mergeMaps(defaultMetadata, input.Metadata)
	body := createEventBody(input)
	if body.Properties == nil {
		body.Properties = map[string]any{}
	}

	// Envelope genérico usado para decodificar a resposta de sucesso da API,
	// cujo campo Data é um ponteiro para o Event criado.
	var envelope successEnvelope[*Event]
	// Executa a chamada POST para o endpoint de criação de eventos.
	// Em caso de erro (rede, status não-2xx, etc.), retorna imediatamente.
	if err := c.do(ctx, http.MethodPost, "/public/logs", body, &envelope); err != nil {
		return nil, err
	}

	// Retorna o evento criado, extraído do envelope de resposta.
	return envelope.Data, nil
}

// createEventBody is the JSON payload for POST /public/logs.
type createEventBody struct {
	ProjectID         string          `json:"project_id"`
	Type              string          `json:"type"`
	TypeGroup         string          `json:"type_group,omitempty"`
	Message           string          `json:"message,omitempty"`
	EyeduxType        EventEyeduxType `json:"eyedux_type,omitempty"`
	Properties        map[string]any  `json:"properties"`
	ExternalObject    *EventObject    `json:"external_object,omitempty"`
	CorrelationObject *EventObject    `json:"correlation_object,omitempty"`
	Metadata          map[string]any  `json:"metadata,omitempty"`
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

	var envelope successEnvelope[[]Event]
	if err := c.do(ctx, http.MethodGet, path, nil, &envelope); err != nil {
		return nil, err
	}

	if envelope.Data == nil {
		return []Event{}, nil
	}
	return envelope.Data, nil
}

// FindEventByExternalID retrieves a single event by its external ID.
// Returns ErrEmptyExternalID if externalID is empty.
func (c *Client) FindEventByExternalID(ctx context.Context, externalID string) (*Event, error) {
	if externalID == "" {
		return nil, ErrEmptyExternalID
	}

	path := "/public/logs/external/" + url.PathEscape(externalID)

	var envelope successEnvelope[*Event]
	if err := c.do(ctx, http.MethodGet, path, nil, &envelope); err != nil {
		return nil, err
	}

	return envelope.Data, nil
}

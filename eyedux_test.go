package eyeduxsdk

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// newTestClient spins up an httptest.Server with handler and returns a Client pointed at it.
// The server is closed automatically when the test ends.
func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	c, err := New("test-api-key")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	c.baseURL = srv.URL
	return c
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body) //nolint:errcheck
}

func apiErrorBody(code, message string) map[string]any {
	return map[string]any{"error": map[string]any{"code": code, "message": message}}
}

// ---- New ----

func TestNew_emptyAPIKey(t *testing.T) {
	_, err := New("")
	if !errors.Is(err, ErrEmptyAPIKey) {
		t.Errorf("err = %v, want ErrEmptyAPIKey", err)
	}
}

func TestNew_trimsAPIKeyAndAppliesTimeout(t *testing.T) {
	c, err := New(" key ", WithTimeout(2*time.Second))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.apiKey != "key" {
		t.Errorf("apiKey = %q, want key", c.apiKey)
	}
	if c.httpClient.Timeout != 2*time.Second {
		t.Errorf("timeout = %s, want 2s", c.httpClient.Timeout)
	}
}

func TestNew_usesCustomHTTPClient(t *testing.T) {
	httpClient := &http.Client{}
	c, err := New("key", WithHTTPClient(httpClient))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.httpClient != httpClient {
		t.Error("client did not use the configured HTTP client")
	}
}

func TestNew_validAPIKey(t *testing.T) {
	c, err := New("key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestNewWithConfig_requiresProjectID(t *testing.T) {
	_, err := NewWithConfig(Config{APIKey: "key"})
	if !errors.Is(err, ErrEmptyProjectID) {
		t.Errorf("err = %v, want ErrEmptyProjectID", err)
	}
}

func TestNewWithConfig_requiresAPIKey(t *testing.T) {
	_, err := NewWithConfig(Config{ProjectID: "project"})
	if !errors.Is(err, ErrEmptyAPIKey) {
		t.Errorf("err = %v, want ErrEmptyAPIKey", err)
	}
}

func TestNewWithConfig_valid(t *testing.T) {
	httpClient := &http.Client{}
	metadata := map[string]any{"service": "api"}
	c, err := NewWithConfig(Config{
		APIKey:          "key",
		ProjectID:       " project ",
		HTTPClient:      httpClient,
		Timeout:         2 * time.Second,
		DefaultMetadata: metadata,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.projectID != "project" {
		t.Errorf("projectID = %q, want project", c.projectID)
	}
	if c.httpClient != httpClient {
		t.Error("client did not use the configured HTTP client")
	}
	if c.defaultMetadata["service"] != "api" {
		t.Errorf("default metadata = %v, want service=api", c.defaultMetadata)
	}
	metadata["service"] = "changed"
	if c.defaultMetadata["service"] != "api" {
		t.Error("client metadata changed when the source map was mutated")
	}
}

func TestNewFromEnv_readsOnlyAPIKey(t *testing.T) {
	t.Setenv("EYEDUX_API_KEY", "key")
	t.Setenv("EYEDUX_PROJECT_ID", "ignored")

	c, err := NewFromEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.projectID != "" {
		t.Errorf("projectID = %q, want empty", c.projectID)
	}
}

func TestNewFromEnv_requiresAPIKey(t *testing.T) {
	t.Setenv("EYEDUX_API_KEY", "")

	_, err := NewFromEnv()
	if !errors.Is(err, ErrEmptyAPIKey) {
		t.Errorf("err = %v, want ErrEmptyAPIKey", err)
	}
}

func TestEventEyeduxTypeValues(t *testing.T) {
	tests := []struct {
		name  string
		value EventEyeduxType
		want  string
	}{
		{name: "system error", value: EventEyeduxTypeSystemError, want: "system-error"},
		{name: "system warning", value: EventEyeduxTypeSystemWarning, want: "system-warning"},
		{name: "system log", value: EventEyeduxTypeSystemLog, want: "system-log"},
		{name: "system debug", value: EventEyeduxTypeSystemDebug, want: "system-debug"},
		{name: "system info", value: EventEyeduxTypeSystemInfo, want: "system-info"},
		{name: "system metric", value: EventEyeduxTypeSystemMetric, want: "system-metric"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if string(test.value) != test.want {
				t.Errorf("value = %q, want %q", test.value, test.want)
			}
		})
	}
}

// ---- CreateEvent ----

func successCreateEventHandler(t *testing.T, expectedEyeduxType EventEyeduxType) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/public/logs" {
			t.Errorf("path = %s, want /public/logs", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-api-key" {
			t.Errorf("Authorization = %q", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", got)
		}
		var requestBody struct {
			EyeduxType *EventEyeduxType `json:"eyedux_type"`
		}
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if requestBody.EyeduxType == nil || *requestBody.EyeduxType != expectedEyeduxType {
			t.Errorf("eyedux_type = %v, want %q", requestBody.EyeduxType, expectedEyeduxType)
		}
		writeJSON(w, http.StatusCreated, map[string]any{
			"data": map[string]any{
				"id":          "abc123",
				"environment": "production",
				"eyedux_type": "system-log",
				"type":        "user.signup",
				"properties":  map[string]any{"plan": "pro"},
				"status":      "active",
				"timestamp":   "2026-08-13T10:00:00Z",
				"created_at":  "2026-08-13T10:00:01Z",
			},
		})
	}
}

func TestCreateEvent_success(t *testing.T) {
	eyeduxType := EventEyeduxTypeSystemLog

	c := newTestClient(t, successCreateEventHandler(t, eyeduxType))

	event, err := c.CreateEvent(context.Background(), CreateEventInput{
		ProjectID:  "64f1a2b3c4d5e6f7a8b9c0d1",
		Type:       "user.signup",
		EyeduxType: eyeduxType,
		Properties: map[string]any{"plan": "pro"},
	})
	if err != nil {
		t.Fatalf("CreateEvent: %v", err)
	}
	if event.ID != "abc123" {
		t.Errorf("ID = %s, want abc123", event.ID)
	}
	if event.Type != "user.signup" {
		t.Errorf("Type = %s, want user.signup", event.Type)
	}
	if event.Environment != "production" {
		t.Errorf("Environment = %s, want production", event.Environment)
	}
	if event.EyeduxType == nil || *event.EyeduxType != eyeduxType {
		t.Errorf("EyeduxType = %v, want %q", event.EyeduxType, eyeduxType)
	}
	if event.Status != "active" {
		t.Errorf("Status = %s, want active", event.Status)
	}
}

func TestCreateEvent_usesDefaultProjectAndMetadata(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		var requestBody struct {
			ProjectID string         `json:"project_id"`
			Metadata  map[string]any `json:"metadata"`
		}
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		expectedProjectID := "default-project"
		if requestCount == 2 {
			expectedProjectID = "explicit-project"
		}
		if requestBody.ProjectID != expectedProjectID {
			t.Errorf("project_id = %q, want %s", requestBody.ProjectID, expectedProjectID)
		}
		if requestBody.Metadata["service"] != "api" {
			t.Errorf("metadata.service = %v, want api", requestBody.Metadata["service"])
		}
		if requestBody.Metadata["environment"] != "test" {
			t.Errorf("metadata.environment = %v, want test", requestBody.Metadata["environment"])
		}
		writeJSON(w, http.StatusCreated, map[string]any{"data": map[string]any{"id": "abc123"}})
	}))
	t.Cleanup(server.Close)

	c, err := New("key",
		WithProjectID("default-project"),
		WithDefaultMetadata(map[string]any{"service": "eyedux", "environment": "test"}),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	c.baseURL = server.URL

	_, err = c.CreateEvent(context.Background(), CreateEventInput{
		Type:       "api.request",
		Properties: map[string]any{"method": "GET"},
		Metadata:   map[string]any{"service": "api"},
	})
	if err != nil {
		t.Fatalf("CreateEvent: %v", err)
	}
	_, err = c.CreateEvent(context.Background(), CreateEventInput{
		Type:       "api.request",
		ProjectID:  "explicit-project",
		Properties: map[string]any{"method": "GET"},
		Metadata:   map[string]any{"service": "api"},
	})
	if err != nil {
		t.Fatalf("CreateEvent: %v", err)
	}
}

func TestCreateEvent_requiresProjectID(t *testing.T) {
	requestCount := 0
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		requestCount++
	})

	_, err := c.CreateEvent(context.Background(), CreateEventInput{
		Type:       "api.error",
		Properties: map[string]any{"message": "failed"},
	})
	if !errors.Is(err, ErrEmptyProjectID) {
		t.Errorf("err = %v, want ErrEmptyProjectID", err)
	}
	if requestCount != 0 {
		t.Errorf("requestCount = %d, want 0", requestCount)
	}
}

func TestCreateEvent_omitsEmptyEyeduxType(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		var requestBody map[string]json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if _, ok := requestBody["eyedux_type"]; ok {
			t.Error("eyedux_type must be omitted when empty")
		}
		writeJSON(w, http.StatusCreated, map[string]any{"data": map[string]any{"id": "abc123"}})
	})

	_, err := c.CreateEvent(context.Background(), CreateEventInput{
		ProjectID:  "project",
		Type:       "api.request",
		Properties: map[string]any{"method": "GET"},
	})
	if err != nil {
		t.Fatalf("CreateEvent: %v", err)
	}
}

func TestCreateEvent_returnsMarshalErrorWithoutRequest(t *testing.T) {
	requestCount := 0
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		requestCount++
	})

	_, err := c.CreateEvent(context.Background(), CreateEventInput{
		ProjectID: "project",
		Type:      "api.request",
		Properties: map[string]any{
			"unsupported": func() {},
		},
	})
	if err == nil {
		t.Fatal("CreateEvent must return an error for an unserializable property")
	}
	if requestCount != 0 {
		t.Errorf("requestCount = %d, want 0", requestCount)
	}
}

func TestCreateEvent_propagatesContextCancellation(t *testing.T) {
	httpClient := &http.Client{}
	c, err := New("key", WithHTTPClient(httpClient), WithProjectID("project"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	c.httpClient.Transport = roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		<-r.Context().Done()
		return nil, r.Context().Err()
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = c.CreateEvent(ctx, CreateEventInput{
		Type:       "api.request",
		Properties: map[string]any{"method": "GET"},
	})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want context.Canceled", err)
	}
}

func TestCreateEvent_returnsDecodeErrorForInvalidSuccessResponse(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, "{")
	})

	_, err := c.CreateEvent(context.Background(), CreateEventInput{
		ProjectID:  "project",
		Type:       "api.request",
		Properties: map[string]any{"method": "GET"},
	})
	if err == nil {
		t.Fatal("CreateEvent must return an error for invalid JSON")
	}
}

func TestCreateEvent_preservesStatusForMalformedAPIError(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = io.WriteString(w, "upstream failure")
	})

	_, err := c.CreateEvent(context.Background(), CreateEventInput{
		ProjectID:  "project",
		Type:       "api.request",
		Properties: map[string]any{"method": "GET"},
	})
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != http.StatusBadGateway {
		t.Errorf("status = %d, want %d", apiErr.StatusCode, http.StatusBadGateway)
	}
}

func TestCreateEvent_conflict(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusConflict, apiErrorBody(ErrCodeEventExternalObjectConflict, "conflict"))
	})

	_, err := c.CreateEvent(context.Background(), CreateEventInput{
		ProjectID:  "64f1a2b3c4d5e6f7a8b9c0d1",
		Type:       "user.signup",
		Properties: map[string]any{"plan": "pro"},
	})
	if !IsConflict(err) {
		t.Errorf("expected conflict error, got %v", err)
	}
	if !IsExternalObjectConflict(err) {
		t.Errorf("expected external object conflict, got %v", err)
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Code != ErrCodeEventExternalObjectConflict {
		t.Errorf("expected code %s, got %v", ErrCodeEventExternalObjectConflict, err)
	}
}

func TestCreateEvent_authError(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusUnprocessableEntity, apiErrorBody(ErrCodeInvalidAPIKey, "invalid key"))
	})

	_, err := c.CreateEvent(context.Background(), CreateEventInput{
		ProjectID:  "64f1a2b3c4d5e6f7a8b9c0d1",
		Type:       "user.signup",
		Properties: map[string]any{"plan": "pro"},
	})
	if !IsAuthError(err) {
		t.Errorf("expected auth error, got %v", err)
	}
}

func TestCreateEvent_rateLimited(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "2")
		writeJSON(w, http.StatusTooManyRequests, apiErrorBody(ErrCodeRateLimitExceeded, "too many requests"))
	})

	_, err := c.CreateEvent(context.Background(), CreateEventInput{
		ProjectID:  "64f1a2b3c4d5e6f7a8b9c0d1",
		Type:       "user.signup",
		Properties: map[string]any{"plan": "pro"},
	})
	if !IsRateLimited(err) {
		t.Errorf("expected rate limit error, got %v", err)
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.RetryAfter == nil || *apiErr.RetryAfter != 2 {
		t.Errorf("RetryAfter = %v, want 2", apiErr.RetryAfter)
	}
}

func TestCreateEvent_internalServerError(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusInternalServerError, apiErrorBody(ErrCodeInternalServerError, "internal error"))
	})

	_, err := c.CreateEvent(context.Background(), CreateEventInput{
		ProjectID:  "64f1a2b3c4d5e6f7a8b9c0d1",
		Type:       "user.signup",
		Properties: map[string]any{"plan": "pro"},
	})
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 500 APIError, got %v", err)
	}
}

// ---- ListEvents ----

func TestListEvents_success(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/public/logs" {
			t.Errorf("path = %s, want /public/logs", r.URL.Path)
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"data": []map[string]any{
				{
					"id":         "abc123",
					"type":       "user.signup",
					"properties": map[string]any{"plan": "pro"},
					"status":     "active",
					"timestamp":  "2026-08-13T10:00:00Z",
					"created_at": "2026-08-13T10:00:01Z",
				},
			},
		})
	})

	events, err := c.ListEvents(context.Background(), ListEventsInput{})
	if err != nil {
		t.Fatalf("ListEvents: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("len(events) = %d, want 1", len(events))
	}
	if events[0].ID != "abc123" {
		t.Errorf("ID = %s, want abc123", events[0].ID)
	}
}

func TestListEvents_empty(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"data": []any{}})
	})

	events, err := c.ListEvents(context.Background(), ListEventsInput{})
	if err != nil {
		t.Fatalf("ListEvents: %v", err)
	}
	if events == nil {
		t.Error("events must not be nil for an empty list")
	}
	if len(events) != 0 {
		t.Errorf("len(events) = %d, want 0", len(events))
	}
}

func TestListEvents_nullDataReturnsEmptySlice(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"data": nil})
	})

	events, err := c.ListEvents(context.Background(), ListEventsInput{})
	if err != nil {
		t.Fatalf("ListEvents: %v", err)
	}
	if events == nil {
		t.Fatal("events must not be nil when data is null")
	}
	if len(events) != 0 {
		t.Errorf("len(events) = %d, want 0", len(events))
	}
}

func TestListEvents_withFilters(t *testing.T) {
	wantType := "user.signup"
	wantCorrID := "session_abc"

	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("type"); got != wantType {
			t.Errorf("type = %q, want %q", got, wantType)
		}
		if got := r.URL.Query().Get("correlation_id"); got != wantCorrID {
			t.Errorf("correlation_id = %q, want %q", got, wantCorrID)
		}
		writeJSON(w, http.StatusOK, map[string]any{"data": []any{}})
	})

	_, err := c.ListEvents(context.Background(), ListEventsInput{
		Type:          &wantType,
		CorrelationID: &wantCorrID,
	})
	if err != nil {
		t.Fatalf("ListEvents: %v", err)
	}
}

// ---- FindEventByExternalID ----

func TestFindEventByExternalID_success(t *testing.T) {
	externalID := "evt_01HX92K"

	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		wantPath := "/public/logs/external/" + externalID
		if r.URL.Path != wantPath {
			t.Errorf("path = %s, want %s", r.URL.Path, wantPath)
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"data": map[string]any{
				"id":         "abc123",
				"type":       "user.signup",
				"properties": map[string]any{"plan": "pro"},
				"status":     "active",
				"timestamp":  "2026-08-13T10:00:00Z",
				"created_at": "2026-08-13T10:00:01Z",
				"external_object": map[string]any{
					"id":       externalID,
					"property": "orderId",
				},
			},
		})
	})

	event, err := c.FindEventByExternalID(context.Background(), externalID)
	if err != nil {
		t.Fatalf("FindEventByExternalID: %v", err)
	}
	if event.ExternalObject == nil || event.ExternalObject.ID != externalID {
		t.Errorf("ExternalObject.ID = %v, want %s", event.ExternalObject, externalID)
	}
}

func TestFindEventByExternalID_emptyExternalID(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler must not be called for empty externalID")
	})

	_, err := c.FindEventByExternalID(context.Background(), "")
	if !errors.Is(err, ErrEmptyExternalID) {
		t.Errorf("err = %v, want ErrEmptyExternalID", err)
	}
}

func TestFindEventByExternalID_notFound(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusNotFound, apiErrorBody(ErrCodeEventExternalIDNotFound, "not found"))
	})

	_, err := c.FindEventByExternalID(context.Background(), "nonexistent")
	if !IsNotFound(err) {
		t.Errorf("expected not found error, got %v", err)
	}
}

// ---- Error helpers ----

func TestIsNotFound(t *testing.T) {
	if IsNotFound(nil) {
		t.Error("IsNotFound(nil) must be false")
	}
	if IsNotFound(&APIError{StatusCode: 422}) {
		t.Error("IsNotFound(422) must be false")
	}
	if !IsNotFound(&APIError{StatusCode: 404}) {
		t.Error("IsNotFound(404) must be true")
	}
}

func TestIsConflict(t *testing.T) {
	if IsConflict(nil) {
		t.Error("IsConflict(nil) must be false")
	}
	if !IsConflict(&APIError{StatusCode: 409}) {
		t.Error("IsConflict(409) must be true")
	}
}

func TestIsExternalObjectConflict(t *testing.T) {
	if IsExternalObjectConflict(nil) {
		t.Error("IsExternalObjectConflict(nil) must be false")
	}
	if IsExternalObjectConflict(&APIError{StatusCode: 409}) {
		t.Error("missing error code must not be treated as an external object conflict")
	}
	if !IsExternalObjectConflict(&APIError{Code: ErrCodeEventExternalObjectConflict}) {
		t.Error("matching error code must be treated as an external object conflict")
	}
}

func TestAPIError_Error(t *testing.T) {
	err := (&APIError{StatusCode: http.StatusConflict, Code: ErrCodeEventExternalObjectConflict}).Error()
	if err != "eyedux: event_external_object_conflict (status 409)" {
		t.Errorf("error = %q, want formatted API error", err)
	}
}

func TestIsRateLimited(t *testing.T) {
	if IsRateLimited(nil) {
		t.Error("IsRateLimited(nil) must be false")
	}
	if !IsRateLimited(&APIError{StatusCode: 429}) {
		t.Error("IsRateLimited(429) must be true")
	}
}

func TestIsAuthError(t *testing.T) {
	if IsAuthError(nil) {
		t.Error("IsAuthError(nil) must be false")
	}
	if !IsAuthError(&APIError{StatusCode: 422, Code: ErrCodeInvalidAPIKey}) {
		t.Error("IsAuthError with invalid_api_key must be true")
	}
}

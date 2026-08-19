package eyeduxsdk

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
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

func TestNew_validAPIKey(t *testing.T) {
	c, err := New("key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c == nil {
		t.Fatal("expected non-nil client")
	}
}

// ---- CreateEvent ----

func TestCreateEvent_success(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
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
		writeJSON(w, http.StatusCreated, map[string]any{
			"data": map[string]any{
				"id":         "abc123",
				"type":       "user.signup",
				"properties": map[string]any{"plan": "pro"},
				"status":     "active",
				"timestamp":  "2026-08-13T10:00:00Z",
				"created_at": "2026-08-13T10:00:01Z",
			},
		})
	})

	event, err := c.CreateEvent(context.Background(), CreateEventInput{
		ProjectID:  "64f1a2b3c4d5e6f7a8b9c0d1",
		Type:       "user.signup",
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
	if event.Status != "active" {
		t.Errorf("Status = %s, want active", event.Status)
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

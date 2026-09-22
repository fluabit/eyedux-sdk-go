package eyeduxsdk

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestClientEmit_usesRequestedEyeduxType(t *testing.T) {
	const expectedType = EventEyeduxTypeSystemLog

	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			EyeduxType EventEyeduxType `json:"eyedux_type"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if body.EyeduxType != expectedType {
			t.Errorf("eyedux_type = %q, want %q", body.EyeduxType, expectedType)
		}
		writeJSON(w, http.StatusCreated, map[string]any{"data": map[string]any{"id": "event-123"}})
	})

	_, err := c.Emit(context.Background(), EmitInput{
		ProjectID:  "project",
		Type:       "api.request",
		Properties: map[string]any{"method": "GET"},
		EyeduxType: expectedType,
	})
	if err != nil {
		t.Fatalf("Emit: %v", err)
	}
}

func TestClientEmitConveniences_useTheirCategories(t *testing.T) {
	tests := []struct {
		name string
		want EventEyeduxType
		emit func(*Client, context.Context, EmitInput) (*Event, error)
	}{
		{name: "warning", want: EventEyeduxTypeSystemWarning, emit: (*Client).EmitWarning},
		{name: "log", want: EventEyeduxTypeSystemLog, emit: (*Client).EmitLog},
		{name: "debug", want: EventEyeduxTypeSystemDebug, emit: (*Client).EmitDebug},
		{name: "info", want: EventEyeduxTypeSystemInfo, emit: (*Client).EmitInfo},
		{name: "audit", want: EventEyeduxTypeAudit, emit: (*Client).EmitAudit},
		{name: "metric", want: EventEyeduxTypeMetric, emit: (*Client).EmitMetric},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				var body struct {
					EyeduxType EventEyeduxType `json:"eyedux_type"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Fatalf("decode request body: %v", err)
				}
				if body.EyeduxType != test.want {
					t.Errorf("eyedux_type = %q, want %q", body.EyeduxType, test.want)
				}
				writeJSON(w, http.StatusCreated, map[string]any{"data": map[string]any{"id": "event-123"}})
			})

			_, err := test.emit(c, context.Background(), EmitInput{
				ProjectID:  "project",
				Type:       "api.request",
				Properties: map[string]any{"method": "GET"},
			})
			if err != nil {
				t.Fatalf("%s: %v", test.name, err)
			}
		})
	}
}

func TestClientEmitAudit_usesStructuredProperties(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			EyeduxType EventEyeduxType `json:"eyedux_type"`
			Properties map[string]any  `json:"properties"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if body.EyeduxType != EventEyeduxTypeAudit {
			t.Errorf("eyedux_type = %q, want %q", body.EyeduxType, EventEyeduxTypeAudit)
		}
		actor := body.Properties["actor"].(map[string]any)
		if actor["type"] != "service" || actor["id"] != "account-service" || actor["source"] != "account-service" {
			t.Errorf("actor = %v, want service/account-service/account-service", actor)
		}
		if body.Properties["result"] != "denied" {
			t.Errorf("result = %v, want denied", body.Properties["result"])
		}
		if body.Properties["reason"] != "missing permission" {
			t.Errorf("reason = %v, want missing permission", body.Properties["reason"])
		}
		writeJSON(w, http.StatusCreated, map[string]any{"data": map[string]any{"id": "event-123"}})
	})

	_, err := c.EmitAudit(context.Background(), EmitInput{
		ProjectID: "project",
		Type:      "user.password_changed",
		AuditProperties: &AuditProperties{
			Actor:  AuditActor{Type: AuditActorTypeService, ID: "account-service", Source: "account-service"},
			Target: AuditTarget{Type: "user", ID: "user_123", Source: "identity-service"},
			Result: AuditResultDenied,
			Reason: "missing permission",
		},
	})
	if err != nil {
		t.Fatalf("EmitAudit: %v", err)
	}
}

func TestClientEmitMetric_usesStructuredProperties(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			EyeduxType EventEyeduxType `json:"eyedux_type"`
			Properties map[string]any  `json:"properties"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if body.EyeduxType != EventEyeduxTypeMetric {
			t.Errorf("eyedux_type = %q, want %q", body.EyeduxType, EventEyeduxTypeMetric)
		}
		if body.Properties["value"] != 120.5 {
			t.Errorf("value = %v, want 120.5", body.Properties["value"])
		}
		if body.Properties["unit"] != "milliseconds" {
			t.Errorf("unit = %v, want milliseconds", body.Properties["unit"])
		}
		dimensions := body.Properties["dimensions"].(map[string]any)
		if dimensions["route"] != "/checkout" {
			t.Errorf("dimensions.route = %v, want /checkout", dimensions["route"])
		}
		writeJSON(w, http.StatusCreated, map[string]any{"data": map[string]any{"id": "event-123"}})
	})

	_, err := c.EmitMetric(context.Background(), EmitInput{
		ProjectID: "project",
		Type:      "api.request.duration",
		MetricProperties: &MetricProperties{
			Value:      120.5,
			Unit:       MetricUnitMilliseconds,
			Dimensions: map[string]string{"route": "/checkout"},
		},
	})
	if err != nil {
		t.Fatalf("EmitMetric: %v", err)
	}
}

func TestClientEmitMetric_returnsValidationError(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("request should not be sent when properties are invalid")
	})

	_, err := c.EmitMetric(context.Background(), EmitInput{
		ProjectID: "project",
		Type:      "api.request.duration",
		MetricProperties: &MetricProperties{
			Value: 150,
			Unit:  MetricUnitPercent,
		},
	})
	if err == nil {
		t.Fatal("EmitMetric: expected error, got nil")
	}
}

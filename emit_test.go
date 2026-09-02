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
		{name: "metric", want: EventEyeduxTypeSystemMetric, emit: (*Client).EmitMetric},
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

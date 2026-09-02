package eyeduxsdk

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
)

func TestCurrentErrorSource_returnsCallingLocation(t *testing.T) {
	source := CurrentErrorSource()

	if source.File != "diagnostics_test.go" {
		t.Errorf("File = %q, want diagnostics_test.go", source.File)
	}
	if !strings.HasSuffix(source.Function, "TestCurrentErrorSource_returnsCallingLocation") {
		t.Errorf("Function = %q, want test function", source.Function)
	}
	if source.Line <= 0 {
		t.Errorf("Line = %d, want a positive line", source.Line)
	}
}

func TestErrorProperties_copiesAndEnriches(t *testing.T) {
	properties := map[string]any{
		"request_id": "req-123",
		"error":      "old error",
		"operation":  "old operation",
	}

	enriched := ErrorProperties(properties, errors.New("request failed"), "create order")
	properties["request_id"] = "changed"

	if enriched["request_id"] != "req-123" {
		t.Errorf("request_id = %v, want req-123", enriched["request_id"])
	}
	if enriched["error"] != "request failed" {
		t.Errorf("error = %v, want request failed", enriched["error"])
	}
	if enriched["operation"] != "create order" {
		t.Errorf("operation = %v, want create order", enriched["operation"])
	}
	if enriched["source_file"] != "diagnostics_test.go" {
		t.Errorf("source_file = %v, want diagnostics_test.go", enriched["source_file"])
	}
	if !strings.HasSuffix(enriched["source_function"].(string), "TestErrorProperties_copiesAndEnriches") {
		t.Errorf("source_function = %v, want test function", enriched["source_function"])
	}
	if _, ok := properties["source_file"]; ok {
		t.Error("input properties were modified")
	}
}

func TestClientEmitError_createsEnrichedEvent(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			EyeduxType EventEyeduxType `json:"eyedux_type"`
			Properties map[string]any  `json:"properties"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if body.Properties["error"] != "request failed" {
			t.Errorf("error = %v, want request failed", body.Properties["error"])
		}
		if body.EyeduxType != EventEyeduxTypeSystemError {
			t.Errorf("eyedux_type = %q, want %q", body.EyeduxType, EventEyeduxTypeSystemError)
		}
		if body.Properties["operation"] != "create order" {
			t.Errorf("operation = %v, want create order", body.Properties["operation"])
		}
		if body.Properties["source_file"] != "diagnostics_test.go" {
			t.Errorf("source_file = %v, want diagnostics_test.go", body.Properties["source_file"])
		}
		if !strings.HasSuffix(body.Properties["source_function"].(string), "TestClientEmitError_createsEnrichedEvent") {
			t.Errorf("source_function = %v, want test function", body.Properties["source_function"])
		}
		writeJSON(w, http.StatusCreated, map[string]any{
			"data": map[string]any{"id": "event-123"},
		})
	})

	properties := map[string]any{"request_id": "req-123"}
	event, err := c.EmitError(context.Background(), EmitInput{
		ProjectID:  "project",
		Type:       "api.error",
		Properties: properties,
		Err:        errors.New("request failed"),
		Operation:  "create order",
	})
	if err != nil {
		t.Fatalf("EmitError: %v", err)
	}
	if event.ID != "event-123" {
		t.Errorf("ID = %q, want event-123", event.ID)
	}
	if _, ok := properties["error"]; ok {
		t.Error("EmitError modified input properties")
	}
}

func TestClientEmitErrorWithSourceSkip_skipsWrapper(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Properties map[string]any `json:"properties"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if !strings.HasSuffix(body.Properties["source_function"].(string), "TestClientEmitErrorWithSourceSkip_skipsWrapper") {
			t.Errorf("source_function = %v, want test function", body.Properties["source_function"])
		}
		writeJSON(w, http.StatusCreated, map[string]any{"data": map[string]any{"id": "event-123"}})
	})

	if _, err := emitErrorThroughWrapper(c); err != nil {
		t.Fatalf("EmitErrorWithSourceSkip: %v", err)
	}
}

func emitErrorThroughWrapper(c *Client) (*Event, error) {
	return c.EmitError(context.Background(), EmitInput{
		ProjectID:  "project",
		Type:       "order.error",
		Err:        errors.New("request failed"),
		Operation:  "create order",
		SourceSkip: 1,
	})
}

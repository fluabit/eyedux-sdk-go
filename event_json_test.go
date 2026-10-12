package eyeduxsdk

import (
	"encoding/json"
	"testing"
)

func TestEventJSON_preservesNestedProperties(t *testing.T) {
	event := Event{
		ID: "event-123",
		Properties: map[string]any{
			"order_id": "order-456",
			"code":     "free-form-code",
			"workflow": []any{"any", "value"},
		},
	}

	encoded, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal Event: %v", err)
	}
	var root map[string]any
	if err := json.Unmarshal(encoded, &root); err != nil {
		t.Fatalf("unmarshal encoded Event: %v", err)
	}
	properties, ok := root["properties"].(map[string]any)
	if !ok {
		t.Fatalf("properties = %T, want map[string]any", root["properties"])
	}
	if properties["order_id"] != "order-456" {
		t.Errorf("properties.order_id = %v, want order-456", properties["order_id"])
	}

	var decoded Event
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal Event: %v", err)
	}
	if decoded.Properties["order_id"] != "order-456" {
		t.Errorf("Properties[order_id] = %v, want order-456", decoded.Properties["order_id"])
	}
	if decoded.Properties["code"] != "free-form-code" {
		t.Errorf("Properties[code] = %v, want free-form-code", decoded.Properties["code"])
	}
	if got, ok := decoded.Properties["workflow"].([]any); !ok || len(got) != 2 {
		t.Errorf("Properties[workflow] = %v, want two-item array", decoded.Properties["workflow"])
	}
}

func TestEventJSON_readsNestedProperties(t *testing.T) {
	var event Event
	if err := json.Unmarshal([]byte(`{"id":"event-123","properties":{"order_id":"order-456"}}`), &event); err != nil {
		t.Fatalf("unmarshal Event: %v", err)
	}
	if event.Properties["order_id"] != "order-456" {
		t.Errorf("Properties[order_id] = %v, want order-456", event.Properties["order_id"])
	}
}

func TestEventJSON_allowsPropertyMatchingEnvelopeField(t *testing.T) {
	encoded, err := json.Marshal(Event{
		Type:       "original",
		Properties: map[string]any{"type": "custom"},
	})
	if err != nil {
		t.Fatalf("marshal Event: %v", err)
	}
	var root map[string]any
	if err := json.Unmarshal(encoded, &root); err != nil {
		t.Fatalf("unmarshal encoded Event: %v", err)
	}
	if root["type"] != "original" {
		t.Errorf("type = %v, want original event type", root["type"])
	}
	properties, ok := root["properties"].(map[string]any)
	if !ok {
		t.Fatalf("properties = %T, want map[string]any", root["properties"])
	}
	if properties["type"] != "custom" {
		t.Errorf("properties.type = %v, want custom property value", properties["type"])
	}
}

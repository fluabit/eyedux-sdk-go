package eyeduxsdk

import (
	"context"
	"encoding/json"
	"net/http"
	"runtime"
	"testing"
)

func TestCollectRuntimeMetadata_includesRuntimeAndProcessInformation(t *testing.T) {
	metadata := collectRuntimeMetadata()

	runtimeMetadata, ok := metadata["runtime"].(map[string]any)
	if !ok {
		t.Fatalf("runtime metadata = %T, want map[string]any", metadata["runtime"])
	}
	if runtimeMetadata["os"] != runtime.GOOS {
		t.Errorf("runtime.os = %v, want %s", runtimeMetadata["os"], runtime.GOOS)
	}
	if runtimeMetadata["architecture"] != runtime.GOARCH {
		t.Errorf("runtime.architecture = %v, want %s", runtimeMetadata["architecture"], runtime.GOARCH)
	}
	if runtimeMetadata["cpu_count"] != runtime.NumCPU() {
		t.Errorf("runtime.cpu_count = %v, want %d", runtimeMetadata["cpu_count"], runtime.NumCPU())
	}

	memoryMetadata, ok := metadata["memory"].(map[string]any)
	if !ok {
		t.Fatalf("memory metadata = %T, want map[string]any", metadata["memory"])
	}
	for _, field := range []string{"alloc_bytes", "heap_alloc_bytes", "sys_bytes"} {
		if _, ok := memoryMetadata[field].(uint64); !ok {
			t.Errorf("memory.%s = %T, want uint64", field, memoryMetadata[field])
		}
	}
	if _, ok := metadata["ip_addresses"].([]string); !ok {
		t.Errorf("ip_addresses = %T, want []string", metadata["ip_addresses"])
	}
}

func TestCreateEvent_addsRuntimeMetadataForSupportedEventTypes(t *testing.T) {
	tests := []struct {
		name      string
		eventType EventEyeduxType
		want      bool
	}{
		{name: "system event", eventType: EventEyeduxTypeSystemLog, want: true},
		{name: "metric", eventType: EventEyeduxTypeMetric, want: true},
		{name: "audit", eventType: EventEyeduxTypeAudit, want: true},
		{name: "custom event", eventType: "custom", want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				var body struct {
					Metadata map[string]any `json:"metadata"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Fatalf("decode request body: %v", err)
				}
				_, got := body.Metadata["runtime_metadata"]
				if got != test.want {
					t.Errorf("runtime metadata present = %t, want %t", got, test.want)
				}
				writeJSON(w, http.StatusCreated, map[string]any{"data": map[string]any{"id": "event-123"}})
			})
			c.metadataProvider = func() map[string]any {
				return map[string]any{"runtime_metadata": true}
			}

			_, err := c.CreateEvent(context.Background(), CreateEventInput{
				ProjectID:  "project",
				Type:       "api.request",
				EyeduxType: test.eventType,
				Properties: map[string]any{"method": "GET"},
			})
			if err != nil {
				t.Fatalf("CreateEvent: %v", err)
			}
		})
	}
}

func TestCreateEvent_runtimeMetadataCanBeOverridden(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Metadata map[string]any `json:"metadata"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if body.Metadata["hostname"] != "application-host" {
			t.Errorf("metadata.hostname = %v, want application-host", body.Metadata["hostname"])
		}
		writeJSON(w, http.StatusCreated, map[string]any{"data": map[string]any{"id": "event-123"}})
	})
	c.metadataProvider = func() map[string]any {
		return map[string]any{"hostname": "runtime-host"}
	}

	_, err := c.CreateEvent(context.Background(), CreateEventInput{
		ProjectID:  "project",
		Type:       "api.request",
		EyeduxType: EventEyeduxTypeSystemLog,
		Properties: map[string]any{"method": "GET"},
		Metadata:   map[string]any{"hostname": "application-host"},
	})
	if err != nil {
		t.Fatalf("CreateEvent: %v", err)
	}
}

func TestAutomaticMetadataCanBeDisabled(t *testing.T) {
	c, err := New("key", WithAutomaticMetadata(false))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if c.automaticMetadata {
		t.Error("automatic metadata is enabled, want disabled")
	}

	c, err = NewWithConfig(Config{
		APIKey:                   "key",
		ProjectID:                "project",
		DisableAutomaticMetadata: true,
	})
	if err != nil {
		t.Fatalf("NewWithConfig: %v", err)
	}
	if c.automaticMetadata {
		t.Error("automatic metadata is enabled, want disabled")
	}
}

package eyeduxsdk

import (
	"encoding/json"
	"testing"
)

func TestAuditPropertiesToMap(t *testing.T) {
	properties, err := (AuditProperties{
		Actor: AuditActor{
			Type:   AuditActorTypeUser,
			ID:     "user_123",
			Source: "identity-service",
		},
		Target: AuditTarget{
			Type:   "user",
			ID:     "user_123",
			Source: "identity-service",
		},
		Result:        AuditResultSuccess,
		StateChanging: true,
		Changes: map[string]any{
			"fields": []string{"password"},
		},
	}).ToMap()
	if err != nil {
		t.Fatalf("ToMap: %v", err)
	}

	encoded, err := json.Marshal(properties)
	if err != nil {
		t.Fatalf("marshal properties: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(encoded, &got); err != nil {
		t.Fatalf("unmarshal properties: %v", err)
	}
	if got["actor"].(map[string]any)["type"] != "user" {
		t.Errorf("actor.type = %v, want user", got["actor"].(map[string]any)["type"])
	}
	if got["actor"].(map[string]any)["id"] != "user_123" {
		t.Errorf("actor.id = %v, want user_123", got["actor"].(map[string]any)["id"])
	}
	if got["actor"].(map[string]any)["source"] != "identity-service" {
		t.Errorf("actor.source = %v, want identity-service", got["actor"].(map[string]any)["source"])
	}
	if got["target"].(map[string]any)["source"] != "identity-service" {
		t.Errorf("target.source = %v, want identity-service", got["target"].(map[string]any)["source"])
	}
	if got["result"] != "success" {
		t.Errorf("result = %v, want success", got["result"])
	}
	if got["changes"].(map[string]any)["fields"].([]any)[0] != "password" {
		t.Errorf("changes.fields = %v, want password", got["changes"])
	}
}

func TestAuditPropertiesToMap_anonymousActorOmitsID(t *testing.T) {
	properties, err := (AuditProperties{
		Actor:  AuditActor{Type: AuditActorTypeAnonymous, Source: "auth-service"},
		Target: AuditTarget{Type: "document", ID: "doc_123", Source: "document-service"},
		Result: AuditResultSuccess,
	}).ToMap()
	if err != nil {
		t.Fatalf("ToMap: %v", err)
	}

	encoded, err := json.Marshal(properties)
	if err != nil {
		t.Fatalf("marshal properties: %v", err)
	}
	if string(encoded) != `{"actor":{"type":"anonymous","source":"auth-service"},"result":"success","target":{"type":"document","id":"doc_123","source":"document-service"}}` {
		t.Errorf("properties JSON = %s", encoded)
	}
}

func TestAuditPropertiesToMap_acceptsCustomActorAndInReviewResult(t *testing.T) {
	properties, err := (AuditProperties{
		Actor:  AuditActor{Type: "automation", ID: "workflow-123", Source: "workflow-service"},
		Target: AuditTarget{Type: "deployment", ID: "deploy-123", Source: "deployment-service"},
		Result: AuditResultInReview,
	}).ToMap()
	if err != nil {
		t.Fatalf("ToMap: %v", err)
	}
	if properties["result"] != AuditResultInReview {
		t.Errorf("result = %v, want %q", properties["result"], AuditResultInReview)
	}
	if properties["actor"].(AuditActor).Type != "automation" {
		t.Errorf("actor.type = %q, want automation", properties["actor"].(AuditActor).Type)
	}
}

func TestAuditPropertiesToMap_rejectsInvalidProperties(t *testing.T) {
	tests := []struct {
		name       string
		properties AuditProperties
	}{
		{
			name: "missing actor type",
			properties: AuditProperties{
				Target: AuditTarget{Type: "user", ID: "user_123"},
				Result: AuditResultSuccess,
			},
		},
		{
			name: "missing actor id",
			properties: AuditProperties{
				Actor:  AuditActor{Type: AuditActorTypeUser, Source: "identity-service"},
				Target: AuditTarget{Type: "user", ID: "user_123", Source: "identity-service"},
				Result: AuditResultSuccess,
			},
		},
		{
			name: "custom actor without id",
			properties: AuditProperties{
				Actor:  AuditActor{Type: "automation", Source: "workflow-service"},
				Target: AuditTarget{Type: "deployment", ID: "deploy-123", Source: "deployment-service"},
				Result: AuditResultInReview,
			},
		},
		{
			name: "missing target",
			properties: AuditProperties{
				Actor:  AuditActor{Type: AuditActorTypeSystem, ID: "system", Source: "system-service"},
				Result: AuditResultSuccess,
			},
		},
		{
			name: "failure without reason",
			properties: AuditProperties{
				Actor:  AuditActor{Type: AuditActorTypeService, ID: "account-service", Source: "account-service"},
				Target: AuditTarget{Type: "user", ID: "user_123", Source: "identity-service"},
				Result: AuditResultFailure,
			},
		},
		{
			name: "denied without reason",
			properties: AuditProperties{
				Actor:  AuditActor{Type: AuditActorTypeService, ID: "account-service", Source: "account-service"},
				Target: AuditTarget{Type: "user", ID: "user_123", Source: "identity-service"},
				Result: AuditResultDenied,
			},
		},
		{
			name: "state-changing success without changes",
			properties: AuditProperties{
				Actor:         AuditActor{Type: AuditActorTypeUser, ID: "user_123", Source: "identity-service"},
				Target:        AuditTarget{Type: "user", ID: "user_123", Source: "identity-service"},
				Result:        AuditResultSuccess,
				StateChanging: true,
			},
		},
		{
			name: "invalid result",
			properties: AuditProperties{
				Actor:  AuditActor{Type: AuditActorTypeSystem, ID: "system", Source: "system-service"},
				Target: AuditTarget{Type: "user", ID: "user_123", Source: "identity-service"},
				Result: "pending",
			},
		},
		{
			name: "missing actor source",
			properties: AuditProperties{
				Actor:  AuditActor{Type: AuditActorTypeUser, ID: "user_123"},
				Target: AuditTarget{Type: "user", ID: "user_123", Source: "identity-service"},
				Result: AuditResultSuccess,
			},
		},
		{
			name: "blank target source",
			properties: AuditProperties{
				Actor:  AuditActor{Type: AuditActorTypeUser, ID: "user_123", Source: "identity-service"},
				Target: AuditTarget{Type: "user", ID: "user_123", Source: "  "},
				Result: AuditResultSuccess,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := test.properties.ToMap(); err == nil {
				t.Error("ToMap must reject invalid audit properties")
			}
		})
	}
}

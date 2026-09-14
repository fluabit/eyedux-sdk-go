package eyeduxsdk

import (
	"fmt"
	"strings"
)

// AuditActorType identifies the kind of actor that performed an audited action.
// The listed constants are conventional values; custom non-empty values are supported.
type AuditActorType string

const (
	AuditActorTypeUser      AuditActorType = "user"
	AuditActorTypeService   AuditActorType = "service"
	AuditActorTypeSystem    AuditActorType = "system"
	AuditActorTypeAdmin     AuditActorType = "admin"
	AuditActorTypeAnonymous AuditActorType = "anonymous"
)

// AuditResult identifies the outcome of an audited action.
type AuditResult string

const (
	AuditResultSuccess  AuditResult = "success"
	AuditResultFailure  AuditResult = "failure"
	AuditResultInReview AuditResult = "in_review"
	AuditResultDenied   AuditResult = "denied"
)

// AuditActor identifies who or what performed an audited action.
// Anonymous actors do not require an ID.
type AuditActor struct {
	Type   AuditActorType `json:"type"`
	ID     string         `json:"id,omitempty"`
	Source string         `json:"source"`
}

// AuditTarget identifies the entity affected by an audited action.
type AuditTarget struct {
	Type   string `json:"type"`
	ID     string `json:"id"`
	Source string `json:"source"`
}

// AuditProperties is the structured properties payload for an audit event.
// Changes is intentionally free-form because the API does not define its
// schema beyond requiring an object for state-changing actions.
type AuditProperties struct {
	Actor         AuditActor     `json:"actor"`
	Target        AuditTarget    `json:"target"`
	Result        AuditResult    `json:"result"`
	Reason        string         `json:"reason,omitempty"`
	Changes       map[string]any `json:"changes,omitempty"`
	StateChanging bool           `json:"-"`
}

// Validate checks the structural rules required by the audit contract.
func (p AuditProperties) Validate() error {
	if err := validateAuditActor(p.Actor); err != nil {
		return err
	}
	if err := validateAuditTarget(p.Target); err != nil {
		return err
	}
	if err := validateAuditResult(p.Result, p.Reason); err != nil {
		return err
	}
	return validateAuditChanges(p.Result, p.StateChanging, p.Changes)
}

func validateAuditActor(actor AuditActor) error {
	if strings.TrimSpace(string(actor.Type)) == "" {
		return fmt.Errorf("eyedux: audit actor type is required")
	}
	if actor.Type != AuditActorTypeAnonymous && strings.TrimSpace(actor.ID) == "" {
		return fmt.Errorf("eyedux: audit actor id is required for actor type %q", actor.Type)
	}
	if strings.TrimSpace(actor.Source) == "" {
		return fmt.Errorf("eyedux: audit actor source is required")
	}
	return nil
}

func validateAuditTarget(target AuditTarget) error {
	if strings.TrimSpace(target.Type) == "" {
		return fmt.Errorf("eyedux: audit target type is required")
	}
	if strings.TrimSpace(target.ID) == "" {
		return fmt.Errorf("eyedux: audit target id is required")
	}
	if strings.TrimSpace(target.Source) == "" {
		return fmt.Errorf("eyedux: audit target source is required")
	}
	return nil
}

func validateAuditResult(result AuditResult, reason string) error {
	switch result {
	case AuditResultSuccess:
	case AuditResultFailure, AuditResultDenied:
		if strings.TrimSpace(reason) == "" {
			return fmt.Errorf("eyedux: audit reason is required for result %q", result)
		}
	case AuditResultInReview:
	case "":
		return fmt.Errorf("eyedux: audit result is required")
	default:
		return fmt.Errorf("eyedux: invalid audit result %q", result)
	}

	return nil
}

func validateAuditChanges(result AuditResult, stateChanging bool, changes map[string]any) error {
	if stateChanging && result == AuditResultSuccess && changes == nil {
		return fmt.Errorf("eyedux: audit changes are required for successful state-changing actions")
	}
	return nil
}

// ToMap validates the model and converts it to CreateEventInput.Properties.
func (p AuditProperties) ToMap() (map[string]any, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}

	properties := map[string]any{
		"actor":  p.Actor,
		"target": p.Target,
		"result": p.Result,
	}
	if p.Reason != "" {
		properties["reason"] = p.Reason
	}
	if p.Changes != nil {
		properties["changes"] = p.Changes
	}

	return properties, nil
}

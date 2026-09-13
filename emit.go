package eyeduxsdk

import "context"

// EmitInput contains the fields used by the convenience emission methods.
// Err, Operation, and SourceSkip are used by EmitError and are ignored by the
// other emission methods. AuditProperties is used by EmitAudit when provided.
type EmitInput struct {
	ProjectID         string
	Type              string
	TypeGroup         string
	EyeduxType        EventEyeduxType
	Properties        map[string]any
	AuditProperties   *AuditProperties
	Err               error
	Operation         string
	SourceSkip        int
	ExternalObject    *EventObject
	CorrelationObject *EventObject
	Metadata          map[string]any
}

// Emit creates an event using input.EyeduxType.
func (c *Client) Emit(ctx context.Context, input EmitInput) (*Event, error) {
	return c.CreateEvent(ctx, CreateEventInput{
		ProjectID:         input.ProjectID,
		Type:              input.Type,
		TypeGroup:         input.TypeGroup,
		EyeduxType:        input.EyeduxType,
		Properties:        input.Properties,
		ExternalObject:    input.ExternalObject,
		CorrelationObject: input.CorrelationObject,
		Metadata:          input.Metadata,
	})
}

// EmitWarning creates a system-warning event.
func (c *Client) EmitWarning(ctx context.Context, input EmitInput) (*Event, error) {
	input.EyeduxType = EventEyeduxTypeSystemWarning
	return c.Emit(ctx, input)
}

// EmitLog creates a system-log event.
func (c *Client) EmitLog(ctx context.Context, input EmitInput) (*Event, error) {
	input.EyeduxType = EventEyeduxTypeSystemLog
	return c.Emit(ctx, input)
}

// EmitDebug creates a system-debug event.
func (c *Client) EmitDebug(ctx context.Context, input EmitInput) (*Event, error) {
	input.EyeduxType = EventEyeduxTypeSystemDebug
	return c.Emit(ctx, input)
}

// EmitInfo creates a system-info event.
func (c *Client) EmitInfo(ctx context.Context, input EmitInput) (*Event, error) {
	input.EyeduxType = EventEyeduxTypeSystemInfo
	return c.Emit(ctx, input)
}

// EmitAudit creates an audit event.
func (c *Client) EmitAudit(ctx context.Context, input EmitInput) (*Event, error) {
	if input.AuditProperties != nil {
		properties, err := input.AuditProperties.ToMap()
		if err != nil {
			return nil, err
		}
		input.Properties = properties
	}
	input.EyeduxType = EventEyeduxTypeAudit
	return c.Emit(ctx, input)
}

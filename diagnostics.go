package eyeduxsdk

import (
	"context"
	"path/filepath"
	"runtime"
	"strings"
)

// ErrorSource identifies the application location that originated an error.
type ErrorSource struct {
	File     string
	Function string
	Line     int
}

// CurrentErrorSource returns the first caller outside the SDK diagnostics helpers.
func CurrentErrorSource() ErrorSource {
	return currentErrorSource(0)
}

func currentErrorSource(sourceSkip int) ErrorSource {
	if sourceSkip < 0 {
		sourceSkip = 0
	}

	pcs := make([]uintptr, 16)
	count := runtime.Callers(2, pcs)
	frames := runtime.CallersFrames(pcs[:count])
	skippedSources := 0

	for {
		frame, more := frames.Next()
		if isErrorDiagnosticHelper(frame.Function) {
			if !more {
				return ErrorSource{}
			}
			continue
		}
		if skippedSources < sourceSkip {
			skippedSources++
			if !more {
				return ErrorSource{}
			}
			continue
		}
		return ErrorSource{
			File:     filepath.Base(frame.File),
			Function: frame.Function,
			Line:     frame.Line,
		}
	}
}

// ErrorProperties enriches properties with error, operation, and source data.
// The input map is copied and is not modified.
func ErrorProperties(properties map[string]any, err error, operation string) map[string]any {
	return errorProperties(properties, err, operation, 0)
}

// ErrorPropertiesWithSourceSkip enriches properties and skips additional
// application frames when identifying the error source.
// The input map is copied and is not modified.
func ErrorPropertiesWithSourceSkip(properties map[string]any, err error, operation string, sourceSkip int) map[string]any {
	return errorProperties(properties, err, operation, sourceSkip)
}

func errorProperties(properties map[string]any, err error, operation string, sourceSkip int) map[string]any {
	enriched := make(map[string]any, len(properties)+5)
	for key, value := range properties {
		enriched[key] = value
	}

	if err != nil {
		enriched["error"] = err.Error()
	}
	if operation != "" {
		enriched["operation"] = operation
	}

	source := currentErrorSource(sourceSkip)
	if source.File != "" {
		enriched["source_file"] = source.File
		enriched["source_line"] = source.Line
		enriched["source_function"] = source.Function
	}

	return enriched
}

// EmitError creates a system-error event with standardized diagnostic properties.
// It does not modify input.Properties or apply a policy to the original error.
func (c *Client) EmitError(ctx context.Context, input EmitInput) (*Event, error) {
	input.Properties = ErrorPropertiesWithSourceSkip(input.Properties, input.Err, input.Operation, input.SourceSkip)
	input.EyeduxType = EventEyeduxTypeSystemError
	return c.Emit(ctx, input)
}

func isErrorDiagnosticHelper(functionName string) bool {
	return strings.HasSuffix(functionName, ".CurrentErrorSource") ||
		strings.HasSuffix(functionName, ".ErrorProperties") ||
		strings.HasSuffix(functionName, ".ErrorPropertiesWithSourceSkip") ||
		strings.HasSuffix(functionName, ".errorProperties") ||
		strings.HasSuffix(functionName, ").EmitError")
}

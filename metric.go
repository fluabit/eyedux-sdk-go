package eyeduxsdk

import (
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strings"
)

// MetricUnit identifies the unit of a metric value.
type MetricUnit string

const (
	MetricUnitCount        MetricUnit = "count"
	MetricUnitBytes        MetricUnit = "bytes"
	MetricUnitMilliseconds MetricUnit = "milliseconds"
	MetricUnitSeconds      MetricUnit = "seconds"
	MetricUnitPercent      MetricUnit = "percent"
	MetricUnitRatio        MetricUnit = "ratio"
)

const (
	metricDimensionKeyMax    = 64
	metricDimensionValueMax  = 128
	metricDimensionCountMax  = 10
	metricDimensionsMaxBytes = 2048
)

var metricDimensionKeyPattern = regexp.MustCompile(`^[a-z][a-z0-9_.-]*$`)

// MetricProperties is the structured properties payload for a metric event.
type MetricProperties struct {
	Value      float64           `json:"value"`
	Unit       MetricUnit        `json:"unit"`
	Dimensions map[string]string `json:"dimensions,omitempty"`
}

// Validate checks the structural rules required by the metric contract.
func (p MetricProperties) Validate() error {
	if math.IsNaN(p.Value) || math.IsInf(p.Value, 0) {
		return fmt.Errorf("eyedux: metric value must be a finite number")
	}
	if err := validateMetricUnit(p.Unit, p.Value); err != nil {
		return err
	}
	return validateMetricDimensions(p.Dimensions)
}

func validateMetricUnit(unit MetricUnit, value float64) error {
	switch unit {
	case MetricUnitCount, MetricUnitBytes, MetricUnitMilliseconds, MetricUnitSeconds:
		return nil
	case MetricUnitPercent:
		if value < 0 || value > 100 {
			return fmt.Errorf("eyedux: metric value must be between 0 and 100 for unit %q", unit)
		}
		return nil
	case MetricUnitRatio:
		if value < 0 || value > 1 {
			return fmt.Errorf("eyedux: metric value must be between 0 and 1 for unit %q", unit)
		}
		return nil
	case "":
		return fmt.Errorf("eyedux: metric unit is required")
	default:
		return fmt.Errorf("eyedux: invalid metric unit %q", unit)
	}
}

func validateMetricDimensions(dimensions map[string]string) error {
	if dimensions == nil {
		return nil
	}
	if len(dimensions) > metricDimensionCountMax {
		return fmt.Errorf("eyedux: metric dimensions must contain at most %d entries", metricDimensionCountMax)
	}

	for key, value := range dimensions {
		if key == "" || len(key) > metricDimensionKeyMax || strings.HasPrefix(key, "__") || !metricDimensionKeyPattern.MatchString(key) {
			return fmt.Errorf("eyedux: metric dimension key %q is invalid", key)
		}
		if len(value) > metricDimensionValueMax {
			return fmt.Errorf("eyedux: metric dimension %q value must be at most %d bytes", key, metricDimensionValueMax)
		}
	}

	encoded, err := json.Marshal(dimensions)
	if err != nil || len(encoded) > metricDimensionsMaxBytes {
		return fmt.Errorf("eyedux: metric dimensions must be at most %d bytes", metricDimensionsMaxBytes)
	}
	return nil
}

// ToMap validates the model and converts it to CreateEventInput.Properties.
func (p MetricProperties) ToMap() (map[string]any, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}

	properties := map[string]any{
		"value": p.Value,
		"unit":  p.Unit,
	}
	if p.Dimensions != nil {
		properties["dimensions"] = p.Dimensions
	}

	return properties, nil
}

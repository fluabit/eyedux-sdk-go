package eyeduxsdk

import (
	"encoding/json"
	"math"
	"strings"
	"testing"
)

func TestMetricPropertiesToMap(t *testing.T) {
	properties, err := (MetricProperties{
		Value: 120.5,
		Unit:  MetricUnitMilliseconds,
		Dimensions: map[string]string{
			"route":  "/checkout",
			"region": "br-sp",
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
	if got["value"] != 120.5 {
		t.Errorf("value = %v, want 120.5", got["value"])
	}
	if got["unit"] != "milliseconds" {
		t.Errorf("unit = %v, want milliseconds", got["unit"])
	}
	dimensions := got["dimensions"].(map[string]any)
	if dimensions["route"] != "/checkout" {
		t.Errorf("dimensions.route = %v, want /checkout", dimensions["route"])
	}
}

func TestMetricPropertiesToMap_omitsDimensionsWhenNil(t *testing.T) {
	properties, err := (MetricProperties{Value: 10, Unit: MetricUnitCount}).ToMap()
	if err != nil {
		t.Fatalf("ToMap: %v", err)
	}
	if _, exists := properties["dimensions"]; exists {
		t.Errorf("dimensions = %v, want absent", properties["dimensions"])
	}
}

func TestMetricPropertiesToMap_acceptsBoundaryValuesForPercentAndRatio(t *testing.T) {
	if _, err := (MetricProperties{Value: 0, Unit: MetricUnitPercent}).ToMap(); err != nil {
		t.Errorf("percent lower bound: %v", err)
	}
	if _, err := (MetricProperties{Value: 100, Unit: MetricUnitPercent}).ToMap(); err != nil {
		t.Errorf("percent upper bound: %v", err)
	}
	if _, err := (MetricProperties{Value: 0, Unit: MetricUnitRatio}).ToMap(); err != nil {
		t.Errorf("ratio lower bound: %v", err)
	}
	if _, err := (MetricProperties{Value: 1, Unit: MetricUnitRatio}).ToMap(); err != nil {
		t.Errorf("ratio upper bound: %v", err)
	}
}

func TestMetricPropertiesToMap_rejectsInvalidProperties(t *testing.T) {
	tests := []struct {
		name       string
		properties MetricProperties
	}{
		{
			name:       "missing unit",
			properties: MetricProperties{Value: 10},
		},
		{
			name:       "invalid unit",
			properties: MetricProperties{Value: 10, Unit: "requests"},
		},
		{
			name:       "percent above 100",
			properties: MetricProperties{Value: 150, Unit: MetricUnitPercent},
		},
		{
			name:       "percent below 0",
			properties: MetricProperties{Value: -1, Unit: MetricUnitPercent},
		},
		{
			name:       "ratio above 1",
			properties: MetricProperties{Value: 1.5, Unit: MetricUnitRatio},
		},
		{
			name:       "ratio below 0",
			properties: MetricProperties{Value: -0.1, Unit: MetricUnitRatio},
		},
		{
			name:       "non-finite value",
			properties: MetricProperties{Value: math.NaN(), Unit: MetricUnitCount},
		},
		{
			name: "too many dimensions",
			properties: MetricProperties{
				Value:      1,
				Unit:       MetricUnitCount,
				Dimensions: elevenDimensions(),
			},
		},
		{
			name:       "invalid dimension key",
			properties: MetricProperties{Value: 1, Unit: MetricUnitCount, Dimensions: map[string]string{"Route": "x"}},
		},
		{
			name:       "reserved dimension key prefix",
			properties: MetricProperties{Value: 1, Unit: MetricUnitCount, Dimensions: map[string]string{"__internal": "x"}},
		},
		{
			name:       "dimension value too long",
			properties: MetricProperties{Value: 1, Unit: MetricUnitCount, Dimensions: map[string]string{"route": strings.Repeat("a", 129)}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := test.properties.ToMap(); err == nil {
				t.Error("ToMap must reject invalid metric properties")
			}
		})
	}
}

func elevenDimensions() map[string]string {
	dimensions := make(map[string]string, 11)
	for i := 0; i < 11; i++ {
		dimensions[string(rune('a'+i))] = "x"
	}
	return dimensions
}

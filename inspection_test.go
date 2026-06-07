package opskit

import (
	"encoding/json"
	"testing"
)

func TestInspectionJSONOmitEmptyFields(t *testing.T) {
	requireJSON(t, Inspection{}, `{}`)
}

func TestInspectionJSONIncludesSummaryDetailsAndAttributes(t *testing.T) {
	inspection := Inspection{
		Summary: "cache online",
		Details: map[string]any{
			"entries": float64(42),
			"mode":    "write-through",
		},
		Attributes: []Attribute{
			Attr("shard", "primary"),
		},
	}

	requireJSON(t, inspection, `{"summary":"cache online","details":{"entries":42,"mode":"write-through"},"attributes":[{"key":"shard","value":"primary"}]}`)
}

func TestInspectionJSONRoundTripArbitraryPayloads(t *testing.T) {
	input := []byte(`{"summary":{"state":"ok"},"details":["one","two"],"attributes":[{"key":"component","value":"cache"}]}`)

	var inspection Inspection
	if err := json.Unmarshal(input, &inspection); err != nil {
		t.Fatalf("Unmarshal Inspection error = %v", err)
	}

	summary, ok := inspection.Summary.(map[string]any)
	if !ok {
		t.Fatalf("Summary type = %T, want map[string]any", inspection.Summary)
	}
	if summary["state"] != "ok" {
		t.Fatalf("Summary[state] = %v, want ok", summary["state"])
	}

	details, ok := inspection.Details.([]any)
	if !ok {
		t.Fatalf("Details type = %T, want []any", inspection.Details)
	}
	if len(details) != 2 || details[0] != "one" || details[1] != "two" {
		t.Fatalf("Details = %+v, want [one two]", details)
	}

	if len(inspection.Attributes) != 1 {
		t.Fatalf("Attributes length = %d, want 1", len(inspection.Attributes))
	}
	if inspection.Attributes[0] != Attr("component", "cache") {
		t.Fatalf("Attributes[0] = %+v, want component cache", inspection.Attributes[0])
	}
}

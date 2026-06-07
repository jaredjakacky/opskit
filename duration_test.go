package opskit

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestNewDuration(t *testing.T) {
	duration := 150*time.Millisecond + 25*time.Microsecond

	got := NewDuration(duration)
	if got.TimeDuration() != duration {
		t.Fatalf("TimeDuration = %v, want %v", got.TimeDuration(), duration)
	}
}

func TestDurationString(t *testing.T) {
	duration := NewDuration(90 * time.Second)

	if duration.String() != "1m30s" {
		t.Fatalf("String() = %q, want 1m30s", duration.String())
	}
}

func TestDurationMarshalJSON(t *testing.T) {
	requireJSON(t, NewDuration(150*time.Millisecond), `"150ms"`)
}

func TestDurationUnmarshalJSON(t *testing.T) {
	var duration Duration
	if err := json.Unmarshal([]byte(`"1m30s"`), &duration); err != nil {
		t.Fatalf("Unmarshal Duration error = %v", err)
	}

	if duration.TimeDuration() != 90*time.Second {
		t.Fatalf("TimeDuration = %v, want 1m30s", duration.TimeDuration())
	}
}

func TestDurationUnmarshalJSONRejectsNonString(t *testing.T) {
	var duration Duration
	err := json.Unmarshal([]byte(`150`), &duration)
	if err == nil {
		t.Fatal("Unmarshal Duration error = nil, want error")
	}
	if !strings.Contains(err.Error(), "opskit: decode duration") {
		t.Fatalf("Unmarshal Duration error = %q, want decode duration context", err.Error())
	}
}

func TestDurationUnmarshalJSONRejectsInvalidDuration(t *testing.T) {
	var duration Duration
	err := json.Unmarshal([]byte(`"not-a-duration"`), &duration)
	if err == nil {
		t.Fatal("Unmarshal Duration error = nil, want error")
	}
	if !strings.Contains(err.Error(), `opskit: parse duration "not-a-duration"`) {
		t.Fatalf("Unmarshal Duration error = %q, want parse duration context", err.Error())
	}
}

func TestNowUTC(t *testing.T) {
	before := time.Now().UTC()
	got := nowUTC()
	after := time.Now().UTC()

	if got == nil {
		t.Fatal("nowUTC returned nil")
	}
	if got.Location() != time.UTC {
		t.Fatalf("Location = %q, want UTC", got.Location())
	}
	if got.Before(before) || got.After(after) {
		t.Fatalf("nowUTC = %v, want between %v and %v", got, before, after)
	}
}

func TestTimeUTCAt(t *testing.T) {
	location := time.FixedZone("test", -5*60*60)
	input := time.Date(2026, 6, 4, 7, 30, 0, 0, location)

	got := timeUTCAt(input)
	if got == nil {
		t.Fatal("timeUTCAt returned nil")
	}
	if got.Location() != time.UTC {
		t.Fatalf("Location = %q, want UTC", got.Location())
	}
	if !got.Equal(input) {
		t.Fatalf("timeUTCAt = %v, want instant equal to %v", got, input)
	}
	if got.Hour() != 12 {
		t.Fatalf("Hour = %d, want 12 after UTC conversion", got.Hour())
	}
}

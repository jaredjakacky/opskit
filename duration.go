package opskit

import (
	"encoding/json"
	"fmt"
	"time"
)

// Duration is a JSON-friendly wrapper around time.Duration.
//
// It marshals as a Go duration string, such as "10ms", "2s", or "1m30s",
// instead of the standard library's raw nanosecond integer representation.
type Duration time.Duration

// NewDuration converts a time.Duration to an Opskit Duration.
func NewDuration(duration time.Duration) Duration {
	return Duration(duration)
}

// TimeDuration converts an Opskit Duration back to time.Duration.
func (d Duration) TimeDuration() time.Duration {
	return time.Duration(d)
}

// String returns the duration using time.Duration string formatting.
func (d Duration) String() string {
	return time.Duration(d).String()
}

// MarshalJSON encodes the duration as a string.
func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

// UnmarshalJSON decodes a duration string.
func (d *Duration) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf("opskit: decode duration: %w", err)
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		return fmt.Errorf("opskit: parse duration %q: %w", value, err)
	}

	*d = Duration(duration)
	return nil
}

func nowUTC() *time.Time {
	now := time.Now().UTC()
	return &now
}

func timeUTCAt(t time.Time) *time.Time {
	utc := t.UTC()
	return &utc
}

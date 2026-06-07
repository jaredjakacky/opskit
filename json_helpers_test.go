package opskit

import (
	"encoding/json"
	"testing"
)

func requireJSON(t *testing.T, value any, want string) {
	t.Helper()

	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("Marshal %T error = %v", value, err)
	}

	if string(data) != want {
		t.Fatalf("Marshal %T = %s, want %s", value, data, want)
	}
}

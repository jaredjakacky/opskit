package opskit

import (
	"encoding/json"
	"testing"
)

func TestFailureJSON(t *testing.T) {
	data, err := json.Marshal(Failure{Code: "timeout", Message: "operation timed out"})
	if err != nil {
		t.Fatalf("Marshal Failure error = %v", err)
	}
	const want = `{"code":"timeout","message":"operation timed out"}`
	if string(data) != want {
		t.Fatalf("Marshal Failure = %s, want %s", data, want)
	}
}

func TestFailurePtrOmitsZeroFailure(t *testing.T) {
	if got := failurePtr(Failure{}); got != nil {
		t.Fatalf("failurePtr zero = %+v, want nil", got)
	}
}

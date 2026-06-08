package opskit

import (
	"errors"
	"testing"
)

func TestErrorSentinels(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{name: "nil component", err: ErrNilComponent, want: "opskit: nil component"},
		{name: "empty component name", err: ErrEmptyComponentName, want: "opskit: component name is required"},
		{name: "invalid component name", err: ErrInvalidComponentName, want: "opskit: component name is invalid"},
		{name: "duplicate component", err: ErrDuplicateComponent, want: "opskit: component already registered"},
		{name: "component not found", err: ErrComponentNotFound, want: "opskit: component not found"},
		{name: "inspection unsupported", err: ErrInspectionUnsupported, want: "opskit: component does not support inspection"},
		{name: "checker unsupported", err: ErrCheckerUnsupported, want: "opskit: component does not support checks"},
		{name: "check group unsupported", err: ErrCheckGroupUnsupported, want: "opskit: component does not support grouped checks"},
		{name: "command handler unsupported", err: ErrCommandHandlerUnsupported, want: "opskit: component does not support commands"},
		{name: "command describer unsupported", err: ErrCommandDescriberUnsupported, want: "opskit: component does not describe commands"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil {
				t.Fatal("error is nil")
			}
			if tt.err.Error() != tt.want {
				t.Fatalf("Error() = %q, want %q", tt.err.Error(), tt.want)
			}
			if !errors.Is(tt.err, tt.err) {
				t.Fatalf("errors.Is(%v, %v) = false, want true", tt.err, tt.err)
			}
		})
	}
}

func TestErrorSentinelsAreDistinct(t *testing.T) {
	errs := []error{
		ErrNilComponent,
		ErrEmptyComponentName,
		ErrInvalidComponentName,
		ErrDuplicateComponent,
		ErrComponentNotFound,
		ErrInspectionUnsupported,
		ErrCheckerUnsupported,
		ErrCheckGroupUnsupported,
		ErrCommandHandlerUnsupported,
		ErrCommandDescriberUnsupported,
	}

	for i, left := range errs {
		for j, right := range errs {
			if i == j {
				continue
			}
			if errors.Is(left, right) {
				t.Fatalf("errors.Is(%v, %v) = true, want false", left, right)
			}
		}
	}
}

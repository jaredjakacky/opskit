package opskit

import (
	"context"
	"strings"
)

const (
	componentStatusPanicMessage       = "component status evaluation panicked"
	componentReadinessPanicMessage    = "component readiness evaluation panicked"
	componentInspectionFailureMessage = "component inspection unavailable"
)

func componentInspectionFailure() *Failure {
	return failurePtr(Failure{
		Code:    FailureCodeInspectionFailed,
		Message: componentInspectionFailureMessage,
	})
}

func (r *Registry) registration(name string) (registration, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	reg, ok := r.registrations[name]
	reg = cloneRegistration(reg)
	return reg, ok
}

func (r *Registry) snapshot() []registration {
	r.mu.RLock()
	defer r.mu.RUnlock()

	registrations := make([]registration, 0, len(r.order))
	for _, name := range r.order {
		registrations = append(registrations, cloneRegistration(r.registrations[name]))
	}

	return registrations
}

func cloneRegistration(reg registration) registration {
	reg.info = cloneComponentInfo(reg.info)
	return reg
}

func (r *Registry) ensureInitializedLocked() {
	if r.registrations == nil {
		r.registrations = make(map[string]registration)
	}
}

// ValidateComponentName verifies that name is a stable, path-safe component
// name.
func ValidateComponentName(name string) error {
	if strings.TrimSpace(name) == "" {
		return ErrEmptyComponentName
	}

	if name != strings.TrimSpace(name) {
		return ErrInvalidComponentName
	}

	if name == "." || name == ".." {
		return ErrInvalidComponentName
	}

	for _, ch := range name {
		switch {
		case ch >= 'A' && ch <= 'Z':
		case ch >= 'a' && ch <= 'z':
		case ch >= '0' && ch <= '9':
		case ch == '.', ch == '_', ch == '-':
		default:
			return ErrInvalidComponentName
		}
	}

	return nil
}

// IsValidComponentName reports whether name is a stable, path-safe component
// name.
func IsValidComponentName(name string) bool {
	return ValidateComponentName(name) == nil
}

func capabilitiesOf(component Component) ComponentCapabilities {
	_, readinessContributor := component.(ReadinessContributor)
	_, inspector := component.(Inspector)
	_, checker := component.(Checker)
	_, checkDescriber := component.(CheckDescriber)
	_, checkGroup := component.(CheckGroup)
	_, commandHandler := component.(CommandHandler)
	_, commandDescriber := component.(CommandDescriber)

	return ComponentCapabilities{
		ReadinessContributor: readinessContributor,
		Inspector:            inspector,
		Checker:              checker,
		CheckDescriber:       checkDescriber,
		CheckGroup:           checkGroup,
		CommandHandler:       commandHandler,
		CommandDescriber:     commandDescriber,
	}
}

func normalizeReadinessPolicy(policy ReadinessPolicy) ReadinessPolicy {
	switch policy {
	case ReadinessRequired, ReadinessOptional, ReadinessInformational:
		return policy
	default:
		return ReadinessRequired
	}
}

func participatesInReadiness(policy ReadinessPolicy) bool {
	return normalizeReadinessPolicy(policy) != ReadinessInformational
}

func blocksReadiness(policy ReadinessPolicy) bool {
	return normalizeReadinessPolicy(policy) == ReadinessRequired
}

func normalizeReadiness(readiness Readiness) Readiness {
	readiness.Items = normalizeReadinessItems(readiness.Items)
	return readiness
}

func componentReadiness(info ComponentInfo, policy ReadinessPolicy, readiness Readiness) ComponentReadiness {
	return ComponentReadiness{
		Component: info,
		Registration: ComponentRegistration{
			ReadinessPolicy: policy,
		},
		Readiness: normalizeReadiness(readiness),
	}
}

func panickedReadiness(info ComponentInfo, reason string) Readiness {
	return Readiness{
		Ready:  false,
		Reason: reason,
		Items: []ReadinessItem{
			{
				Name:   info.Name,
				Kind:   info.Kind,
				Impact: ReadinessImpactBlocking,
				Ready:  false,
				State:  StateUnknown,
				Reason: reason,
			},
		},
	}
}

func safeComponentInfo(component Component) (info ComponentInfo, err error) {
	defer func() {
		if recover() != nil {
			info = ComponentInfo{}
			err = ErrComponentPanicked
		}
	}()

	return component.ComponentInfo(), nil
}

// The safeComponent helpers recover component panics only. Their Registry
// callers enforce the context acceptance boundary after each helper returns.
func safeComponentStatus(component Component, ctx context.Context) (status Status, panicked bool) {
	defer func() {
		if recover() != nil {
			status = UnknownStatus(componentStatusPanicMessage)
			panicked = true
		}
	}()

	return component.Status(ctx), false
}

func safeComponentReadiness(contributor ReadinessContributor, ctx context.Context, info ComponentInfo) (readiness Readiness, panicked bool) {
	defer func() {
		if recover() != nil {
			readiness = panickedReadiness(info, componentReadinessPanicMessage)
			panicked = true
		}
	}()

	return normalizeReadiness(contributor.Readiness(ctx)), false
}

func safeComponentInspection(inspector Inspector, ctx context.Context) (inspection Inspection, err error, panicked bool) {
	defer func() {
		if recover() != nil {
			err = ErrComponentPanicked
			panicked = true
		}
	}()

	inspection, err = inspector.Inspect(ctx)
	return inspection, err, false
}

func safeComponentChecks(describer CheckDescriber, ctx context.Context) (checks []CheckDescriptor, err error, panicked bool) {
	defer func() {
		if recover() != nil {
			err = ErrComponentPanicked
			panicked = true
		}
	}()

	return describer.Checks(ctx), nil, false
}

func safeComponentCommands(describer CommandDescriber, ctx context.Context) (commands []CommandDescriptor, err error, panicked bool) {
	defer func() {
		if recover() != nil {
			err = ErrComponentPanicked
			panicked = true
		}
	}()

	return describer.Commands(ctx), nil, false
}

func normalizeReadinessItemState(ready bool, state State) State {
	if state != "" {
		return state
	}

	return stateFromReady(ready)
}

func stateFromReady(ready bool) State {
	if ready {
		return StateReady
	}

	return StateNotReady
}

func canceledComponentStatus(err error) ComponentStatus {
	message := contextFailureMessage(err)

	return ComponentStatus{
		Component: ComponentInfo{
			Name: "opskit.registry",
			Kind: "opskit",
		},
		Status: Status{
			State:   StateUnknown,
			Ready:   false,
			Message: "status evaluation canceled",
			Attributes: []Attribute{
				Attr("error", message),
			},
		},
	}
}

func canceledReadinessItem(err error) ReadinessItem {
	message := contextFailureMessage(err)

	return ReadinessItem{
		Name:    "opskit.registry",
		Kind:    "opskit",
		Impact:  ReadinessImpactBlocking,
		Ready:   false,
		State:   StateUnknown,
		Reason:  "readiness evaluation canceled",
		Message: message,
	}
}

func canceledRegistryReadiness(readiness SystemReadiness, err error) SystemReadiness {
	readiness.Ready = false
	readiness.Reason = "readiness evaluation canceled"
	readiness.Components = append(readiness.Components, ComponentReadiness{
		Component: ComponentInfo{
			Name: "opskit.registry",
			Kind: "opskit",
		},
		Registration: ComponentRegistration{
			ReadinessPolicy: ReadinessRequired,
		},
		Readiness: Readiness{
			Ready:  false,
			Reason: "readiness evaluation canceled",
			Items:  []ReadinessItem{canceledReadinessItem(err)},
		},
	})
	return readiness
}

// contextFailureMessage deliberately classifies rather than formats err. The
// callers pass context.Context.Err results, but keeping this boundary closed
// prevents a future internal caller from publishing arbitrary error text.
func contextFailureMessage(err error) string {
	if err == context.DeadlineExceeded {
		return "context deadline exceeded"
	}

	return "context canceled"
}

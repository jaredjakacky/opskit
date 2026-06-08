package opskit

import (
	"context"
	"strings"
)

const (
	componentStatusPanicMessage     = "component status evaluation panicked"
	componentReadinessPanicMessage  = "component readiness evaluation panicked"
	componentInspectionPanicMessage = "component inspection evaluation panicked"
)

func (r *Registry) registration(name string) (registration, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	reg, ok := r.registrations[name]
	return reg, ok
}

func (r *Registry) snapshot() []registration {
	r.mu.RLock()
	defer r.mu.RUnlock()

	registrations := make([]registration, 0, len(r.order))
	for _, name := range r.order {
		registrations = append(registrations, r.registrations[name])
	}

	return registrations
}

func (r *Registry) ensureInitializedLocked() {
	if r.registrations == nil {
		r.registrations = make(map[string]registration)
	}
}

func isValidComponentName(name string) bool {
	if name == "" {
		return false
	}

	if name != strings.TrimSpace(name) {
		return false
	}

	if name == "." || name == ".." {
		return false
	}

	for _, ch := range name {
		switch {
		case ch >= 'A' && ch <= 'Z':
		case ch >= 'a' && ch <= 'z':
		case ch >= '0' && ch <= '9':
		case ch == '.', ch == '_', ch == '-':
		default:
			return false
		}
	}

	return true
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

func readinessItemFromReadiness(info ComponentInfo, readiness Readiness, policy ReadinessPolicy) []ReadinessItem {
	if len(readiness.Components) == 0 {
		return []ReadinessItem{
			{
				Name:   info.Name,
				Kind:   info.Kind,
				Policy: policy,
				Ready:  readiness.Ready,
				State:  stateFromReady(readiness.Ready),
				Reason: readiness.Reason,
			},
		}
	}

	items := make([]ReadinessItem, 0, len(readiness.Components))
	for _, item := range readiness.Components {
		item.State = normalizeReadinessItemState(item.Ready, item.State)
		if item.Name == "" {
			item.Name = info.Name
		}
		if item.Kind == "" {
			item.Kind = info.Kind
		}
		if item.Policy == "" {
			item.Policy = policy
		}
		items = append(items, item)
	}

	return items
}

func readinessWithPolicy(info ComponentInfo, readiness Readiness, policy ReadinessPolicy) Readiness {
	readiness.Components = readinessItemFromReadiness(info, readiness, policy)
	return readiness
}

func readinessFromStatusWithPolicy(info ComponentInfo, status Status, policy ReadinessPolicy) Readiness {
	return readinessWithPolicy(info, ReadinessFromStatus(info, status), policy)
}

func panickedReadiness(info ComponentInfo, policy ReadinessPolicy, reason string) Readiness {
	return Readiness{
		Ready:  false,
		Reason: reason,
		Components: []ReadinessItem{
			{
				Name:   info.Name,
				Kind:   info.Kind,
				Policy: policy,
				Ready:  false,
				State:  StateUnknown,
				Reason: reason,
			},
		},
	}
}

func safeComponentStatus(component Component, ctx context.Context) (status Status, panicked bool) {
	defer func() {
		if recover() != nil {
			status = UnknownStatus(componentStatusPanicMessage)
			panicked = true
		}
	}()

	return component.Status(ctx), false
}

func safeComponentReadiness(contributor ReadinessContributor, ctx context.Context, info ComponentInfo, policy ReadinessPolicy) (readiness Readiness, panicked bool) {
	defer func() {
		if recover() != nil {
			readiness = panickedReadiness(info, policy, componentReadinessPanicMessage)
			panicked = true
		}
	}()

	return readinessWithPolicy(info, contributor.Readiness(ctx), policy), false
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
				Attr("error", err.Error()),
			},
		},
	}
}

func canceledReadinessItem(err error) ReadinessItem {
	return ReadinessItem{
		Name:    "opskit.registry",
		Kind:    "opskit",
		Ready:   false,
		State:   StateUnknown,
		Reason:  "readiness evaluation canceled",
		Message: err.Error(),
	}
}

package opskit

import (
	"context"
	"strings"
	"sync"
)

// Registry stores operational components.
//
// Registry is intentionally passive. It does not run checks, serve HTTP,
// dispatch commands, or own lifecycle. It only stores components and provides
// a common read model for other kits.
//
// Registry read methods call component methods synchronously, so callers that
// expose registry data through probes or admin endpoints should pass bounded
// contexts.
//
// The zero value is ready to use.
type Registry struct {
	mu            sync.RWMutex
	registrations map[string]registration
	order         []string
}

type registration struct {
	component       Component
	info            ComponentInfo
	readinessPolicy ReadinessPolicy
}

// RegisterOption configures how a component is registered.
//
// Register options are intentionally implemented only by this package so the
// registration policy surface stays stable.
type RegisterOption interface {
	applyRegisterOption(*registration)
}

type registerOptionFunc func(*registration)

func (fn registerOptionFunc) applyRegisterOption(reg *registration) {
	fn(reg)
}

// WithReadinessPolicy configures how the component participates in readiness.
//
// Unknown policies are treated as ReadinessRequired so an invalid option cannot
// silently remove a component from aggregate readiness.
func WithReadinessPolicy(policy ReadinessPolicy) RegisterOption {
	return registerOptionFunc(func(reg *registration) {
		reg.readinessPolicy = normalizeReadinessPolicy(policy)
	})
}

// Required makes a component block aggregate readiness when it is not ready.
func Required() RegisterOption {
	return WithReadinessPolicy(ReadinessRequired)
}

// Optional makes a component visible in readiness details without allowing it to
// block aggregate readiness.
func Optional() RegisterOption {
	return WithReadinessPolicy(ReadinessOptional)
}

// Informational omits a component from readiness. The component remains visible
// through status, inspection, and admin snapshots.
func Informational() RegisterOption {
	return WithReadinessPolicy(ReadinessInformational)
}

// NewRegistry creates an empty Registry.
func NewRegistry() *Registry {
	return &Registry{}
}

// Register adds a component to the registry.
func (r *Registry) Register(component Component, opts ...RegisterOption) error {
	if component == nil {
		return ErrNilComponent
	}

	info := component.ComponentInfo()
	if strings.TrimSpace(info.Name) == "" {
		return ErrEmptyComponentName
	}
	if !isValidComponentName(info.Name) {
		return ErrInvalidComponentName
	}

	reg := registration{
		component:       component,
		info:            info,
		readinessPolicy: ReadinessRequired,
	}

	for _, opt := range opts {
		if opt != nil {
			opt.applyRegisterOption(&reg)
		}
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.ensureInitializedLocked()

	if _, exists := r.registrations[info.Name]; exists {
		return ErrDuplicateComponent
	}

	r.registrations[info.Name] = reg
	r.order = append(r.order, info.Name)

	return nil
}

// MustRegister adds a component to the registry and panics on error.
func (r *Registry) MustRegister(component Component, opts ...RegisterOption) {
	if err := r.Register(component, opts...); err != nil {
		panic(err)
	}
}

// Component returns a registered component by name.
func (r *Registry) Component(name string) (Component, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	reg, ok := r.registrations[name]
	if !ok {
		return nil, false
	}

	return reg.component, true
}

// Components returns all registered components in registration order.
func (r *Registry) Components() []Component {
	registrations := r.snapshot()

	components := make([]Component, 0, len(registrations))
	for _, reg := range registrations {
		components = append(components, reg.component)
	}

	return components
}

// Status returns the status of all registered components.
//
// Status calls each component's Status method synchronously. Use a context with
// an appropriate deadline when serving request paths. If evaluation is
// canceled, Status returns a synthetic opskit.registry entry that describes the
// cancellation. If a component status method panics, Status recovers and
// returns an unknown status for that component without exposing the panic value.
func (r *Registry) Status(ctx context.Context) SystemStatus {
	ctx = normalizeContext(ctx)

	registrations := r.snapshot()

	system := SystemStatus{
		Components: make([]ComponentStatus, 0, len(registrations)),
	}

	for _, reg := range registrations {
		if err := ctx.Err(); err != nil {
			system.Components = append(system.Components, canceledComponentStatus(err))
			return system
		}

		component := reg.component
		status, _ := safeComponentStatus(component, ctx)

		system.Components = append(system.Components, ComponentStatus{
			Component: reg.info,
			Registration: ComponentRegistration{
				ReadinessPolicy: reg.readinessPolicy,
			},
			Capabilities: capabilitiesOf(component),
			Status:       status,
		})
	}

	return system
}

// Readiness returns aggregate readiness for required readiness components.
//
// Optional components appear in readiness details, but do not block aggregate
// readiness. Informational components remain visible through Status and Inspect,
// but are omitted from readiness.
//
// Readiness calls readiness contributors and component status methods
// synchronously. Use a context with an appropriate deadline for probe paths. If
// evaluation is canceled, Readiness includes a synthetic opskit.registry item
// that describes the cancellation. If a component readiness or status method
// panics, Readiness recovers and returns an unknown not-ready item without
// exposing the panic value.
func (r *Registry) Readiness(ctx context.Context) Readiness {
	ctx = normalizeContext(ctx)

	registrations := r.snapshot()

	readiness := Readiness{
		Ready:      true,
		Components: make([]ReadinessItem, 0, len(registrations)),
	}

	required := 0

	for _, reg := range registrations {
		if err := ctx.Err(); err != nil {
			readiness.Ready = false
			readiness.Reason = "readiness evaluation canceled"
			readiness.Components = append(readiness.Components, canceledReadinessItem(err))
			return readiness
		}

		if !participatesInReadiness(reg.readinessPolicy) {
			continue
		}

		component := reg.component

		if contributor, ok := component.(ReadinessContributor); ok {
			componentReadiness, _ := safeComponentReadiness(contributor, ctx, reg.info, reg.readinessPolicy)

			if blocksReadiness(reg.readinessPolicy) {
				required++
			}

			if blocksReadiness(reg.readinessPolicy) && !componentReadiness.Ready {
				readiness.Ready = false
			}

			readiness.Components = append(readiness.Components, componentReadiness.Components...)
			continue
		}

		status, panicked := safeComponentStatus(component, ctx)
		componentReadiness := readinessFromStatusWithPolicy(reg.info, status, reg.readinessPolicy)
		if panicked {
			componentReadiness = panickedReadiness(reg.info, reg.readinessPolicy, componentStatusPanicMessage)
		}

		if blocksReadiness(reg.readinessPolicy) {
			required++
		}

		if blocksReadiness(reg.readinessPolicy) && !status.Ready {
			readiness.Ready = false
		}

		readiness.Components = append(readiness.Components, componentReadiness.Components...)
	}

	if required == 0 {
		readiness.Ready = false
		readiness.Reason = "no required readiness components registered"
		return readiness
	}

	if readiness.Ready {
		readiness.Reason = "all readiness components ready"
	} else {
		readiness.Reason = "one or more readiness components are not ready"
	}

	return readiness
}

// Snapshot returns the operational snapshot for one registered component.
//
// Snapshot calls component status, readiness, and inspection methods
// synchronously when those capabilities are available. Use a context with an
// appropriate deadline when serving admin request paths. If a component method
// panics, Snapshot recovers and returns partial safe operational data without
// exposing the panic value.
func (r *Registry) Snapshot(ctx context.Context, name string) (ComponentSnapshot, error) {
	ctx = normalizeContext(ctx)

	reg, ok := r.registration(name)
	if !ok {
		return ComponentSnapshot{}, ErrComponentNotFound
	}

	if err := ctx.Err(); err != nil {
		return ComponentSnapshot{}, err
	}

	component := reg.component
	status, statusPanicked := safeComponentStatus(component, ctx)

	snapshot := ComponentSnapshot{
		Component: reg.info,
		Registration: ComponentRegistration{
			ReadinessPolicy: reg.readinessPolicy,
		},
		Capabilities: capabilitiesOf(component),
		Status:       status,
	}

	if err := ctx.Err(); err != nil {
		return ComponentSnapshot{}, err
	}

	if participatesInReadiness(reg.readinessPolicy) {
		if contributor, ok := component.(ReadinessContributor); ok {
			readiness, _ := safeComponentReadiness(contributor, ctx, reg.info, reg.readinessPolicy)
			snapshot.Readiness = &readiness
		} else if statusPanicked {
			readiness := panickedReadiness(reg.info, reg.readinessPolicy, componentStatusPanicMessage)
			snapshot.Readiness = &readiness
		} else {
			readiness := readinessFromStatusWithPolicy(reg.info, snapshot.Status, reg.readinessPolicy)
			snapshot.Readiness = &readiness
		}

		if err := ctx.Err(); err != nil {
			return ComponentSnapshot{}, err
		}
	}

	if inspector, ok := component.(Inspector); ok {
		inspection, err, panicked := safeComponentInspection(inspector, ctx)
		if panicked {
			snapshot.InspectionError = componentInspectionPanicMessage
			return snapshot, nil
		}
		if err != nil {
			snapshot.InspectionError = err.Error()
			return snapshot, nil
		}
		snapshot.Inspection = &inspection
	}

	return snapshot, nil
}

// Inspect returns safe operational inspection data for one registered component.
//
// Inspect calls the component inspector synchronously. Use a context with an
// appropriate deadline when serving admin request paths. If the inspector
// panics, Inspect recovers and returns ErrComponentPanicked without exposing the
// panic value.
func (r *Registry) Inspect(ctx context.Context, name string) (Inspection, error) {
	ctx = normalizeContext(ctx)

	component, ok := r.Component(name)
	if !ok {
		return Inspection{}, ErrComponentNotFound
	}

	if err := ctx.Err(); err != nil {
		return Inspection{}, err
	}

	inspector, ok := component.(Inspector)
	if !ok {
		return Inspection{}, ErrInspectionUnsupported
	}

	inspection, err, _ := safeComponentInspection(inspector, ctx)
	return inspection, err
}

// Checker returns a registered component as a Checker.
func (r *Registry) Checker(name string) (Checker, error) {
	component, ok := r.Component(name)
	if !ok {
		return nil, ErrComponentNotFound
	}

	checker, ok := component.(Checker)
	if !ok {
		return nil, ErrCheckerUnsupported
	}

	return checker, nil
}

// CheckDescriber returns a registered component as a CheckDescriber.
func (r *Registry) CheckDescriber(name string) (CheckDescriber, error) {
	component, ok := r.Component(name)
	if !ok {
		return nil, ErrComponentNotFound
	}

	describer, ok := component.(CheckDescriber)
	if !ok {
		return nil, ErrCheckDescriberUnsupported
	}

	return describer, nil
}

// Checks returns passive check metadata for one registered component.
func (r *Registry) Checks(ctx context.Context, name string) ([]CheckDescriptor, error) {
	ctx = normalizeContext(ctx)

	component, ok := r.Component(name)
	if !ok {
		return nil, ErrComponentNotFound
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	describer, ok := component.(CheckDescriber)
	if !ok {
		return nil, ErrCheckDescriberUnsupported
	}

	checks, err, _ := safeComponentChecks(describer, ctx)
	if err != nil {
		return nil, err
	}

	return cloneCheckDescriptors(checks), nil
}

// CheckGroup returns a registered component as a CheckGroup.
func (r *Registry) CheckGroup(name string) (CheckGroup, error) {
	component, ok := r.Component(name)
	if !ok {
		return nil, ErrComponentNotFound
	}

	group, ok := component.(CheckGroup)
	if !ok {
		return nil, ErrCheckGroupUnsupported
	}

	return group, nil
}

// CommandHandler returns a registered component as a CommandHandler.
func (r *Registry) CommandHandler(name string) (CommandHandler, error) {
	component, ok := r.Component(name)
	if !ok {
		return nil, ErrComponentNotFound
	}

	handler, ok := component.(CommandHandler)
	if !ok {
		return nil, ErrCommandHandlerUnsupported
	}

	return handler, nil
}

// CommandDescriber returns a registered component as a CommandDescriber.
func (r *Registry) CommandDescriber(name string) (CommandDescriber, error) {
	component, ok := r.Component(name)
	if !ok {
		return nil, ErrComponentNotFound
	}

	describer, ok := component.(CommandDescriber)
	if !ok {
		return nil, ErrCommandDescriberUnsupported
	}

	return describer, nil
}

// Commands returns passive command metadata for one registered component.
func (r *Registry) Commands(ctx context.Context, name string) ([]CommandDescriptor, error) {
	ctx = normalizeContext(ctx)

	component, ok := r.Component(name)
	if !ok {
		return nil, ErrComponentNotFound
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	describer, ok := component.(CommandDescriber)
	if !ok {
		return nil, ErrCommandDescriberUnsupported
	}

	commands, err, _ := safeComponentCommands(describer, ctx)
	if err != nil {
		return nil, err
	}

	return cloneCommandDescriptors(commands), nil
}

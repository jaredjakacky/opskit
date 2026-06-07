package opskit

import "context"

// Readiness describes whether a component or aggregate system can receive work.
//
// Ready is authoritative. When Components are present, the contributor is still
// responsible for setting Ready to the aggregate readiness decision.
type Readiness struct {
	Ready      bool            `json:"ready"`
	Reason     string          `json:"reason,omitempty"`
	Components []ReadinessItem `json:"components,omitempty"`
}

// ReadinessItem describes one component's contribution to readiness.
type ReadinessItem struct {
	Name    string          `json:"name"`
	Kind    string          `json:"kind,omitempty"`
	Policy  ReadinessPolicy `json:"policy,omitempty"`
	Ready   bool            `json:"ready"`
	State   State           `json:"state"`
	Reason  string          `json:"reason,omitempty"`
	Message string          `json:"message,omitempty"`
}

// ReadinessContributor reports readiness separately from general status.
//
// Status answers: "what state is this component in?"
// Readiness answers: "should this component allow the service to receive work?"
type ReadinessContributor interface {
	Readiness(context.Context) Readiness
}

// ReadyReadiness returns a ready readiness result.
func ReadyReadiness(reason string, components ...ReadinessItem) Readiness {
	return Readiness{
		Ready:      true,
		Reason:     reason,
		Components: cloneReadinessItems(components),
	}
}

// NotReadyReadiness returns a not-ready readiness result.
func NotReadyReadiness(reason string, components ...ReadinessItem) Readiness {
	return Readiness{
		Ready:      false,
		Reason:     reason,
		Components: cloneReadinessItems(components),
	}
}

// ReadinessFromStatus builds a readiness result from component status.
func ReadinessFromStatus(info ComponentInfo, status Status) Readiness {
	reason := "component ready"
	if !status.Ready {
		reason = "component not ready"
	}

	return Readiness{
		Ready:  status.Ready,
		Reason: reason,
		Components: []ReadinessItem{
			ReadinessItemFromStatus(info, status),
		},
	}
}

// ReadinessItemFromStatus builds a readiness item from component status.
func ReadinessItemFromStatus(info ComponentInfo, status Status) ReadinessItem {
	return ReadinessItem{
		Name:    info.Name,
		Kind:    info.Kind,
		Ready:   status.Ready,
		State:   normalizeReadinessItemState(status.Ready, status.State),
		Message: status.Message,
	}
}

func cloneReadinessItems(items []ReadinessItem) []ReadinessItem {
	if len(items) == 0 {
		return nil
	}

	cloned := make([]ReadinessItem, len(items))
	copy(cloned, items)
	return cloned
}

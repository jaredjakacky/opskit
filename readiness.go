package opskit

import "context"

// SystemReadiness describes whether the registered system can receive work.
//
// Components preserve the identity, registration policy, and component-owned
// readiness result for each registered readiness participant.
type SystemReadiness struct {
	Ready      bool                 `json:"ready"`
	Reason     string               `json:"reason,omitempty"`
	Components []ComponentReadiness `json:"components,omitempty"`
}

// ComponentReadiness is one registered component's readiness contribution.
type ComponentReadiness struct {
	Component    ComponentInfo         `json:"component"`
	Registration ComponentRegistration `json:"registration"`
	Readiness    Readiness             `json:"readiness"`
}

// Readiness describes whether one component can allow the service to receive
// work.
//
// Ready is authoritative. When Items are present, the contributor is still
// responsible for setting Ready to its aggregate readiness decision.
type Readiness struct {
	Ready  bool            `json:"ready"`
	Reason string          `json:"reason,omitempty"`
	Items  []ReadinessItem `json:"items,omitempty"`
}

// ReadinessImpact describes whether a child item can block its contributor's
// readiness decision. It is deliberately separate from ReadinessPolicy, which
// applies only to components registered with Registry.
type ReadinessImpact string

const (
	// ReadinessImpactBlocking means an unsatisfied item can block its
	// contributor's readiness decision.
	ReadinessImpactBlocking ReadinessImpact = "blocking"

	// ReadinessImpactNonBlocking means the item is readiness detail but does not
	// block its contributor's readiness decision.
	ReadinessImpactNonBlocking ReadinessImpact = "non_blocking"
)

// ReadinessItem describes one child or domain item within its parent
// component's readiness result. Name is scoped to the parent component and does
// not need to be globally unique.
type ReadinessItem struct {
	Name    string          `json:"name"`
	Kind    string          `json:"kind,omitempty"`
	Impact  ReadinessImpact `json:"impact,omitempty"`
	Ready   bool            `json:"ready"`
	State   State           `json:"state"`
	Reason  string          `json:"reason,omitempty"`
	Message string          `json:"message,omitempty"`
}

// ReadinessContributor reports readiness separately from general status.
//
// Status answers: "what state is this component in?"
// Readiness answers: "should this component allow the service to receive work?"
//
// Readiness is a descriptive hook used on probe and request paths. It should
// return quickly from cached or local state and must not perform dependency
// checks, call external services, schedule work, or mutate lifecycle state.
type ReadinessContributor interface {
	Readiness(context.Context) Readiness
}

// ReadyReadiness returns a ready readiness result.
func ReadyReadiness(reason string, items ...ReadinessItem) Readiness {
	return Readiness{
		Ready:  true,
		Reason: reason,
		Items:  normalizeReadinessItems(items),
	}
}

// NotReadyReadiness returns a not-ready readiness result.
func NotReadyReadiness(reason string, items ...ReadinessItem) Readiness {
	return Readiness{
		Ready:  false,
		Reason: reason,
		Items:  normalizeReadinessItems(items),
	}
}

// ReadinessFromItems builds a readiness result whose aggregate readiness is
// derived from blocking readiness items. Missing and unknown impacts fail
// closed as blocking. When no blocking items are supplied, the result is not
// ready; callers with a different domain rule can construct Readiness directly.
func ReadinessFromItems(reason string, items ...ReadinessItem) Readiness {
	normalized := normalizeReadinessItems(items)

	if len(normalized) == 0 {
		if reason == "" {
			reason = "no readiness items"
		}
		return Readiness{
			Ready:  false,
			Reason: reason,
		}
	}

	blocking := 0
	ready := true
	for _, item := range normalized {
		if !blocksReadinessImpact(item.Impact) {
			continue
		}

		blocking++
		if !item.Ready {
			ready = false
		}
	}

	if blocking == 0 {
		if reason == "" {
			reason = "no blocking readiness items"
		}
		return Readiness{
			Ready:  false,
			Reason: reason,
			Items:  normalized,
		}
	}

	if reason == "" {
		if ready {
			reason = "all blocking readiness items ready"
		} else {
			reason = "one or more blocking readiness items are not ready"
		}
	}

	return Readiness{
		Ready:  ready,
		Reason: reason,
		Items:  normalized,
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
		Items: []ReadinessItem{
			ReadinessItemFromStatus(info, status),
		},
	}
}

// ReadinessItemFromStatus builds a readiness item from component status.
func ReadinessItemFromStatus(info ComponentInfo, status Status) ReadinessItem {
	return ReadinessItem{
		Name:    info.Name,
		Kind:    info.Kind,
		Impact:  ReadinessImpactBlocking,
		Ready:   status.Ready,
		State:   normalizeReadinessItemState(status.Ready, status.State),
		Message: status.Message,
	}
}

func normalizeReadinessItems(items []ReadinessItem) []ReadinessItem {
	if len(items) == 0 {
		return nil
	}

	cloned := make([]ReadinessItem, len(items))
	for i, item := range items {
		item.Impact = normalizeReadinessImpact(item.Impact)
		item.State = normalizeReadinessItemState(item.Ready, item.State)
		cloned[i] = item
	}
	return cloned
}

func normalizeReadinessImpact(impact ReadinessImpact) ReadinessImpact {
	switch impact {
	case ReadinessImpactBlocking, ReadinessImpactNonBlocking:
		return impact
	default:
		return ReadinessImpactBlocking
	}
}

func blocksReadinessImpact(impact ReadinessImpact) bool {
	return normalizeReadinessImpact(impact) == ReadinessImpactBlocking
}

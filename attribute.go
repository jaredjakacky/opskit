package opskit

// Attribute is a safe operational key/value pair.
//
// Attribute values may be exposed through logs, admin endpoints, readiness
// summaries, telemetry, and diagnostics. Callers must not include secrets,
// credentials, tokens, raw connection strings, or unredacted user data.
//
// Attribute keys are not validated by Opskit. Prefer stable, low-cardinality
// safe tokens using ASCII letters, ASCII digits, dots, underscores, or hyphens.
type Attribute struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// Attr returns a safe operational attribute. Attr does not validate or redact
// its inputs; callers own key stability and value safety.
func Attr(key, value string) Attribute {
	return Attribute{
		Key:   key,
		Value: value,
	}
}

func cloneAttributes(attrs []Attribute) []Attribute {
	if len(attrs) == 0 {
		return nil
	}

	cloned := make([]Attribute, len(attrs))
	copy(cloned, attrs)
	return cloned
}

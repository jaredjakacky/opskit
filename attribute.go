package opskit

// Attribute is a safe operational key/value pair.
//
// Attribute values may be exposed through logs, admin endpoints, readiness
// summaries, telemetry, and diagnostics. Callers must not include secrets,
// credentials, tokens, raw connection strings, or unredacted user data.
type Attribute struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// Attr returns a safe operational attribute.
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

package opskit

import "testing"

func TestAttr(t *testing.T) {
	attr := Attr("component", "cache")

	if attr.Key != "component" {
		t.Fatalf("Key = %q, want component", attr.Key)
	}
	if attr.Value != "cache" {
		t.Fatalf("Value = %q, want cache", attr.Value)
	}
}

func TestAttributeJSON(t *testing.T) {
	requireJSON(t, Attr("component", "cache"), `{"key":"component","value":"cache"}`)
}

func TestCloneAttributes(t *testing.T) {
	attrs := []Attribute{
		Attr("component", "cache"),
		Attr("shard", "primary"),
	}

	cloned := cloneAttributes(attrs)
	attrs[0] = Attr("component", "mutated")

	if len(cloned) != 2 {
		t.Fatalf("cloned length = %d, want 2", len(cloned))
	}
	if cloned[0] != Attr("component", "cache") {
		t.Fatalf("cloned[0] = %+v, want component cache", cloned[0])
	}
	if cloned[1] != Attr("shard", "primary") {
		t.Fatalf("cloned[1] = %+v, want shard primary", cloned[1])
	}
}

func TestCloneAttributesEmpty(t *testing.T) {
	if got := cloneAttributes(nil); got != nil {
		t.Fatalf("cloneAttributes(nil) = %+v, want nil", got)
	}
	if got := cloneAttributes([]Attribute{}); got != nil {
		t.Fatalf("cloneAttributes(empty) = %+v, want nil", got)
	}
}

package art

import "testing"

func TestBuild(t *testing.T) {
	ps := Build([]ProviderConfig{
		{Name: "coverartarchive"},
		{Name: "template", Settings: map[string]string{"pattern": "https://x/{rgid}.jpg"}},
		{Name: "nope"},     // unknown, skipped
		{Name: "template"}, // missing pattern, factory rejects it, skipped
	})
	if len(ps) != 2 {
		t.Fatalf("want 2 built providers, got %d", len(ps))
	}
	if _, ok := ps[0].(CAA); !ok {
		t.Fatalf("first provider should be CAA, got %T", ps[0])
	}
	if _, ok := ps[1].(Template); !ok {
		t.Fatalf("second provider should be Template, got %T", ps[1])
	}
}

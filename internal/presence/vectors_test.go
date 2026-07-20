package presence

import (
	"encoding/json"
	"os"
	"testing"
)

// the vectors live outside this package because the connect page reimplements render in ts
// against the same file, so an edge or key-set change moves the fixture first and both sides
// follow. shared with nobody at the minute, the page got deleted out from under it
func TestRender_SharedVectors(t *testing.T) {
	b, err := os.ReadFile("../../testdata/template-vectors.json")
	if err != nil {
		t.Fatalf("shared vectors unreadable: %v", err)
	}
	var vs []struct {
		Template string            `json:"template"`
		Fixture  map[string]string `json:"fixture"`
		Expected string            `json:"expected"`
	}
	if err := json.Unmarshal(b, &vs); err != nil {
		t.Fatalf("shared vectors are not valid json: %v", err)
	}
	if len(vs) == 0 {
		t.Fatal("shared vectors are empty")
	}
	for _, v := range vs {
		tk := Track{
			Artist: v.Fixture["artist"], AlbumArtist: v.Fixture["albumartist"],
			Album: v.Fixture["album"], Title: v.Fixture["track"],
			AlbumID: v.Fixture["albumid"], RGID: v.Fixture["rgid"],
		}
		if got := render(v.Template, tk); got != v.Expected {
			t.Errorf("vector %q: got %q want %q", v.Template, got, v.Expected)
		}
	}
}

// pins the go defaults to the same fixture the page previews from. without it the card you
// design and the card you actually get drift apart, which is a miserable bug to chase
func TestDefaultCard_SharedFixture(t *testing.T) {
	b, err := os.ReadFile("../../testdata/default-card.json")
	if err != nil {
		t.Fatalf("shared default card unreadable: %v", err)
	}
	var d struct {
		Type              string `json:"type"`
		Header            string `json:"header"`
		Details           string `json:"details"`
		State             string `json:"state"`
		StatusDisplayType string `json:"status_display_type"`
	}
	if err := json.Unmarshal(b, &d); err != nil {
		t.Fatalf("shared default card is not valid json: %v", err)
	}
	p := Look{}.Prefs()
	if p.Type != d.Type || p.Header != d.Header || p.Details != d.Details || p.State != d.State || p.StatusDisplayType != d.StatusDisplayType {
		t.Fatalf("the default card drifted from the shared fixture: got %+v want %+v", p, d)
	}
}

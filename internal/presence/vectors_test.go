package presence

import (
	"encoding/json"
	"os"
	"testing"
)

// the connect page reimplements render in ts, both sides load these vectors so a
// key-set or edge change moves the file first and drags the other side with it
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

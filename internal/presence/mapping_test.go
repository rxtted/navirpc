package presence

import "testing"

func TestMap_ArtistHeaderAndTimestamps(t *testing.T) {
	tr := Track{Title: "Happier Than Ever", Artist: "Billie Eilish", Album: "Happier Than Ever", DurationMs: 180000}
	a := Map(tr, Prefs{Header: "artist"}, 40000, 1000000)
	if a.Type != 2 || a.Platform != "desktop" {
		t.Fatalf("bad base: %+v", a)
	}
	if a.Name != "Billie Eilish" || a.Details != "Happier Than Ever" || a.State != "Happier Than Ever" {
		t.Fatalf("bad text: %+v", a)
	}
	if a.Start != 960000 || a.End != 1140000 {
		t.Fatalf("bad ts: start=%d end=%d", a.Start, a.End)
	}
}

func TestMap_HeaderModes(t *testing.T) {
	tr := Track{Title: "t", Artist: "ar", Album: "al", DurationMs: 1000}
	cases := map[string]string{"artist": "ar", "album": "al", "track": "t", "Navidrome": "Navidrome"}
	for header, want := range cases {
		if got := Map(tr, Prefs{Header: header}, 0, 0).Name; got != want {
			t.Errorf("header %q: name = %q, want %q", header, got, want)
		}
	}
}

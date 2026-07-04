package presence

import "testing"

func TestMap_Templates(t *testing.T) {
	tk := Track{Title: "Hang 'Em High", Artist: "My Chemical Romance", AlbumArtist: "MCR", Album: "Three Cheers", DurationMs: 200000}

	a := Map(tk, Prefs{Header: "{artist}", Details: "{track}", State: "{album}"}, 30000, 100000)
	if a.Name != "My Chemical Romance" || a.Details != "Hang 'Em High" || a.State != "Three Cheers" {
		t.Fatalf("default templates: %+v", a)
	}
	if a.Type != 2 || a.Platform != "desktop" {
		t.Fatalf("activity shape: %+v", a)
	}
	if a.Start != 100000-30000 || a.End != a.Start+200000 {
		t.Fatalf("timestamps anchor to position: start=%d end=%d", a.Start, a.End)
	}

	a = Map(tk, Prefs{Header: "{artist} - {album}", Details: "{track}", State: "{albumartist}"}, 0, 0)
	if a.Name != "My Chemical Romance - Three Cheers" || a.State != "MCR" {
		t.Fatalf("custom template interpolation: %+v", a)
	}
}

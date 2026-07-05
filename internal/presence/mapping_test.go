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

func TestMap_TypeAndStatusDisplay(t *testing.T) {
	types := map[string]int{"": 2, "listening": 2, "playing": 0, "streaming": 1, "watching": 3, "competing": 5, "bogus": 2}
	for kind, want := range types {
		if got := Map(Track{}, Prefs{Type: kind}, 0, 0).Type; got != want {
			t.Errorf("type %q: got %d want %d", kind, got, want)
		}
	}
	lines := map[string]int{"": 0, "name": 0, "state": 1, "details": 2, "bogus": 0}
	for line, want := range lines {
		if got := Map(Track{}, Prefs{StatusDisplayType: line}, 0, 0).StatusDisplayType; got != want {
			t.Errorf("status_display_type %q: got %d want %d", line, got, want)
		}
	}
}

func TestMap_URLsAndButtonsTemplate(t *testing.T) {
	tk := Track{Album: "Revenge", AlbumID: "al1", RGID: "rg1", Artist: "MCR"}
	a := Map(tk, Prefs{
		DetailsURL: "https://nav/album/{albumid}",
		StateURL:   "https://mb/{rgid}",
		LargeText:  "{album}",
		SmallImage: "https://icon/{rgid}.png",
		SmallText:  "{artist}",
		Buttons: []Button{
			{Label: "Open {album}", URL: "https://nav/album/{albumid}"},
			{Label: "Static", URL: "https://example.com"},
		},
	}, 0, 0)
	if a.DetailsURL != "https://nav/album/al1" || a.StateURL != "https://mb/rg1" {
		t.Fatalf("url templating: details=%q state=%q", a.DetailsURL, a.StateURL)
	}
	if a.LargeText != "Revenge" || a.SmallImage != "https://icon/rg1.png" || a.SmallText != "MCR" {
		t.Fatalf("asset templates: %+v", a)
	}
	if len(a.Buttons) != 2 || a.Buttons[0].Label != "Open Revenge" || a.Buttons[0].URL != "https://nav/album/al1" {
		t.Fatalf("button templating: %+v", a.Buttons)
	}
	if a.Buttons[1].URL != "https://example.com" {
		t.Fatalf("static button passthrough: %+v", a.Buttons[1])
	}
}

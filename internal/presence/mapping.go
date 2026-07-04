package presence

import "strings"

type Track struct {
	Title       string
	Artist      string
	AlbumArtist string
	Album       string
	RGID        string // musicbrainz release-group id
	AlbumID     string // musicbrainz release id
	DurationMs  int64
}

type Activity struct {
	Type       int
	Name       string
	Platform   string
	Details    string
	State      string
	Start      int64
	End        int64
	LargeImage string
}

// each field is a template with {artist} {albumartist} {album} {track} placeholders.
type Prefs struct {
	Header  string
	Details string
	State   string
}

// position and now are unix-ms.
func Map(t Track, prefs Prefs, positionMs, nowMs int64) Activity {
	start := nowMs - positionMs
	return Activity{
		Type:     2,
		Name:     render(prefs.Header, t),
		Platform: "desktop",
		Details:  render(prefs.Details, t),
		State:    render(prefs.State, t),
		Start:    start,
		End:      start + t.DurationMs,
	}
}

func render(tmpl string, t Track) string {
	return strings.NewReplacer(
		"{artist}", t.Artist,
		"{albumartist}", t.AlbumArtist,
		"{album}", t.Album,
		"{track}", t.Title,
	).Replace(tmpl)
}

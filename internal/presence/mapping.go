package presence

type Track struct {
	Title      string
	Artist     string
	Album      string
	RGID       string // musicbrainz release-group id
	AlbumID    string // musicbrainz release id
	DurationMs int64
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

type Prefs struct {
	Header string // "artist" | "album" | "track" | any fixed string
}

// position and now are unix-ms so start anchors the progress bar to the real position.
func Map(t Track, prefs Prefs, positionMs, nowMs int64) Activity {
	start := nowMs - positionMs
	return Activity{
		Type:     2,
		Name:     header(t, prefs.Header),
		Platform: "desktop",
		Details:  t.Title,
		State:    t.Album,
		Start:    start,
		End:      start + t.DurationMs,
	}
}

func header(t Track, mode string) string {
	switch mode {
	case "", "artist":
		return t.Artist
	case "album":
		return t.Album
	case "track":
		return t.Title
	default:
		return mode
	}
}

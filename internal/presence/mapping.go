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

type Button struct {
	Label string `json:"label"`
	URL   string `json:"url"`
}

type Activity struct {
	Type              int
	Name              string
	Platform          string
	Details           string
	DetailsURL        string
	State             string
	StateURL          string
	StatusDisplayType int
	Start             int64
	End               int64
	LargeImage        string
	LargeText         string
	SmallImage        string
	SmallText         string
	Buttons           []Button
}

// every text field is a template with {artist} {albumartist} {album} {track} placeholders,
// url fields also take {albumid} {rgid}. Type and StatusDisplayType map a keyword to an int.
type Prefs struct {
	Type              string
	Header            string
	Details           string
	State             string
	DetailsURL        string
	StateURL          string
	StatusDisplayType string
	LargeText         string
	SmallImage        string
	SmallText         string
	Buttons           []Button
}

// position and now are unix-ms.
func Map(t Track, prefs Prefs, positionMs, nowMs int64) Activity {
	start := nowMs - positionMs
	a := Activity{
		Type:              activityType(prefs.Type),
		Name:              render(prefs.Header, t),
		Platform:          "desktop",
		Details:           render(prefs.Details, t),
		DetailsURL:        render(prefs.DetailsURL, t),
		State:             render(prefs.State, t),
		StateURL:          render(prefs.StateURL, t),
		StatusDisplayType: statusDisplayType(prefs.StatusDisplayType),
		Start:             start,
		End:               start + t.DurationMs,
		LargeText:         render(prefs.LargeText, t),
		SmallImage:        render(prefs.SmallImage, t),
		SmallText:         render(prefs.SmallText, t),
	}
	for _, b := range prefs.Buttons {
		a.Buttons = append(a.Buttons, Button{Label: render(b.Label, t), URL: render(b.URL, t)})
	}
	return a
}

func render(tmpl string, t Track) string {
	return strings.NewReplacer(
		"{artist}", t.Artist,
		"{albumartist}", t.AlbumArtist,
		"{album}", t.Album,
		"{track}", t.Title,
		"{albumid}", t.AlbumID,
		"{rgid}", t.RGID,
	).Replace(tmpl)
}

func activityType(kind string) int {
	switch kind {
	case "playing":
		return 0
	case "streaming":
		return 1
	case "watching":
		return 3
	case "competing":
		return 5
	default:
		return 2 // listening
	}
}

func statusDisplayType(line string) int {
	switch line {
	case "state":
		return 1
	case "details":
		return 2
	default:
		return 0 // name
	}
}

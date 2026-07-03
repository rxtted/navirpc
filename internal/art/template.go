package art

import (
	"errors"
	"strings"
)

func init() {
	register("template", func(s map[string]string) (Provider, error) {
		if s["pattern"] == "" {
			return nil, errors.New("template provider needs a pattern")
		}
		return Template{Pattern: s["pattern"]}, nil
	})
}

// a user-supplied url pattern with {rgid} {albumid} {artist} {album} placeholders;
// resolves only when every placeholder it uses has a value.
type Template struct{ Pattern string }

func (t Template) Resolve(m Meta) (string, bool) {
	if t.Pattern == "" {
		return "", false
	}
	fields := map[string]string{"{rgid}": m.RGID, "{albumid}": m.AlbumID, "{artist}": m.Artist, "{album}": m.Album}
	out := t.Pattern
	for ph, val := range fields {
		if strings.Contains(out, ph) {
			if val == "" {
				return "", false
			}
			out = strings.ReplaceAll(out, ph, val)
		}
	}
	return out, true
}

package art

import (
	"encoding/json"
	"net/url"
	"strings"
)

func init() {
	register("itunes", func(map[string]string) (Provider, error) { return iTunes{}, nil })
}

// looks album art up by artist and album name via the iTunes Search API, no api key.
// the catch-all fallback for a track CAA can't resolve by mbid.
type iTunes struct{}

func (iTunes) Resolve(m Meta, get Getter) (string, bool) {
	if get == nil || m.Artist == "" || m.Album == "" {
		return "", false
	}
	q := url.Values{
		"term":   {m.Artist + " " + m.Album},
		"media":  {"music"},
		"entity": {"album"},
		"limit":  {"5"},
	}
	body, status, err := get.Get("https://itunes.apple.com/search?" + q.Encode())
	if err != nil || status < 200 || status >= 300 {
		return "", false
	}
	var out struct {
		Results []struct {
			ArtworkURL100  string `json:"artworkUrl100"`
			ArtistName     string `json:"artistName"`
			CollectionName string `json:"collectionName"`
		} `json:"results"`
	}
	if json.Unmarshal(body, &out) != nil {
		return "", false
	}
	// the search is fuzzy, so take the first result whose artist and album both match the
	// query rather than iTunes' top relevance hit
	for _, res := range out.Results {
		if res.ArtworkURL100 != "" && nameOverlap(res.ArtistName, m.Artist) && nameOverlap(res.CollectionName, m.Album) {
			return strings.Replace(res.ArtworkURL100, "100x100bb", "600x600bb", 1), true
		}
	}
	return "", false
}

func nameOverlap(a, b string) bool {
	na, nb := strings.ToLower(strings.TrimSpace(a)), strings.ToLower(strings.TrimSpace(b))
	return na != "" && nb != "" && (strings.Contains(na, nb) || strings.Contains(nb, na))
}

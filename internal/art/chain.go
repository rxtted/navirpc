package art

// resolves the first provider that hits.
func Chain(ps []Provider, m Meta, get Getter) (string, bool) {
	for _, p := range ps {
		if url, ok := p.Resolve(m, get); ok {
			return url, true
		}
	}
	return "", false
}

// the identity a resolved url is remembered under, strongest id first. empty means
// the track carries nothing usable, resolve every time.
func Key(m Meta) string {
	if m.AlbumID != "" {
		return m.AlbumID
	}
	if m.RGID != "" {
		return m.RGID
	}
	if m.Artist == "" && m.Album == "" {
		return ""
	}
	return m.Artist + " - " + m.Album
}

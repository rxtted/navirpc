package art

// resolves the first provider that hits, caching the result per album including
// misses, so a coverless album isn't re-resolved on every track.
func Chain(ps []Provider, cache Cache, m Meta, get Getter) (string, bool) {
	key := m.AlbumID
	if key == "" {
		key = m.RGID
	}
	if key == "" {
		key = m.Artist + " - " + m.Album
	}
	if key != "" && cache != nil {
		if v, ok := cache.Get(key); ok {
			return v, v != ""
		}
	}
	for _, p := range ps {
		if url, ok := p.Resolve(m, get); ok {
			cacheSet(cache, key, url)
			return url, true
		}
	}
	cacheSet(cache, key, "")
	return "", false
}

func cacheSet(cache Cache, key, val string) {
	if key != "" && cache != nil {
		cache.Set(key, val)
	}
}

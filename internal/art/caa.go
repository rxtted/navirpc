package art

func init() {
	register("coverartarchive", func(map[string]string) (Provider, error) { return CAA{}, nil })
}

// the default provider when art is on, Cover Art Archive by release-group mbid.
type CAA struct{}

func (CAA) Resolve(m Meta, _ Getter) (string, bool) {
	if m.RGID == "" {
		return "", false
	}
	return "https://coverartarchive.org/release-group/" + m.RGID + "/front", true
}

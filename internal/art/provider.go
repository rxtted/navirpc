package art

type Meta struct {
	RGID    string // musicbrainz release-group id
	AlbumID string // musicbrainz release id
	Artist  string
	Album   string
}

// the outbound http a lookup provider uses to resolve art, template providers ignore it.
// any host a lookup provider fetches must be in the manifest allowlist.
type Getter interface {
	Get(url string) (body []byte, status int, err error)
}

type Provider interface {
	Resolve(Meta, Getter) (url string, ok bool)
}

type Cache interface {
	Get(key string) (string, bool)
	Set(key, val string)
}

// one enabled provider from user config, its registered name plus its own settings, an
// api key or a url pattern, nil for providers that need none.
type ProviderConfig struct {
	Name     string            `json:"name"`
	Settings map[string]string `json:"settings,omitempty"`
}

type factory func(settings map[string]string) (Provider, error)

var registry = map[string]factory{}

// each provider file registers itself here from its init function.
func register(name string, f factory) { registry[name] = f }

// turns the user's ordered provider configs into a chain, skipping any unknown name or
// one whose factory rejects its settings.
func Build(configs []ProviderConfig) []Provider {
	var ps []Provider
	for _, c := range configs {
		f, ok := registry[c.Name]
		if !ok {
			continue
		}
		if p, err := f(c.Settings); err == nil {
			ps = append(ps, p)
		}
	}
	return ps
}

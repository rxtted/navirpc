package art

type Meta struct {
	RGID    string // musicbrainz release-group id
	AlbumID string // musicbrainz release id
	Artist  string
	Album   string
}

// a Provider turns track metadata into a public image url. template providers build
// the url from metadata (discord fetches it, so any host works); lookup providers make
// their own outbound call and so must be manifest-allowlisted.
type Provider interface {
	Resolve(Meta) (url string, ok bool)
}

type Cache interface {
	Get(key string) (string, bool)
	Set(key, val string)
}

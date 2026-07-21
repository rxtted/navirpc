// the discord wire protocol, request construction, status classification and
// session-token semantics.
package discord

const apiBase = "https://discord.com/api/v10"

// discord is behind cloudflare, which 1010-blocks a default http client bc of course it does.
// exported because the art getter needs it too
const UserAgent = "Mozilla/5.0 navirpc/0.1"

type Request struct {
	Method    string
	URL       string
	Headers   map[string]string
	Body      []byte
	TimeoutMs int32
}

type Response struct {
	StatusCode int
	Headers    map[string]string
	Body       []byte
}

type Doer interface {
	Do(Request) (Response, error)
}

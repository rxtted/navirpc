//go:build wasip1

package main

import (
	"fmt"

	"atrophy/navirpc/internal/discord"
	"github.com/navidrome/navidrome/plugins/pdk/go/host"
)

// the one genuinely untestable line in the plugin, everything that decides lives
// in internal/discord and gets tested there
type hostDoer struct{}

func (hostDoer) Do(r discord.Request) (discord.Response, error) {
	resp, err := host.HTTPSend(host.HTTPRequest{
		Method: r.Method, URL: r.URL, Headers: r.Headers, Body: r.Body, TimeoutMs: r.TimeoutMs,
	})
	if err != nil {
		return discord.Response{}, fmt.Errorf("host http: %w", err)
	}
	return discord.Response{StatusCode: int(resp.StatusCode), Headers: resp.Headers, Body: resp.Body}, nil
}

type httpGetter struct{}

func (httpGetter) Get(url string) ([]byte, int, error) {
	resp, err := host.HTTPSend(host.HTTPRequest{
		Method:    "GET",
		URL:       url,
		Headers:   map[string]string{"User-Agent": discord.UserAgent},
		TimeoutMs: 10000,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("art fetch: %w", err)
	}
	return resp.Body, int(resp.StatusCode), nil
}

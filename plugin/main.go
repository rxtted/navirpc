//go:build wasip1

package main

import (
	"github.com/navidrome/navidrome/plugins/pdk/go/lifecycle"
	"github.com/navidrome/navidrome/plugins/pdk/go/scrobbler"
)

type plugin struct{}

func init() {
	lifecycle.Register(&plugin{})
	scrobbler.Register(&plugin{})
}

var _ lifecycle.InitProvider = (*plugin)(nil)

func (plugin) OnInit() error { return nil }

func (plugin) IsAuthorized(scrobbler.IsAuthorizedRequest) (bool, error) { return true, nil }
func (plugin) NowPlaying(scrobbler.NowPlayingRequest) error             { return nil }
func (plugin) Scrobble(scrobbler.ScrobbleRequest) error                 { return nil }
func (plugin) PlaybackReport(scrobbler.PlaybackReportRequest) error     { return nil }

func main() {}

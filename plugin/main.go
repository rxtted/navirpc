//go:build wasip1

// probe scaffold: lifecycle + scrobbler + config, logging only. loads into a real
// navidrome to observe that the build runs, the PlaybackReport cadence and fields, and
// whether the per-user config array is readable. no outbound calls.
package main

import (
	"encoding/json"
	"strconv"

	"github.com/navidrome/navidrome/plugins/pdk/go/lifecycle"
	"github.com/navidrome/navidrome/plugins/pdk/go/pdk"
	"github.com/navidrome/navidrome/plugins/pdk/go/scrobbler"
)

type probe struct{}

func init() {
	lifecycle.Register(&probe{})
	scrobbler.Register(&probe{})
}

var _ lifecycle.InitProvider = (*probe)(nil)

type userBinding struct {
	Username string `json:"username"`
	Token    string `json:"token"`
}

func (probe) OnInit() error {
	pdk.Log(pdk.LogInfo, "navirpc: init ok")
	raw, ok := pdk.GetConfig("users")
	if !ok {
		pdk.Log(pdk.LogInfo, "navirpc: no users config set")
		return nil
	}
	var users []userBinding
	if err := json.Unmarshal([]byte(raw), &users); err != nil {
		pdk.Log(pdk.LogWarn, "navirpc: users config is not a json array: "+raw)
		return nil
	}
	pdk.Log(pdk.LogInfo, "navirpc: config users count="+strconv.Itoa(len(users)))
	for _, u := range users {
		pdk.Log(pdk.LogInfo, "navirpc: bound user="+u.Username+" tokenLen="+strconv.Itoa(len(u.Token)))
	}
	return nil
}

// scrobbler.Register needs all four methods; only PlaybackReport does anything here.
func (probe) IsAuthorized(scrobbler.IsAuthorizedRequest) (bool, error) { return true, nil }
func (probe) NowPlaying(scrobbler.NowPlayingRequest) error             { return nil }
func (probe) Scrobble(scrobbler.ScrobbleRequest) error                 { return nil }

func (probe) PlaybackReport(r scrobbler.PlaybackReportRequest) error {
	pdk.Log(pdk.LogInfo, "navirpc: PlaybackReport"+
		" user="+r.Username+
		" state="+r.State+
		" posMs="+strconv.FormatInt(r.PositionMs, 10)+
		" rate="+strconv.FormatFloat(r.PlaybackRate, 'f', 2, 64)+
		" title="+r.Track.Title+
		" artist="+r.Track.Artist+
		" durSec="+strconv.FormatFloat(float64(r.Track.Duration), 'f', 0, 32)+
		" rgid="+r.Track.MBZReleaseGroupID+
		" albumId="+r.Track.MBZAlbumID)
	return nil
}

func main() {}

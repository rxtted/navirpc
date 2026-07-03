//go:build wasip1

package main

import (
	"encoding/json"
	"time"

	"github.com/navidrome/navidrome/plugins/pdk/go/lifecycle"
	"github.com/navidrome/navidrome/plugins/pdk/go/pdk"
	"github.com/navidrome/navidrome/plugins/pdk/go/scrobbler"
	"github.com/rxtted/navirpc/internal/auth"
	"github.com/rxtted/navirpc/internal/presence"
)

const centralClientID = "1522831068774924308"

type plugin struct{}

func init() {
	lifecycle.Register(&plugin{})
	scrobbler.Register(&plugin{})
}

var _ lifecycle.InitProvider = (*plugin)(nil)

type userConfig struct {
	Username string `json:"username"`
	Token    string `json:"token"`
}

func (plugin) OnInit() error {
	clientID := centralClientID
	if v, ok := pdk.GetConfig("client_id"); ok && v != "" {
		clientID = v
	}
	raw, ok := pdk.GetConfig("users")
	if !ok || raw == "" {
		return nil
	}
	var users []userConfig
	if err := json.Unmarshal([]byte(raw), &users); err != nil {
		pdk.Log(pdk.LogWarn, "navirpc: users config is not a json array")
		return nil
	}
	store := kvStore{}
	for _, u := range users {
		var cur *auth.Stored
		if s, ok := store.Load(u.Username); ok {
			cur = &s
		}
		store.Save(u.Username, *auth.Reconcile(seedFrom(u.Token), clientID, "", cur))
	}
	return nil
}

// the token config field is either a raw refresh token or the connect page's json blob.
func seedFrom(token string) string {
	var blob struct {
		Token string `json:"token"`
	}
	if json.Unmarshal([]byte(token), &blob) == nil && blob.Token != "" {
		return blob.Token
	}
	return token
}

func (plugin) PlaybackReport(r scrobbler.PlaybackReportRequest) error {
	switch r.State {
	case "playing", "starting":
		publishNowPlaying(r)
	case "stopped", "expired":
		if sess := loadSession(r.Username); sess != "" {
			_ = discordPublisher{}.Clear(r.Username, sess)
			clearSession(r.Username)
		}
	}
	return nil
}

func publishNowPlaying(r scrobbler.PlaybackReportRequest) {
	if _, err := auth.EnsureFresh(r.Username, kvStore{}, discordRefresher{}, time.Now().Unix()); err != nil {
		pdk.Log(pdk.LogError, "navirpc: token for "+r.Username+" unusable, reconnect: "+err.Error())
		return
	}
	track := presence.Track{
		Title: r.Track.Title, Artist: r.Track.Artist, Album: r.Track.Album,
		RGID: r.Track.MBZReleaseGroupID, AlbumID: r.Track.MBZAlbumID,
		DurationMs: int64(r.Track.Duration * 1000),
	}
	act := presence.Map(track, presence.Prefs{Header: "artist"}, r.PositionMs, time.Now().UnixMilli())
	sess, err := discordPublisher{}.Publish(r.Username, presence.Desired{Kind: "play", Act: act})
	if err != nil {
		pdk.Log(pdk.LogWarn, "navirpc: publish failed for "+r.Username+": "+err.Error())
		return
	}
	saveSession(r.Username, sess)
}

func (plugin) IsAuthorized(scrobbler.IsAuthorizedRequest) (bool, error) { return true, nil }
func (plugin) NowPlaying(scrobbler.NowPlayingRequest) error             { return nil }
func (plugin) Scrobble(scrobbler.ScrobbleRequest) error                 { return nil }

func main() {}

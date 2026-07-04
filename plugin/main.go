//go:build wasip1

package main

import (
	"encoding/json"
	"time"

	"github.com/navidrome/navidrome/plugins/pdk/go/host"
	"github.com/navidrome/navidrome/plugins/pdk/go/lifecycle"
	"github.com/navidrome/navidrome/plugins/pdk/go/pdk"
	"github.com/navidrome/navidrome/plugins/pdk/go/scheduler"
	"github.com/navidrome/navidrome/plugins/pdk/go/scrobbler"
	"github.com/rxtted/navirpc/internal/art"
	"github.com/rxtted/navirpc/internal/auth"
	"github.com/rxtted/navirpc/internal/presence"
)

const centralClientID = "1522831068774924308"

// clear the card as soon as playback stops, there's no timer to fire a deferred clear.
const clearDebounceMs = 0

// keepalive and flush tick, well under the 20-min session TTL, frequent enough to
// resync a throttled scrub within the rate window.
const tickCron = "@every 15s"

type plugin struct{}

func init() {
	lifecycle.Register(&plugin{})
	scrobbler.Register(&plugin{})
	scheduler.Register(&plugin{})
}

var (
	_ lifecycle.InitProvider     = (*plugin)(nil)
	_ scheduler.CallbackProvider = (*plugin)(nil)
)

type userConfig struct {
	Username string `json:"username"`
	Token    string `json:"token"`
}

func readUsers() []userConfig {
	raw, ok := pdk.GetConfig("users")
	if !ok || raw == "" {
		return nil
	}
	var users []userConfig
	if json.Unmarshal([]byte(raw), &users) != nil {
		pdk.Log(pdk.LogWarn, "navirpc: users config is not a json array")
		return nil
	}
	return users
}

func (plugin) OnInit() error {
	clientID := centralClientID
	if v, ok := pdk.GetConfig("client_id"); ok && v != "" {
		clientID = v
	}
	store := kvStore{}
	for _, u := range readUsers() {
		seed := seedFrom(u.Token)
		var cur *auth.Stored
		if s, ok := store.Load(u.Username); ok {
			cur = &s
		}
		next := auth.Reconcile(seed, clientID, "", cur)
		// write only on a real config change of seed or client_id, so a reload can't clobber
		// a token the report path just rotated on the other goroutine
		if cur == nil || cur.Seed != seed || cur.ClientID != clientID {
			if err := store.Save(u.Username, *next); err != nil {
				pdk.Log(pdk.LogWarn, "navirpc: could not persist token for "+u.Username+": "+err.Error())
			}
		}
		clearState(u.Username) // drop stale playback/presence so the first report starts fresh
	}
	if _, err := host.SchedulerScheduleRecurring(tickCron, "", "keepalive"); err != nil {
		pdk.Log(pdk.LogWarn, "navirpc: scheduler register failed: "+err.Error())
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
	nowMs := time.Now().UnixMilli()
	snap := loadSnapshot(r.Username)
	us := presence.RestoreUserState(clearDebounceMs, snap)

	var desired presence.Desired
	switch r.State {
	case "playing", "starting":
		tk := track(r)
		act := presence.Map(tk, presence.Prefs{Header: "artist"}, r.PositionMs, nowMs)
		// resolve art only on an album change, reuse the current cover otherwise
		if tk.Album != "" && tk.Album == snap.LastAct.State && snap.LastAct.LargeImage != "" {
			act.LargeImage = snap.LastAct.LargeImage
		} else {
			act.LargeImage = resolveArt(tk)
		}
		desired, _ = us.OnReport(r.State, act, nowMs)
	case "paused", "stopped", "expired":
		us.OnReport(r.State, presence.Activity{}, nowMs)
		desired, _ = us.Due(nowMs)
	default:
		return nil
	}
	saveSnapshot(r.Username, us.Snapshot())

	if desired.Seq == 0 {
		return nil
	}
	if _, err := auth.EnsureFresh(r.Username, kvStore{}, discordRefresher{}, time.Now().Unix()); err != nil {
		pdk.Log(pdk.LogError, "navirpc: token for "+r.Username+" unusable, reconnect: "+err.Error())
		return nil
	}
	ps, err := presence.Reconcile(r.Username, desired, loadPresence(r.Username), discordPublisher{}, nowMs)
	if err != nil {
		pdk.Log(pdk.LogWarn, "navirpc: reconcile for "+r.Username+" failed: "+err.Error())
	}
	savePresence(r.Username, ps)
	return nil
}

// the scheduler tick, keepalive and flush per active user, using the stored access
// token. it never refreshes so the token key stays single-owner, and it writes only
// the presence key.
func (plugin) OnCallback(scheduler.SchedulerCallbackRequest) error {
	nowMs := time.Now().UnixMilli()
	nowUnix := time.Now().Unix()
	for _, u := range readUsers() {
		snap := loadSnapshot(u.Username)
		if snap.LastKind == "" {
			continue // nothing active to keep alive
		}
		s, ok := kvStore{}.Load(u.Username)
		if !ok || s.Dead || auth.NeedsRefresh(s, nowUnix) {
			continue // no usable access token, the report path refreshes on the next action
		}
		ps := loadPresence(u.Username)
		if ps.SessionToken == "" {
			continue // no established session to keep alive, only the report path creates one
		}
		desired := presence.Desired{Seq: snap.Seq, Kind: snap.LastKind, Act: snap.LastAct}
		ps, err := presence.Reconcile(u.Username, desired, ps, discordPublisher{}, nowMs)
		if err != nil {
			pdk.Log(pdk.LogWarn, "navirpc: tick for "+u.Username+" failed: "+err.Error())
		}
		savePresence(u.Username, ps)
	}
	return nil
}

func track(r scrobbler.PlaybackReportRequest) presence.Track {
	return presence.Track{
		Title: r.Track.Title, Artist: r.Track.Artist, Album: r.Track.Album,
		RGID: r.Track.MBZReleaseGroupID, AlbumID: r.Track.MBZAlbumID,
		DurationMs: int64(r.Track.Duration * 1000),
	}
}

func resolveArt(t presence.Track) string {
	providers := art.Build(configuredArtProviders())
	url, _ := art.Chain(providers, nil, art.Meta{RGID: t.RGID, AlbumID: t.AlbumID, Artist: t.Artist, Album: t.Album}, httpGetter{})
	return url
}

// the ordered art provider chain from config. navidrome doesn't apply schema defaults to
// what a plugin reads, so this default mirrors the manifest schema default and the two
// must stay in sync. a user drops a provider to disable it, reorders it, or empties the
// list for no art.
func configuredArtProviders() []art.ProviderConfig {
	def := []art.ProviderConfig{{Name: "coverartarchive"}, {Name: "itunes"}}
	raw, ok := pdk.GetConfig("art_providers")
	if !ok || raw == "" {
		return def
	}
	var names []string
	if json.Unmarshal([]byte(raw), &names) != nil {
		return def
	}
	cfgs := make([]art.ProviderConfig, 0, len(names))
	for _, n := range names {
		cfgs = append(cfgs, art.ProviderConfig{Name: n})
	}
	return cfgs
}

func (plugin) IsAuthorized(r scrobbler.IsAuthorizedRequest) (bool, error) {
	s, ok := kvStore{}.Load(r.Username)
	return ok && !s.Dead && s.Refresh != "", nil
}

func (plugin) NowPlaying(scrobbler.NowPlayingRequest) error { return nil }
func (plugin) Scrobble(scrobbler.ScrobbleRequest) error     { return nil }

func main() {}

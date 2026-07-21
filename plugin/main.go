//go:build wasip1

package main

import (
	"encoding/json"
	"strings"
	"time"

	"atrophy/navirpc/internal/art"
	"atrophy/navirpc/internal/auth"
	"atrophy/navirpc/internal/discord"
	"atrophy/navirpc/internal/presence"
	"github.com/navidrome/navidrome/plugins/pdk/go/host"
	"github.com/navidrome/navidrome/plugins/pdk/go/lifecycle"
	"github.com/navidrome/navidrome/plugins/pdk/go/pdk"
	"github.com/navidrome/navidrome/plugins/pdk/go/scheduler"
	"github.com/navidrome/navidrome/plugins/pdk/go/scrobbler"
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
	Config   string `json:"config"`
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

func findUser(username string) (userConfig, bool) {
	for _, u := range readUsers() {
		if u.Username == username {
			return u, true
		}
	}
	return userConfig{}, false
}

func (plugin) OnInit() error {
	store := kvStore{}
	nowMs := time.Now().UnixMilli()
	configured := map[string]bool{}
	for _, u := range readUsers() {
		configured[u.Username] = true
		// an own-app token carries its own client_id and wins, else the central app
		seed, clientID := authFrom(u.Token)
		if clientID == "" {
			clientID = centralClientID
		}
		var cur *auth.Stored
		if s, ok := store.Load(u.Username); ok {
			cur = &s
		}
		next := auth.Reconcile(seed, clientID, cur)
		// write only on a real config change of seed or client_id, so a reload can't clobber
		// a token the report path just rotated on the other goroutine
		if cur == nil || cur.Seed != seed || cur.ClientID != clientID {
			if err := store.Save(u.Username, *next); err != nil {
				pdk.Log(pdk.LogWarn, "navirpc: could not persist token for "+u.Username+": "+err.Error())
			}
		}
		// a reboot mid-play is a stop nobody reported. this used to delete the stored
		// state outright, which threw away the one handle that could take the card down,
		// so arm the clear instead and let the machinery that makes stops reliable do
		// reboots too
		st := loadSnapshot(u.Username)
		if st.LastKind != "" {
			us := presence.RestoreUserState(clearDebounceMs, st.Snapshot)
			us.OnReport("stopped", presence.Activity{}, nowMs)
			us.Due(nowMs) // dropping both returns looks wrong, the seq bump is the whole point
			st.Snapshot = us.Snapshot()
			saveSnapshot(u.Username, st)
		}
	}
	retireAbsent(configured)
	if _, err := host.SchedulerScheduleRecurring(tickCron, "", "keepalive"); err != nil {
		pdk.Log(pdk.LogWarn, "navirpc: scheduler register failed: "+err.Error())
	}
	return nil
}

// pulling someone out of the config should stop their presence, and until this existed it
// didnt, their stored token just kept broadcasting. nothing ever ticks an unconfigured user
// so the armed clear cant reach them, removal is the one path that deletes inline
func retireAbsent(configured map[string]bool) {
	keys, err := host.KVStoreList("presence:")
	if err != nil {
		return
	}
	for _, k := range keys {
		user := strings.TrimPrefix(k, "presence:")
		if configured[user] {
			continue
		}
		s, ok := kvStore{}.Load(user)
		if !ok || s.Access == "" {
			continue
		}
		if ps := loadPresence(user); ps.SessionToken != "" {
			creds := presence.Creds{Access: s.Access, ClientID: s.ClientID}
			if err := (discord.Publisher{D: hostDoer{}}).Clear(user, ps.SessionToken, creds); err != nil {
				pdk.Log(pdk.LogWarn, "navirpc: could not clear the card for retired user "+user+": "+err.Error())
			}
		}
	}
	for _, prefix := range []string{"token:", "playback:", "presence:"} {
		keys, err := host.KVStoreList(prefix)
		if err != nil {
			continue
		}
		for _, k := range keys {
			if !configured[strings.TrimPrefix(k, prefix)] {
				_ = host.KVStoreDelete(k)
			}
		}
	}
}

// the token field is a raw refresh token or the connect page's auth unit {token, client_id}.
// client_id is empty when the central app was used, the caller falls back to it.
func authFrom(token string) (seed, clientID string) {
	var blob struct {
		Token    string `json:"token"`
		ClientID string `json:"client_id"`
	}
	if json.Unmarshal([]byte(token), &blob) == nil && blob.Token != "" {
		return blob.Token, blob.ClientID
	}
	return token, ""
}

func (plugin) PlaybackReport(r scrobbler.PlaybackReportRequest) error {
	nowMs := time.Now().UnixMilli()
	st := loadSnapshot(r.Username)
	us := presence.RestoreUserState(clearDebounceMs, st.Snapshot)

	var desired presence.Desired
	switch r.State {
	case "playing", "starting":
		u, ok := findUser(r.Username)
		if !ok {
			return nil // gone from config, the authorization gate catches this first, belt and braces
		}
		l := parseLook(r.Username, u.Config)
		tk := track(r)
		act := presence.Map(tk, l.Prefs(), r.PositionMs, nowMs)
		// resolve art only on an album change, keyed off the raw track identity. the
		// rendered state text is a user template and can be anything
		m := art.Meta{RGID: tk.RGID, AlbumID: tk.AlbumID, Artist: tk.Artist, Album: tk.Album}
		if key := art.Key(m); key != "" && key == st.ArtKey {
			act.LargeImage = st.ArtURL
		} else {
			act.LargeImage, _ = art.Chain(art.Build(configuredArtProviders()), m, httpGetter{})
			st.ArtKey, st.ArtURL = key, act.LargeImage
		}
		desired, _ = us.OnReport(r.State, act, nowMs)
	case "paused", "stopped", "expired":
		us.OnReport(r.State, presence.Activity{}, nowMs)
		desired, _ = us.Due(nowMs)
	default:
		return nil
	}
	st.Snapshot = us.Snapshot()
	saveSnapshot(r.Username, st)

	if desired.Seq == 0 {
		return nil
	}
	s, err := auth.EnsureFresh(r.Username, kvStore{}, discord.Refresher{D: hostDoer{}}, time.Now().Unix())
	if err != nil {
		pdk.Log(pdk.LogError, "navirpc: token for "+r.Username+" unusable, reconnect: "+err.Error())
		return nil
	}
	creds := presence.Creds{Access: s.Access, ClientID: s.ClientID}
	ps, err := presence.Reconcile(r.Username, desired, loadPresence(r.Username), discord.Publisher{D: hostDoer{}}, creds, nowMs)
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
			continue // no session, nothing to keep alive or clear, only the report path creates one
		}
		desired := presence.Desired{Seq: snap.Seq, Kind: snap.LastKind, Act: snap.LastAct}
		creds := presence.Creds{Access: s.Access, ClientID: s.ClientID}
		ps, err := presence.Reconcile(u.Username, desired, ps, discord.Publisher{D: hostDoer{}}, creds, nowMs)
		if err != nil {
			pdk.Log(pdk.LogWarn, "navirpc: tick for "+u.Username+" failed: "+err.Error())
		}
		savePresence(u.Username, ps)
	}
	return nil
}

func track(r scrobbler.PlaybackReportRequest) presence.Track {
	return presence.Track{
		Title: r.Track.Title, Artist: r.Track.Artist, AlbumArtist: r.Track.AlbumArtist, Album: r.Track.Album,
		RGID: r.Track.MBZReleaseGroupID, AlbumID: r.Track.MBZAlbumID,
		DurationMs: int64(r.Track.Duration * 1000),
	}
}

// the per-user look, authored on the connect page and pasted into the config field. an
// empty field or an absent key falls back to the default card.
func parseLook(username, config string) presence.Look {
	var l presence.Look
	if config == "" {
		return l
	}
	if json.Unmarshal([]byte(config), &l) != nil {
		pdk.Log(pdk.LogWarn, "navirpc: look config for "+username+" is not valid json, using the default card")
	}
	return l
}

// an absent or blank setting defaults to caa then itunes, an explicit empty list is no art.
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
	if _, ok := findUser(r.Username); !ok {
		return false, nil
	}
	s, ok := kvStore{}.Load(r.Username)
	return ok && !s.Dead && s.Refresh != "", nil
}

func (plugin) NowPlaying(scrobbler.NowPlayingRequest) error { return nil }
func (plugin) Scrobble(scrobbler.ScrobbleRequest) error     { return nil }

func main() {}

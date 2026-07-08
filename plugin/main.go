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
	for _, u := range readUsers() {
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
		clearState(u.Username) // drop stale playback/presence so the first report starts fresh
	}
	if _, err := host.SchedulerScheduleRecurring(tickCron, "", "keepalive"); err != nil {
		pdk.Log(pdk.LogWarn, "navirpc: scheduler register failed: "+err.Error())
	}
	return nil
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
		u, _ := findUser(r.Username)
		l := parseLook(u.Config)
		tk := track(r)
		act := presence.Map(tk, l.prefs(), r.PositionMs, nowMs)
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
			continue // no session, nothing to keep alive or clear, only the report path creates one
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
		Title: r.Track.Title, Artist: r.Track.Artist, AlbumArtist: r.Track.AlbumArtist, Album: r.Track.Album,
		RGID: r.Track.MBZReleaseGroupID, AlbumID: r.Track.MBZAlbumID,
		DurationMs: int64(r.Track.Duration * 1000),
	}
}

// the per-user look, authored on the connect page and pasted into the config field. an
// empty field or an absent key falls back to the default card.
type look struct {
	Type              string            `json:"type"`
	Header            string            `json:"header"`
	Details           string            `json:"details"`
	State             string            `json:"state"`
	DetailsURL        string            `json:"details_url"`
	StateURL          string            `json:"state_url"`
	StatusDisplayType string            `json:"status_display_type"`
	LargeText         string            `json:"large_text"`
	SmallImage        string            `json:"small_image"`
	SmallText         string            `json:"small_text"`
	Buttons           []presence.Button `json:"buttons"`
}

func parseLook(config string) look {
	var l look
	if config != "" {
		json.Unmarshal([]byte(config), &l)
	}
	return l
}

func (l look) prefs() presence.Prefs {
	return presence.Prefs{
		Type:              orElse(l.Type, "listening"),
		Header:            orElse(l.Header, "{artist}"),
		Details:           orElse(l.Details, "{track}"),
		State:             orElse(l.State, "{album}"),
		DetailsURL:        l.DetailsURL,
		StateURL:          l.StateURL,
		StatusDisplayType: orElse(l.StatusDisplayType, "name"),
		LargeText:         l.LargeText,
		SmallImage:        l.SmallImage,
		SmallText:         l.SmallText,
		Buttons:           l.Buttons,
	}
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

func orElse(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

func (plugin) IsAuthorized(r scrobbler.IsAuthorizedRequest) (bool, error) {
	s, ok := kvStore{}.Load(r.Username)
	return ok && !s.Dead && s.Refresh != "", nil
}

func (plugin) NowPlaying(scrobbler.NowPlayingRequest) error { return nil }
func (plugin) Scrobble(scrobbler.ScrobbleRequest) error     { return nil }

func main() {}

//go:build wasip1

package main

import (
	"encoding/json"
	"errors"
	"net/url"
	"strconv"
	"strings"

	"github.com/navidrome/navidrome/plugins/pdk/go/host"
	"github.com/rxtted/navirpc/internal/auth"
	"github.com/rxtted/navirpc/internal/presence"
)

const apiBase = "https://discord.com/api/v10"

// discord is behind cloudflare, which 1010-blocks a default http agent.
const userAgent = "Mozilla/5.0 navirpc/0.1"

type discordRefresher struct{}

func (discordRefresher) Refresh(clientID, refreshToken string) (access, newRefresh string, expiresIn int64, err error) {
	form := url.Values{"grant_type": {"refresh_token"}, "client_id": {clientID}, "refresh_token": {refreshToken}}
	resp, err := host.HTTPSend(host.HTTPRequest{
		Method:    "POST",
		URL:       apiBase + "/oauth2/token",
		Headers:   map[string]string{"Content-Type": "application/x-www-form-urlencoded", "User-Agent": userAgent},
		Body:      []byte(form.Encode()),
		TimeoutMs: 10000,
	})
	if err != nil {
		return "", "", 0, err
	}
	if resp.StatusCode == 400 {
		return "", "", 0, auth.ErrInvalidGrant
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", "", 0, errors.New("token refresh http " + strconv.Itoa(int(resp.StatusCode)))
	}
	var out struct {
		Access    string `json:"access_token"`
		Refresh   string `json:"refresh_token"`
		ExpiresIn int64  `json:"expires_in"`
	}
	if err := json.Unmarshal(resp.Body, &out); err != nil {
		return "", "", 0, err
	}
	return out.Access, out.Refresh, out.ExpiresIn, nil
}

// discordPublisher implements presence.Publisher, resolving each user's access token
// from the kv-store where the reconciler refreshes it before calling here.
type discordPublisher struct{}

func (discordPublisher) Publish(username string, d presence.Desired, sessionToken string) (string, error) {
	s, ok := kvStore{}.Load(username)
	if !ok || s.Access == "" {
		return "", errors.New("no access token for " + username)
	}
	payload := map[string]any{"activities": []discordActivity{toDiscordActivity(d.Act, s.ClientID)}}
	if sessionToken != "" {
		payload["token"] = sessionToken // update the existing session rather than create a new one
	}
	body, _ := json.Marshal(payload)
	resp, err := discordPost("/users/@me/headless-sessions", s.Access, body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode == 429 {
		return "", rateLimited{ms: retryAfterMs(resp.Headers)}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", errors.New("headless-session http " + strconv.Itoa(int(resp.StatusCode)))
	}
	var out struct {
		Token string `json:"token"`
	}
	json.Unmarshal(resp.Body, &out)
	if out.Token != "" {
		return out.Token, nil
	}
	return sessionToken, nil
}

func (discordPublisher) Clear(username, sessionToken string) error {
	if sessionToken == "" {
		return nil
	}
	s, ok := kvStore{}.Load(username)
	if !ok || s.Access == "" {
		return nil
	}
	body, _ := json.Marshal(map[string]string{"token": sessionToken})
	_, err := discordPost("/users/@me/headless-sessions/delete", s.Access, body)
	return err
}

// httpGetter is art.Getter over the host client for lookup providers.
type httpGetter struct{}

func (httpGetter) Get(url string) ([]byte, int, error) {
	resp, err := host.HTTPSend(host.HTTPRequest{
		Method:    "GET",
		URL:       url,
		Headers:   map[string]string{"User-Agent": userAgent},
		TimeoutMs: 10000,
	})
	if err != nil {
		return nil, 0, err
	}
	return resp.Body, int(resp.StatusCode), nil
}

func discordPost(path, bearer string, body []byte) (*host.HTTPResponse, error) {
	return host.HTTPSend(host.HTTPRequest{
		Method:    "POST",
		URL:       apiBase + path,
		Headers:   map[string]string{"Content-Type": "application/json", "Authorization": "Bearer " + bearer, "User-Agent": userAgent},
		Body:      body,
		TimeoutMs: 10000,
	})
}

type discordActivity struct {
	Type              int                `json:"type"`
	Name              string             `json:"name"`
	Platform          string             `json:"platform"`
	ApplicationID     string             `json:"application_id,omitempty"`
	Details           string             `json:"details,omitempty"`
	DetailsURL        string             `json:"details_url,omitempty"`
	State             string             `json:"state,omitempty"`
	StateURL          string             `json:"state_url,omitempty"`
	StatusDisplayType int                `json:"status_display_type,omitempty"`
	Timestamps        *discordTimestamps `json:"timestamps,omitempty"`
	Assets            *discordAssets     `json:"assets,omitempty"`
	Buttons           []discordButton    `json:"buttons,omitempty"`
}

type discordTimestamps struct {
	Start int64 `json:"start,omitempty"`
	End   int64 `json:"end,omitempty"`
}

type discordAssets struct {
	LargeImage string `json:"large_image,omitempty"`
	LargeText  string `json:"large_text,omitempty"`
	SmallImage string `json:"small_image,omitempty"`
	SmallText  string `json:"small_text,omitempty"`
}

// buttons go up as objects, discord normalizes them to a label array plus metadata.
type discordButton struct {
	Label string `json:"label"`
	URL   string `json:"url"`
}

func toDiscordActivity(a presence.Activity, clientID string) discordActivity {
	da := discordActivity{
		Type: a.Type, Name: a.Name, Platform: a.Platform, ApplicationID: clientID,
		Details: a.Details, DetailsURL: a.DetailsURL,
		State: a.State, StateURL: a.StateURL,
		StatusDisplayType: a.StatusDisplayType,
	}
	if a.Start != 0 || a.End != 0 {
		da.Timestamps = &discordTimestamps{Start: a.Start, End: a.End}
	}
	if a.LargeImage != "" || a.LargeText != "" || a.SmallImage != "" || a.SmallText != "" {
		da.Assets = &discordAssets{LargeImage: a.LargeImage, LargeText: a.LargeText, SmallImage: a.SmallImage, SmallText: a.SmallText}
	}
	for _, b := range a.Buttons {
		if b.Label != "" && b.URL != "" {
			da.Buttons = append(da.Buttons, discordButton{Label: b.Label, URL: b.URL})
		}
	}
	return da
}

// rateLimited carries discord's retry window so the reconciler backs off by exactly that.
type rateLimited struct{ ms int64 }

func (e rateLimited) Error() string {
	return "headless-session rate limited, retry in " + strconv.FormatInt(e.ms, 10) + "ms"
}
func (e rateLimited) RetryAfterMs() int64 { return e.ms }

func retryAfterMs(h map[string]string) int64 {
	for _, k := range []string{"Retry-After", "X-RateLimit-Reset-After"} {
		if v := headerGet(h, k); v != "" {
			if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 {
				return int64(f * 1000)
			}
		}
	}
	return 2000
}

func headerGet(h map[string]string, key string) string {
	for k, v := range h {
		if strings.EqualFold(k, key) {
			return v
		}
	}
	return ""
}

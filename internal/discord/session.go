package discord

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"atrophy/navirpc/internal/presence"
)

var _ presence.Publisher = Publisher{}

type Publisher struct {
	D Doer
}

func (p Publisher) Publish(user string, d presence.Desired, sessionToken string, c presence.Creds) (string, error) {
	if c.Access == "" {
		return "", errors.New("no access token for " + user)
	}
	payload := map[string]any{"activities": []wireActivity{toWire(d.Act, c.ClientID)}}
	if sessionToken != "" {
		payload["token"] = sessionToken // update the existing session rather than create a new one
	}
	body, _ := json.Marshal(payload)
	resp, err := p.post("/users/@me/headless-sessions", c.Access, body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode == 429 {
		return "", rateLimited{ms: retryAfterMs(resp.Headers)}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", errors.New("headless-session http " + strconv.Itoa(resp.StatusCode))
	}
	var out struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(resp.Body, &out); err != nil || out.Token == "" {
		if sessionToken == "" {
			// the create succeeded at discord but handed back no usable handle, record
			// failure so the reconciler retries rather than recording a session it can
			// never update or delete
			return "", errors.New("create returned no session token")
		}
		return sessionToken, nil // an update, the held token stays valid regardless
	}
	return out.Token, nil
}

func (p Publisher) Clear(_, sessionToken string, c presence.Creds) error {
	if sessionToken == "" || c.Access == "" {
		return nil
	}
	body, _ := json.Marshal(map[string]string{"token": sessionToken})
	resp, err := p.post("/users/@me/headless-sessions/delete", c.Access, body)
	if err != nil {
		return err
	}
	switch {
	case resp.StatusCode == 429:
		return rateLimited{ms: retryAfterMs(resp.Headers)}
	case resp.StatusCode == 400:
		// a 400 here is 50014 invalid token, unaddressable and never going to
		// work on a retry. drop the handle, a real session ttls out on its own
		return nil
	case resp.StatusCode < 200 || resp.StatusCode >= 300:
		return errors.New("headless-session delete http " + strconv.Itoa(resp.StatusCode))
	}
	return nil
}

func (p Publisher) post(path, bearer string, body []byte) (Response, error) {
	return p.D.Do(Request{
		Method:    "POST",
		URL:       apiBase + path,
		Headers:   map[string]string{"Content-Type": "application/json", "Authorization": "Bearer " + bearer, "User-Agent": UserAgent},
		Body:      body,
		TimeoutMs: 10000,
	})
}

// rateLimited carries discord's retry window so the reconciler backs off by exactly that
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

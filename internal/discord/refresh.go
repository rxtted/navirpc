package discord

import (
	"encoding/json"
	"errors"
	"net/url"
	"strconv"
	"strings"

	"atrophy/navirpc/internal/auth"
)

var _ auth.Refresher = Refresher{}

// the first line of whatever discord sent, enough to log without dumping a cdn error page
func snippet(b []byte) string {
	s := strings.TrimSpace(string(b))
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	if len(s) > 80 {
		s = s[:80]
	}
	return s
}

type Refresher struct {
	D Doer
}

func (r Refresher) Refresh(clientID, refreshToken string) (access, newRefresh string, expiresIn int64, err error) {
	form := url.Values{"grant_type": {"refresh_token"}, "client_id": {clientID}, "refresh_token": {refreshToken}}
	resp, err := r.D.Do(Request{
		Method:    "POST",
		URL:       apiBase + "/oauth2/token",
		Headers:   map[string]string{"Content-Type": "application/x-www-form-urlencoded", "User-Agent": UserAgent},
		Body:      []byte(form.Encode()),
		TimeoutMs: 10000,
	})
	if err != nil {
		return "", "", 0, err
	}
	if resp.StatusCode == 400 {
		var e struct {
			Err string `json:"error"`
		}
		json.Unmarshal(resp.Body, &e) //nolint:errcheck // an unparseable 400 stays transient below, ambiguity never gets the kill verdict
		if e.Err == "invalid_grant" {
			return "", "", 0, auth.ErrInvalidGrant
		}
		// only a real dead-grant verdict kills a stored token, the kill is irreversible.
		// widen this condition and someone's redoing oauth over a cdn whoopsie
		reason := e.Err
		if reason == "" {
			reason = snippet(resp.Body)
		}
		return "", "", 0, errors.New("token refresh 400 " + reason)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", "", 0, errors.New("token refresh http " + strconv.Itoa(resp.StatusCode))
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

package discord

import (
	"encoding/json"
	"errors"
	"net/url"
	"strconv"

	"atrophy/navirpc/internal/auth"
)

var _ auth.Refresher = Refresher{}

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
		return "", "", 0, auth.ErrInvalidGrant
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

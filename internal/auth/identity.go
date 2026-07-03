package auth

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
)

// reads the discord user id from an id_token's `sub` claim.

// the id is kept only for a connect-page "connected as" display and for logs. if no
// display/debug use happens, drop this and the config field.
func ParseIDTokenSub(idToken string) (string, error) {
	parts := strings.Split(idToken, ".")
	if len(parts) < 2 {
		return "", errors.New("malformed id_token")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", err
	}
	var claims struct {
		Sub string `json:"sub"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return "", err
	}
	if claims.Sub == "" {
		return "", errors.New("id_token has no sub")
	}
	return claims.Sub, nil
}

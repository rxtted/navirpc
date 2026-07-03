package auth

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
)

// reads the discord user id from an id_token's `sub` claim. no signature check: the
// token came from our own oauth exchange, we only need the subject.
//
// the id is kept only for a connect-page "connected as" display and for logs. it's a
// removal candidate: the fail-closed bind check it was added for was cut, because the id
// rides in the same config unit as the token, so it can't catch a mis-pasted unit. if no
// display/debug use lands, drop this and the DiscordUserID config field.
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

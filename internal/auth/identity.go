package auth

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
)

var ErrIdentityMismatch = errors.New("discord account mismatch")

// reads the discord user id from an id_token's `sub` claim. no signature check: the
// token came from our own oauth exchange, we only need the subject.
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

// fails closed when a bound account id doesn't match the live one; an empty declared id
// (never bound) passes.
func BindCheck(declared, resolved string) error {
	if declared != "" && declared != resolved {
		return ErrIdentityMismatch
	}
	return nil
}

package auth

import (
	"encoding/base64"
	"testing"
)

func idToken(payload string) string {
	return "hdr." + base64.RawURLEncoding.EncodeToString([]byte(payload)) + ".sig"
}

func TestParseIDTokenSub(t *testing.T) {
	sub, err := ParseIDTokenSub(idToken(`{"sub":"110517284401657037","iss":"https://discord.com"}`))
	if err != nil || sub != "110517284401657037" {
		t.Fatalf("sub=%q err=%v", sub, err)
	}
	if _, err := ParseIDTokenSub("not-a-jwt"); err == nil {
		t.Fatal("malformed token should error")
	}
	if _, err := ParseIDTokenSub(idToken(`{"iss":"x"}`)); err == nil {
		t.Fatal("missing sub should error")
	}
}

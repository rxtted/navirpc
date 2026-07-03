//go:build wasip1

package main

import (
	"encoding/json"

	"github.com/navidrome/navidrome/plugins/pdk/go/host"
	"github.com/rxtted/navirpc/internal/auth"
)

// kvStore adapts navidrome's kv-store to auth.TokenStore, keyed per navidrome user.
type kvStore struct{}

func (kvStore) Load(username string) (auth.Stored, bool) {
	b, ok, err := host.KVStoreGet("tok:" + username)
	if err != nil || !ok {
		return auth.Stored{}, false
	}
	var s auth.Stored
	if json.Unmarshal(b, &s) != nil {
		return auth.Stored{}, false
	}
	return s, true
}

func (kvStore) Save(username string, s auth.Stored) error {
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return host.KVStoreSet("tok:"+username, b)
}

func loadSession(username string) string {
	b, ok, err := host.KVStoreGet("sess:" + username)
	if err != nil || !ok {
		return ""
	}
	return string(b)
}

func saveSession(username, token string) { _ = host.KVStoreSet("sess:"+username, []byte(token)) }
func clearSession(username string)       { _ = host.KVStoreDelete("sess:" + username) }

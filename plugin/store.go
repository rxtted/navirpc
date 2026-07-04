//go:build wasip1

package main

import (
	"encoding/json"

	"github.com/navidrome/navidrome/plugins/pdk/go/host"
	"github.com/rxtted/navirpc/internal/auth"
	"github.com/rxtted/navirpc/internal/presence"
)

// kvStore adapts navidrome's kv-store to auth.TokenStore, keyed per navidrome user.
type kvStore struct{}

func (kvStore) Load(username string) (auth.Stored, bool) {
	b, ok, err := host.KVStoreGet("token:" + username)
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
	return host.KVStoreSet("token:"+username, b)
}

// per-user state is split by owner: playback: is the report path's snapshot (only it
// writes it), presence: is the shared publish state the scheduler tick also updates.
// keeping them separate is what lets the tick run without a lock, see the spec.

func loadSnapshot(username string) presence.Snapshot {
	var snap presence.Snapshot
	if b, ok, err := host.KVStoreGet("playback:" + username); err == nil && ok {
		json.Unmarshal(b, &snap)
	}
	return snap
}

func saveSnapshot(username string, snap presence.Snapshot) {
	if b, err := json.Marshal(snap); err == nil {
		_ = host.KVStoreSet("playback:"+username, b)
	}
}

func loadPresence(username string) presence.PubState {
	var ps presence.PubState
	if b, ok, err := host.KVStoreGet("presence:" + username); err == nil && ok {
		json.Unmarshal(b, &ps)
	}
	return ps
}

func savePresence(username string, ps presence.PubState) {
	if b, err := json.Marshal(ps); err == nil {
		_ = host.KVStoreSet("presence:"+username, b)
	}
}

func clearState(username string) {
	_ = host.KVStoreDelete("playback:" + username)
	_ = host.KVStoreDelete("presence:" + username)
}

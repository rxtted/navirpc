package auth

const refreshMarginSec = 3600

type Stored struct {
	Seed      string // the refresh token the user pasted, the config value
	ClientID  string
	Refresh   string // current refresh token, rotates, starts as Seed
	Access    string
	ExpiresAt int64
	Dead      bool
}

// a changed seed or client id is a reconnect or own-app switch, re-adopt from the new seed
// and drop any dead flag.
func Reconcile(cfgSeed, cfgClientID string, cur *Stored) *Stored {
	if cur == nil || cur.Seed != cfgSeed || cur.ClientID != cfgClientID {
		return &Stored{Seed: cfgSeed, ClientID: cfgClientID, Refresh: cfgSeed}
	}
	return cur
}

func NeedsRefresh(cur Stored, nowUnix int64) bool {
	return !cur.Dead && cur.Refresh != "" && nowUnix >= cur.ExpiresAt-refreshMarginSec
}

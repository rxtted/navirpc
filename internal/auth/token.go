package auth

const refreshMarginSec = 3600

type Stored struct {
	Seed          string // the refresh token the user pasted, the config value
	ClientID      string
	Refresh       string // current refresh token, rotates, starts as Seed
	Access        string
	ExpiresAt     int64
	Dead          bool
	DiscordUserID string
}

// adopts a fresh config unit when the pasted seed, client id, or bound user changes.
// a reconnect or an own-app switch resets the live token from the new seed and clears Dead.
// an unchanged config keeps the stored live state.
func Reconcile(cfgSeed, cfgClientID, cfgUserID string, cur *Stored) *Stored {
	if cur == nil || cur.Seed != cfgSeed || cur.ClientID != cfgClientID {
		return &Stored{Seed: cfgSeed, ClientID: cfgClientID, Refresh: cfgSeed, DiscordUserID: cfgUserID}
	}
	return cur
}

func NeedsRefresh(cur Stored, nowUnix int64) bool {
	return !cur.Dead && cur.Refresh != "" && nowUnix >= cur.ExpiresAt-refreshMarginSec
}

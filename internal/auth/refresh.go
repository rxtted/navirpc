package auth

import "errors"

var ErrInvalidGrant = errors.New("invalid_grant")

type TokenStore interface {
	Load(userID string) (Stored, bool)
	Save(userID string, s Stored) error
}

type Refresher interface {
	Refresh(clientID, refreshToken string) (access, newRefresh string, expiresIn int64, err error)
}

// the single refresh call site (single-owner: only the reconciler calls it). when the
// access token is stale it refreshes, persists the rotated refresh token BEFORE handing
// back the new access token (write-before-use), and marks the token dead on invalid_grant.
// a persist failure returns an error rather than an unpersisted token, so the next tick
// retries from stored state instead of stranding the account on a rotated-away token.
func EnsureFresh(userID string, store TokenStore, rf Refresher, now int64) (Stored, error) {
	cur, ok := store.Load(userID)
	if !ok {
		return Stored{}, errors.New("no token for user")
	}
	if cur.Dead {
		return cur, ErrInvalidGrant
	}
	if !NeedsRefresh(cur, now) {
		return cur, nil
	}

	access, newRefresh, expiresIn, err := rf.Refresh(cur.ClientID, cur.Refresh)
	if err != nil {
		if errors.Is(err, ErrInvalidGrant) {
			cur.Dead = true
			_ = store.Save(userID, cur)
			return cur, ErrInvalidGrant
		}
		return Stored{}, err
	}

	next := cur
	if newRefresh != "" {
		next.Refresh = newRefresh
	}
	next.Access = access
	next.ExpiresAt = now + expiresIn
	if err := store.Save(userID, next); err != nil {
		return Stored{}, err
	}
	return next, nil
}

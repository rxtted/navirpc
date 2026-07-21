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

// persist the rotated refresh token before using the new access token. discord
// invalidates the old refresh on use, so a save that lands late would strand the account.
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
			// a lost dead flag means every report retries the doomed refresh forever,
			// so a failed save rides out with the sentinel instead of vanishing
			if saveErr := store.Save(userID, cur); saveErr != nil {
				return cur, errors.Join(ErrInvalidGrant, saveErr)
			}
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

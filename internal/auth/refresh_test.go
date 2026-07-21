package auth

import (
	"errors"
	"testing"
)

type fakeStore struct {
	s       Stored
	ok      bool
	saved   *Stored
	saveErr error
}

func (f *fakeStore) Load(string) (Stored, bool) { return f.s, f.ok }
func (f *fakeStore) Save(_ string, s Stored) error {
	if f.saveErr != nil {
		return f.saveErr
	}
	cp := s
	f.saved = &cp
	f.s = s
	return nil
}

type fakeRefresher struct {
	access, refresh string
	expiresIn       int64
	err             error
	calls           int
}

func (f *fakeRefresher) Refresh(string, string) (string, string, int64, error) {
	f.calls++
	return f.access, f.refresh, f.expiresIn, f.err
}

func TestEnsureFresh_ValidTokenNoRefresh(t *testing.T) {
	store := &fakeStore{s: Stored{Refresh: "r", ExpiresAt: 1_000_000}, ok: true}
	rf := &fakeRefresher{}
	got, err := EnsureFresh("u", store, rf, 100)
	if err != nil || rf.calls != 0 || got.Access != "" {
		t.Fatalf("valid token should not refresh: err=%v calls=%d", err, rf.calls)
	}
}

func TestEnsureFresh_PersistsRotatedTokenBeforeReturn(t *testing.T) {
	store := &fakeStore{s: Stored{Refresh: "old", ExpiresAt: 0}, ok: true}
	rf := &fakeRefresher{access: "new-at", refresh: "new-rt", expiresIn: 604800}
	got, err := EnsureFresh("u", store, rf, 1000)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if store.saved == nil || store.saved.Refresh != "new-rt" {
		t.Fatal("rotated refresh token must be persisted (write-before-use)")
	}
	if got.Access != "new-at" || got.ExpiresAt != 1000+604800 {
		t.Fatalf("returned token wrong: %+v", got)
	}
}

func TestEnsureFresh_SaveFailureReturnsError(t *testing.T) {
	store := &fakeStore{s: Stored{Refresh: "old", ExpiresAt: 0}, ok: true, saveErr: errors.New("kv down")}
	rf := &fakeRefresher{access: "at", refresh: "rt", expiresIn: 100}
	got, err := EnsureFresh("u", store, rf, 1000)
	if err == nil || got.Access != "" {
		t.Fatalf("save failure must not return a success token: got=%+v err=%v", got, err)
	}
}

func TestEnsureFresh_InvalidGrantMarksDead(t *testing.T) {
	store := &fakeStore{s: Stored{Refresh: "old", ExpiresAt: 0}, ok: true}
	rf := &fakeRefresher{err: ErrInvalidGrant}
	got, err := EnsureFresh("u", store, rf, 1000)
	if !errors.Is(err, ErrInvalidGrant) || !got.Dead {
		t.Fatalf("invalid_grant should mark dead: got=%+v err=%v", got, err)
	}
	if store.saved == nil || !store.saved.Dead {
		t.Fatal("dead flag must be persisted")
	}
}

func TestEnsureFresh_DeadMarkSaveFailureSurfaces(t *testing.T) {
	diskErr := errors.New("disk full")
	store := &fakeStore{s: Stored{Refresh: "rt", ExpiresAt: 0}, ok: true, saveErr: diskErr}
	rf := &fakeRefresher{err: ErrInvalidGrant}
	_, err := EnsureFresh("u", store, rf, 1000)
	if !errors.Is(err, ErrInvalidGrant) || !errors.Is(err, diskErr) {
		t.Fatalf("dead-mark save failure rides with the sentinel: %v", err)
	}
}

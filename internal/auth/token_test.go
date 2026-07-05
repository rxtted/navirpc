package auth

import "testing"

func TestReconcile_NewSeedAdoptsAndClearsDead(t *testing.T) {
	cur := &Stored{Seed: "old", ClientID: "c", Refresh: "rotated", Dead: true}
	got := Reconcile("new", "c", cur)
	if got.Seed != "new" || got.Refresh != "new" || got.Dead {
		t.Fatalf("new seed should adopt and clear dead: %+v", got)
	}
}

func TestReconcile_NewClientIDForcesReseed(t *testing.T) {
	cur := &Stored{Seed: "s", ClientID: "old", Refresh: "rotated"}
	got := Reconcile("s", "new", cur)
	if got.ClientID != "new" || got.Refresh != "s" {
		t.Fatalf("client id change should reseed: %+v", got)
	}
}

func TestReconcile_UnchangedKeepsLiveToken(t *testing.T) {
	cur := &Stored{Seed: "s", ClientID: "c", Refresh: "rotated", Access: "at"}
	got := Reconcile("s", "c", cur)
	if got != cur || got.Refresh != "rotated" {
		t.Fatalf("unchanged config should keep the live token: %+v", got)
	}
}

func TestNeedsRefresh(t *testing.T) {
	if !NeedsRefresh(Stored{Refresh: "r", ExpiresAt: 0}, 100) {
		t.Fatal("no access token yet (ExpiresAt 0) should need refresh")
	}
	if NeedsRefresh(Stored{Refresh: "r", ExpiresAt: 1_000_000}, 100) {
		t.Fatal("far from expiry should not need refresh")
	}
	if !NeedsRefresh(Stored{Refresh: "r", ExpiresAt: 100 + refreshMarginSec}, 200) {
		t.Fatal("inside the margin should need refresh")
	}
	if NeedsRefresh(Stored{Refresh: "r", ExpiresAt: 1_000_000, Dead: true}, 999_999) {
		t.Fatal("a dead token should never refresh")
	}
}

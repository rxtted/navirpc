package presence

import (
	"errors"
	"testing"
)

type fakePub struct {
	published  []Desired
	gotSession []string
	cleared    []string
	session    string
	err        error
}

func (f *fakePub) Publish(_ string, d Desired, sessionToken string) (string, error) {
	f.published = append(f.published, d)
	f.gotSession = append(f.gotSession, sessionToken)
	return f.session, f.err
}
func (f *fakePub) Clear(_, sessionToken string) error {
	f.cleared = append(f.cleared, sessionToken)
	return f.err
}

func TestReconcile_PublishesNewer(t *testing.T) {
	pub := &fakePub{session: "sess"}
	ps, err := Reconcile("u", Desired{Seq: 5, Kind: "play"}, PubState{PublishedSeq: 2}, pub, 1000)
	if err != nil || len(pub.published) != 1 || ps.PublishedSeq != 5 || ps.SessionToken != "sess" {
		t.Fatalf("should publish and advance: ps=%+v pub=%+v", ps, pub)
	}
}

func TestReconcile_DropsStale(t *testing.T) {
	pub := &fakePub{}
	ps, _ := Reconcile("u", Desired{Seq: 2, Kind: "play"}, PubState{PublishedSeq: 5}, pub, 1000)
	if len(pub.published) != 0 || ps.PublishedSeq != 5 {
		t.Fatalf("stale desired should be dropped: pub=%+v", pub)
	}
}

func TestReconcile_PublishUsesExistingSession(t *testing.T) {
	pub := &fakePub{session: "new"}
	ps, _ := Reconcile("u", Desired{Seq: 6, Kind: "play"}, PubState{PublishedSeq: 5, SessionToken: "old"}, pub, 1000)
	if len(pub.gotSession) != 1 || pub.gotSession[0] != "old" {
		t.Fatalf("publish should pass the existing session token (update, not create): %+v", pub.gotSession)
	}
	if ps.SessionToken != "new" {
		t.Fatalf("session token should become the returned one: ps=%+v", ps)
	}
}

func TestReconcile_KeepaliveRepublishesUnchanged(t *testing.T) {
	pub := &fakePub{session: "sess"}
	ps, _ := Reconcile("u", Desired{Seq: 5, Kind: "play"},
		PubState{PublishedSeq: 5, SessionToken: "sess", LastPublishMs: 1000}, pub, 1000+keepaliveMs)
	if len(pub.published) != 1 {
		t.Fatalf("keepalive should re-publish an unchanged session: %+v", pub.published)
	}
	if ps.LastPublishMs != 1000+keepaliveMs {
		t.Fatalf("keepalive should reset LastPublishMs: ps=%+v", ps)
	}
}

func TestReconcile_NoKeepaliveWithinWindow(t *testing.T) {
	pub := &fakePub{session: "sess"}
	ps, _ := Reconcile("u", Desired{Seq: 5, Kind: "play"},
		PubState{PublishedSeq: 5, SessionToken: "sess", LastPublishMs: 1000}, pub, 2000)
	if len(pub.published) != 0 || ps.PublishedSeq != 5 {
		t.Fatalf("no keepalive within the window: pub=%+v", pub)
	}
}

func TestReconcile_ClearUsesSessionToken(t *testing.T) {
	pub := &fakePub{}
	ps, err := Reconcile("u", Desired{Seq: 6, Kind: "clear"}, PubState{PublishedSeq: 5, SessionToken: "sess"}, pub, 1000)
	if err != nil || len(pub.cleared) != 1 || pub.cleared[0] != "sess" || ps.SessionToken != "" {
		t.Fatalf("clear should call Clear with the session and reset it: ps=%+v pub=%+v", ps, pub)
	}
}

func TestReconcile_FailedClearRetries(t *testing.T) {
	pub := &fakePub{err: errors.New("discord 429")}
	ps, err := Reconcile("u", Desired{Seq: 6, Kind: "clear"}, PubState{PublishedSeq: 5, SessionToken: "sess"}, pub, 1000)
	if err == nil || ps.SessionToken != "sess" || ps.PublishedSeq != 5 {
		t.Fatalf("a failed clear should keep the session for the retry: ps=%+v", ps)
	}
	pub.err = nil
	ps, err = Reconcile("u", Desired{Seq: 6, Kind: "clear"}, ps, pub, 2000)
	if err != nil || ps.SessionToken != "" || ps.PublishedSeq != 6 || len(pub.cleared) != 2 {
		t.Fatalf("the retried clear should land and reset the session: ps=%+v pub=%+v", ps, pub)
	}
}

func TestReconcile_ClearWithoutSessionNoOps(t *testing.T) {
	pub := &fakePub{}
	ps, err := Reconcile("u", Desired{Seq: 6, Kind: "clear"}, PubState{PublishedSeq: 5}, pub, 1000)
	if err != nil || len(pub.cleared) != 0 || ps.PublishedSeq != 6 {
		t.Fatalf("no session means the clear is already true: ps=%+v pub=%+v", ps, pub)
	}
}

func TestReconcile_PublishErrorBacksOff(t *testing.T) {
	pub := &fakePub{err: errors.New("discord 500")}
	ps, err := Reconcile("u", Desired{Seq: 5, Kind: "play"}, PubState{PublishedSeq: 2}, pub, 1000)
	if err == nil || ps.PublishedSeq != 2 || ps.BackoffUntil <= 1000 {
		t.Fatalf("publish error should back off and not advance seq: ps=%+v err=%v", ps, err)
	}
}

func TestReconcile_ThrottlesBurst(t *testing.T) {
	pub := &fakePub{session: "s"}
	ps := PubState{}
	var seq int64
	for i := 0; i < rateMax; i++ {
		seq++
		ps, _ = Reconcile("u", Desired{Seq: seq, Kind: "play"}, ps, pub, 1000)
	}
	if len(pub.published) != rateMax {
		t.Fatalf("first %d in the window should publish, got %d", rateMax, len(pub.published))
	}
	seq++
	ps, _ = Reconcile("u", Desired{Seq: seq, Kind: "play"}, ps, pub, 1000)
	if len(pub.published) != rateMax {
		t.Fatalf("a publish past the window cap should be throttled: %d", len(pub.published))
	}
	seq++
	ps, _ = Reconcile("u", Desired{Seq: seq, Kind: "clear"}, ps, pub, 1000)
	if len(pub.cleared) != 1 {
		t.Fatalf("clear should be exempt from the throttle: %+v", pub.cleared)
	}
	seq++
	Reconcile("u", Desired{Seq: seq, Kind: "play"}, ps, pub, 1000+rateWindowMs+1)
	if len(pub.published) != rateMax+1 {
		t.Fatalf("publishing resumes after the window ages out: %d", len(pub.published))
	}
}

type retryErr struct{ ms int64 }

func (e retryErr) Error() string       { return "rate limited" }
func (e retryErr) RetryAfterMs() int64 { return e.ms }

func TestReconcile_HonorsRetryAfter(t *testing.T) {
	pub := &fakePub{err: retryErr{ms: 1500}}
	ps, err := Reconcile("u", Desired{Seq: 5, Kind: "play"}, PubState{PublishedSeq: 2}, pub, 1000)
	if err == nil || ps.BackoffUntil != 1000+1500 {
		t.Fatalf("should back off by the retry-after window, not the generic backoff: ps=%+v err=%v", ps, err)
	}
}

func TestReconcile_SkipsDuringBackoff(t *testing.T) {
	pub := &fakePub{}
	ps, _ := Reconcile("u", Desired{Seq: 5, Kind: "play"}, PubState{PublishedSeq: 2, BackoffUntil: 9000}, pub, 1000)
	if len(pub.published) != 0 || ps.PublishedSeq != 2 {
		t.Fatalf("should skip while backing off: pub=%+v", pub)
	}
}

func TestReconcile_ClearBypassesBackoff(t *testing.T) {
	pub := &fakePub{}
	Reconcile("u", Desired{Seq: 6, Kind: "clear"}, PubState{PublishedSeq: 5, SessionToken: "sess", BackoffUntil: 9000}, pub, 1000)
	if len(pub.cleared) != 1 {
		t.Fatalf("a clear must bypass backoff and still attempt: %+v", pub.cleared)
	}
}

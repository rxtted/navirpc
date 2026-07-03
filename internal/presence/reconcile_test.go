package presence

import (
	"errors"
	"testing"
)

type fakePub struct {
	published []Desired
	cleared   []string
	session   string
	err       error
}

func (f *fakePub) Publish(_ string, d Desired) (string, error) {
	f.published = append(f.published, d)
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

func TestReconcile_ClearUsesSessionToken(t *testing.T) {
	pub := &fakePub{}
	ps, err := Reconcile("u", Desired{Seq: 6, Kind: "clear"}, PubState{PublishedSeq: 5, SessionToken: "sess"}, pub, 1000)
	if err != nil || len(pub.cleared) != 1 || pub.cleared[0] != "sess" || ps.SessionToken != "" {
		t.Fatalf("clear should call Clear with the session and reset it: ps=%+v pub=%+v", ps, pub)
	}
}

func TestReconcile_PublishErrorBacksOff(t *testing.T) {
	pub := &fakePub{err: errors.New("discord 500")}
	ps, err := Reconcile("u", Desired{Seq: 5, Kind: "play"}, PubState{PublishedSeq: 2}, pub, 1000)
	if err == nil || ps.PublishedSeq != 2 || ps.BackoffUntil <= 1000 {
		t.Fatalf("publish error should back off and not advance seq: ps=%+v err=%v", ps, err)
	}
}

func TestReconcile_SkipsDuringBackoff(t *testing.T) {
	pub := &fakePub{}
	ps, _ := Reconcile("u", Desired{Seq: 5, Kind: "play"}, PubState{PublishedSeq: 2, BackoffUntil: 9000}, pub, 1000)
	if len(pub.published) != 0 || ps.PublishedSeq != 2 {
		t.Fatalf("should skip while backing off: pub=%+v", pub)
	}
}

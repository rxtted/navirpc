package presence

type Publisher interface {
	Publish(userID string, d Desired) (sessionToken string, err error)
	Clear(userID, sessionToken string) error
}

type PubState struct {
	PublishedSeq int64
	SessionToken string
	BackoffUntil int64
	Fails        int
}

// publishes desired state to discord, newer-event-wins: a desired whose seq is not
// ahead of what's published is dropped, and a publish failure backs off (tracked in
// the returned state, not by sleeping) so the next tick retries. the caller persists
// the returned state.
func Reconcile(userID string, d Desired, ps PubState, pub Publisher, nowMs int64) (PubState, error) {
	if d.Seq <= ps.PublishedSeq || nowMs < ps.BackoffUntil {
		return ps, nil
	}

	var (
		session string
		err     error
	)
	if d.Kind == "clear" {
		err = pub.Clear(userID, ps.SessionToken)
	} else {
		session, err = pub.Publish(userID, d)
	}
	if err != nil {
		ps.Fails++
		ps.BackoffUntil = nowMs + backoffMs(ps.Fails)
		return ps, err
	}

	ps.PublishedSeq = d.Seq
	ps.Fails = 0
	ps.BackoffUntil = 0
	if d.Kind == "clear" {
		ps.SessionToken = ""
	} else {
		ps.SessionToken = session
	}
	return ps, nil
}

func backoffMs(fails int) int64 {
	const base, maxMs = 5000, 300000
	d := int64(base) << (fails - 1)
	if d <= 0 || d > maxMs {
		return maxMs
	}
	return d
}

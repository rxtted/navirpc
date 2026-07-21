package presence

import "errors"

// re-publish an unchanged session at least this often so its 20-minute discord TTL never
// lapses during steady playback.
const keepaliveMs = 15 * 60 * 1000

// discord's headless-session bucket is 5 requests / 20s. stay a step under it and coalesce
// the rest, so somebody scrub-spamming through a track never trips a 429
const (
	rateWindowMs = 20000
	rateMax      = 4
)

// a Publish error may carry discord's rate-limit window, honoured over the generic backoff.
type retryAfterErr interface{ RetryAfterMs() int64 }

// the caller proves it ran the refresh by handing the creds over, the publisher
// never reaches into storage for them
type Creds struct {
	Access   string
	ClientID string
}

type Publisher interface {
	Publish(userID string, d Desired, sessionToken string, c Creds) (newSessionToken string, err error)
	Clear(userID, sessionToken string, c Creds) error
}

type PubState struct {
	PublishedSeq  int64
	SessionToken  string
	LastPublishMs int64
	PublishTimes  []int64 // recent publish timestamps, for the rate window
	BackoffUntil  int64
	Fails         int
}

// newer-event-wins. the caller persists the returned state, published seq, backoff, session token.
func Reconcile(userID string, d Desired, ps PubState, pub Publisher, c Creds, nowMs int64) (PubState, error) {
	// a clear is exempt from backoff, as it is from the throttle, so a stop is never left
	// stuck behind a transient publish failure showing a stale card.
	if d.Kind != "clear" && nowMs < ps.BackoffUntil {
		return ps, nil
	}
	keepaliveDue := d.Kind != "clear" && ps.SessionToken != "" && ps.LastPublishMs != 0 && nowMs-ps.LastPublishMs >= keepaliveMs
	if d.Seq <= ps.PublishedSeq && !keepaliveDue {
		return ps, nil
	}

	// no session means theres no card to take down, call the clear done
	if d.Kind == "clear" && ps.SessionToken == "" {
		ps.PublishedSeq = d.Seq
		return ps, nil
	}

	// throttle publishes to the rate window. clears are exempt so a stop is never stuck
	// behind the throttle. a throttled publish is retried by a later report.
	ps.PublishTimes = recentTimes(ps.PublishTimes, nowMs-rateWindowMs)
	if d.Kind != "clear" && len(ps.PublishTimes) >= rateMax {
		return ps, nil
	}

	var (
		session string
		err     error
	)
	if d.Kind == "clear" {
		err = pub.Clear(userID, ps.SessionToken, c)
	} else {
		session, err = pub.Publish(userID, d, ps.SessionToken, c)
	}
	if err != nil {
		ps.Fails++
		var ra retryAfterErr
		if errors.As(err, &ra) && ra.RetryAfterMs() > 0 {
			ps.BackoffUntil = nowMs + ra.RetryAfterMs()
		} else {
			ps.BackoffUntil = nowMs + backoffMs(ps.Fails)
		}
		return ps, err
	}

	ps.PublishTimes = append(ps.PublishTimes, nowMs)
	if d.Seq > ps.PublishedSeq {
		ps.PublishedSeq = d.Seq
	}
	ps.Fails = 0
	ps.BackoffUntil = 0
	ps.LastPublishMs = nowMs
	if d.Kind == "clear" {
		ps.SessionToken = ""
	} else {
		ps.SessionToken = session
	}
	return ps, nil
}

func recentTimes(times []int64, cutoff int64) []int64 {
	var out []int64
	for _, t := range times {
		if t >= cutoff {
			out = append(out, t)
		}
	}
	return out
}

func backoffMs(fails int) int64 {
	const base, maxMs = 5000, 300000
	d := int64(base) << (fails - 1)
	if d <= 0 || d > maxMs {
		return maxMs
	}
	return d
}

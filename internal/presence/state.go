package presence

type Desired struct {
	Seq  int64
	Kind string // "play" | "pause" | "clear"
	Act  Activity
}

// desired-state for one user; records what presence should be and defers the
// debounced clear to Due. never does i/o.
type UserState struct {
	seq            int64
	debounceMs     int64
	pendingClearAt int64 // 0 = no pending clear
}

func NewUserState(debounceMs int64) *UserState {
	return &UserState{debounceMs: debounceMs}
}

func (s *UserState) OnReport(state string, act Activity, nowMs int64) (Desired, bool) {
	switch state {
	case "playing", "starting":
		s.pendingClearAt = 0
		s.seq++
		return Desired{Seq: s.seq, Kind: "play", Act: act}, true
	case "paused":
		s.pendingClearAt = 0
		act.End = 0 // freeze the bar
		s.seq++
		return Desired{Seq: s.seq, Kind: "pause", Act: act}, true
	case "stopped", "expired":
		// arm a clear; a new play before the deadline cancels it, otherwise Due emits it
		s.pendingClearAt = nowMs + s.debounceMs
		return Desired{}, false
	default:
		return Desired{}, false
	}
}

func (s *UserState) Due(nowMs int64) (Desired, bool) {
	if s.pendingClearAt != 0 && nowMs >= s.pendingClearAt {
		s.pendingClearAt = 0
		s.seq++
		return Desired{Seq: s.seq, Kind: "clear"}, true
	}
	return Desired{}, false
}

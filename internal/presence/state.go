package presence

type Desired struct {
	Seq  int64
	Kind string // "play" | "clear"
	Act  Activity
}

// a same-track tick whose Start moves less than this is drift, not a seek.
const seekToleranceMs = 3000

// desired-state for one user; records what presence should be and defers the debounced
// clear to Due. never does i/o.
type UserState struct {
	seq            int64
	debounceMs     int64
	pendingClearAt int64 // 0 = no pending clear
	lastKind       string
	lastAct        Activity
}

func NewUserState(debounceMs int64) *UserState {
	return &UserState{debounceMs: debounceMs}
}

func (s *UserState) OnReport(state string, act Activity, nowMs int64) (Desired, bool) {
	switch state {
	case "playing", "starting":
		return s.emit("play", act)
	case "paused", "stopped", "expired":
		// hide the card. a music client pauses far more than it stops, so pause is the
		// real "not listening" signal and clears like a stop. arm a clear; a new play
		// before the deadline cancels it, otherwise Due emits it. forget the last activity
		// so a later play re-emits even for the same track.
		s.pendingClearAt = nowMs + s.debounceMs
		s.lastKind, s.lastAct = "", Activity{}
		return Desired{}, false
	default:
		return Desired{}, false
	}
}

func (s *UserState) emit(kind string, act Activity) (Desired, bool) {
	s.pendingClearAt = 0
	if kind == s.lastKind && sameTrack(act, s.lastAct) && abs(act.Start-s.lastAct.Start) <= seekToleranceMs {
		return Desired{Seq: s.seq, Kind: kind, Act: s.lastAct}, false
	}
	s.seq++
	s.lastKind, s.lastAct = kind, act
	return Desired{Seq: s.seq, Kind: kind, Act: act}, true
}

func (s *UserState) Due(nowMs int64) (Desired, bool) {
	if s.pendingClearAt != 0 && nowMs >= s.pendingClearAt {
		s.pendingClearAt = 0
		s.seq++
		return Desired{Seq: s.seq, Kind: "clear"}, true
	}
	return Desired{}, false
}

// the serializable form of a UserState. the plugin gets a fresh wasm instance per call,
// so this round-trips through the kv-store between reports.
type Snapshot struct {
	Seq            int64
	PendingClearAt int64
	LastKind       string
	LastAct        Activity
}

func (s *UserState) Snapshot() Snapshot {
	return Snapshot{s.seq, s.pendingClearAt, s.lastKind, s.lastAct}
}

func RestoreUserState(debounceMs int64, snap Snapshot) *UserState {
	return &UserState{
		seq:            snap.Seq,
		debounceMs:     debounceMs,
		pendingClearAt: snap.PendingClearAt,
		lastKind:       snap.LastKind,
		lastAct:        snap.LastAct,
	}
}

func sameTrack(a, b Activity) bool {
	return a.Name == b.Name && a.Details == b.Details && a.State == b.State && a.LargeImage == b.LargeImage
}

func abs(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}

package presence

import "testing"

func act() Activity {
	return Activity{Type: 2, Name: "ar", Details: "t", State: "al", Start: 100, End: 200}
}

func TestState_PlaySetsDesiredWithEnd(t *testing.T) {
	s := NewUserState(5000)
	d, ok := s.OnReport("playing", act(), 1000)
	if !ok || d.Kind != "play" || d.Act.End != 200 {
		t.Fatalf("play: ok=%v d=%+v", ok, d)
	}
	if d.Seq <= 0 {
		t.Fatalf("seq not set: %d", d.Seq)
	}
}

func TestState_PauseArmsClear(t *testing.T) {
	s := NewUserState(5000)
	s.OnReport("playing", act(), 1000)
	if _, ok := s.OnReport("paused", act(), 2000); ok {
		t.Fatal("pause should not emit inline, it arms a clear like stop")
	}
	d, ok := s.Due(7000)
	if !ok || d.Kind != "clear" {
		t.Fatalf("pause should clear after the debounce: ok=%v d=%+v", ok, d)
	}
}

func TestState_StopArmsButDoesNotClearInline(t *testing.T) {
	s := NewUserState(5000)
	s.OnReport("playing", act(), 1000)
	if _, ok := s.OnReport("stopped", Activity{}, 2000); ok {
		t.Fatal("stop should not emit immediately")
	}
	if _, ok := s.Due(6000); ok {
		t.Fatal("Due before deadline should be false")
	}
	d, ok := s.Due(7000) // 2000 + 5000 = 7000
	if !ok || d.Kind != "clear" {
		t.Fatalf("Due after deadline should clear: ok=%v d=%+v", ok, d)
	}
}

func TestState_NewPlayCancelsPendingClear(t *testing.T) {
	s := NewUserState(5000)
	s.OnReport("playing", act(), 1000)
	s.OnReport("stopped", Activity{}, 2000)
	if _, ok := s.OnReport("playing", act(), 3000); !ok {
		t.Fatal("new play should emit")
	}
	if _, ok := s.Due(9000); ok {
		t.Fatal("pending clear should have been cancelled by the new play")
	}
}

func TestState_SeqStrictlyIncreases(t *testing.T) {
	s := NewUserState(5000)
	a := act()
	d1, _ := s.OnReport("playing", a, 1000)
	a.Name = "second"
	d2, _ := s.OnReport("playing", a, 2000)
	a.Name = "third"
	d3, _ := s.OnReport("playing", a, 3000)
	if !(d1.Seq < d2.Seq && d2.Seq < d3.Seq) {
		t.Fatalf("seq not strictly increasing: %d %d %d", d1.Seq, d2.Seq, d3.Seq)
	}
}

func TestState_UnchangedTickNoOps(t *testing.T) {
	s := NewUserState(5000)
	d1, ok1 := s.OnReport("playing", act(), 1000)
	d2, ok2 := s.OnReport("playing", act(), 6000)
	if !ok1 || ok2 {
		t.Fatalf("first emits, an identical tick is a no-op: ok1=%v ok2=%v", ok1, ok2)
	}
	if d2.Seq != d1.Seq {
		t.Fatalf("a no-op tick must not advance seq: %d -> %d", d1.Seq, d2.Seq)
	}
}

func TestState_ClearSurvivesSnapshot(t *testing.T) {
	s := NewUserState(0)
	s.OnReport("playing", act(), 1000)
	s.OnReport("stopped", Activity{}, 2000)
	d, _ := s.Due(2000)
	snap := s.Snapshot()
	if snap.LastKind != "clear" || snap.Seq != d.Seq {
		t.Fatalf("the snapshot should carry the clear as desired state: %+v", snap)
	}
}

func TestState_RepeatStopNoOps(t *testing.T) {
	s := NewUserState(0)
	s.OnReport("playing", act(), 1000)
	s.OnReport("stopped", Activity{}, 2000)
	s.Due(2000)
	s.OnReport("stopped", Activity{}, 3000)
	if _, ok := s.Due(9000); ok {
		t.Fatal("a stop after the clear emitted should not arm another")
	}
}

func TestState_PlayAfterClearReemitsSameTrack(t *testing.T) {
	s := NewUserState(0)
	s.OnReport("playing", act(), 1000)
	s.OnReport("paused", act(), 2000)
	s.Due(2000)
	if _, ok := s.OnReport("playing", act(), 3000); !ok {
		t.Fatal("resuming the same track after a clear should re-emit")
	}
}

func TestState_SeekReemits(t *testing.T) {
	s := NewUserState(5000)
	a := act()
	d1, _ := s.OnReport("playing", a, 1000)
	a.Start += 30000 // position jumped
	d2, ok := s.OnReport("playing", a, 2000)
	if !ok || d2.Seq <= d1.Seq {
		t.Fatalf("a seek should re-emit with a new seq: ok=%v %d -> %d", ok, d1.Seq, d2.Seq)
	}
}

func playing(t *testing.T) Snapshot {
	t.Helper()
	us := NewUserState(0)
	us.OnReport("playing", Activity{Name: "Saosin"}, 1000)
	return us.Snapshot()
}

func TestArmClear_ArmsFromPlay(t *testing.T) {
	snap := playing(t)
	next, armed := ArmClear(0, snap, snap.Seq, 2000)
	if !armed || next.LastKind != "clear" || next.Seq != snap.Seq+1 {
		t.Fatalf("play arms a clear: armed=%v %+v", armed, next)
	}
}

func TestArmClear_OutranksAPublishedSeq(t *testing.T) {
	snap := playing(t)
	next, armed := ArmClear(0, snap, snap.Seq+4, 2000)
	if !armed || next.Seq <= snap.Seq+4 {
		t.Fatalf("clear must outrank what was published: armed=%v seq=%d published=%d", armed, next.Seq, snap.Seq+4)
	}
}

func TestArmClear_NothingToArm(t *testing.T) {
	if _, armed := ArmClear(0, Snapshot{}, 0, 2000); armed {
		t.Fatal("empty snapshot arms nothing")
	}
	armedOnce, _ := ArmClear(0, playing(t), 0, 2000)
	if _, again := ArmClear(0, armedOnce, 0, 3000); again {
		t.Fatalf("re-arming an armed clear: %+v", armedOnce)
	}
}

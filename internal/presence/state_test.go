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

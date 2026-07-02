package presence

import "testing"

func act() Activity { return Activity{Type: 2, Name: "ar", Details: "t", State: "al", Start: 100, End: 200} }

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

func TestState_PauseFreezesBar(t *testing.T) {
	s := NewUserState(5000)
	s.OnReport("playing", act(), 1000)
	d, ok := s.OnReport("paused", act(), 2000)
	if !ok || d.Kind != "pause" || d.Act.End != 0 {
		t.Fatalf("pause should zero End: ok=%v d=%+v", ok, d)
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
	d1, _ := s.OnReport("playing", act(), 1000)
	d2, _ := s.OnReport("paused", act(), 2000)
	d3, _ := s.OnReport("playing", act(), 3000)
	if !(d1.Seq < d2.Seq && d2.Seq < d3.Seq) {
		t.Fatalf("seq not strictly increasing: %d %d %d", d1.Seq, d2.Seq, d3.Seq)
	}
}

package monitor

import (
	"testing"

	"pgregory.net/rapid"
)

func TestNextExamples(t *testing.T) {
	th := Thresholds{Failure: 3, Recovery: 2}
	run := func(results ...bool) (State, []Transition) {
		s := State{Health: Unknown}
		var got []Transition
		for _, ok := range results {
			var tr Transition
			s, tr = Next(s, ok, th)
			got = append(got, tr)
		}
		return s, got
	}

	if s, _ := run(true); s.Health != Up {
		t.Fatalf("one success from unknown should be up, got %v", s.Health)
	}
	if s, tr := run(false, false); s.Health != Unknown || tr[1] != NoChange {
		t.Fatalf("two failures (below the threshold) change nothing, got %v %v", s.Health, tr)
	}
	if s, tr := run(true, false, false, false); s.Health != Down || tr[3] != Opened {
		t.Fatalf("three failures in a row open an incident, got %v %v", s.Health, tr)
	}
	if _, tr := run(true, false, false, true, false, false); tr[5] != NoChange {
		t.Fatalf("a success resets the failure streak, got %v", tr)
	}
	if s, tr := run(false, false, false, true, true); s.Health != Up || tr[4] != Resolved {
		t.Fatalf("two successes resolve it, got %v %v", s.Health, tr)
	}
	if _, tr := run(false, false, false, true, false, true); tr[3] != NoChange || tr[5] != NoChange {
		t.Fatalf("a flapping target stays down without resolving, got %v", tr)
	}
}

// Properties that must hold for any sequence of results and any thresholds.
func TestNextProperties(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		th := Thresholds{Failure: rapid.IntRange(1, 6).Draw(t, "failure"), Recovery: rapid.IntRange(1, 6).Draw(t, "recovery")}
		results := rapid.SliceOfN(rapid.Bool(), 0, 60).Draw(t, "results")

		s := State{Health: Unknown}
		open := false
		for i, ok := range results {
			prev := s
			var tr Transition
			s, tr = Next(s, ok, th)

			switch tr {
			case Opened:
				if open {
					t.Fatalf("step %d: opened an incident while one was already open", i)
				}
				if ok || prev.Health == Down || s.Failures < th.Failure {
					t.Fatalf("step %d: opened without %d consecutive failures from a non-down state", i, th.Failure)
				}
				open = true
			case Resolved:
				if !open {
					t.Fatalf("step %d: resolved with nothing open", i)
				}
				if !ok || s.Successes < th.Recovery {
					t.Fatalf("step %d: resolved without %d consecutive successes", i, th.Recovery)
				}
				open = false
			}
			if (s.Health == Down) != open {
				t.Fatalf("step %d: health %v disagrees with an incident being open (%v)", i, s.Health, open)
			}
			if s.Failures > 0 && s.Successes > 0 {
				t.Fatalf("step %d: both streaks running at once", i)
			}
			if ok && s.Failures != 0 || !ok && s.Successes != 0 {
				t.Fatalf("step %d: the other streak wasn't reset", i)
			}
		}
	})
}

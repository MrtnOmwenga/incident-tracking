// Package monitor runs checks against projects and turns their results into incidents.
package monitor

// Health is what Lighthouse currently believes about a monitored target.
type Health string

const (
	Unknown Health = "unknown"
	Up      Health = "up"
	Down    Health = "down"
)

// State is a monitor's health and the streak of results behind it.
type State struct {
	Health    Health
	Failures  int // consecutive failed checks
	Successes int // consecutive successful checks
}

// Thresholds add hysteresis: a target must fail `Failure` times in a row to be declared down, and
// succeed `Recovery` times in a row to be declared up again, so one blip doesn't open an incident
// and a flapping target doesn't open and close one every minute.
type Thresholds struct {
	Failure  int
	Recovery int
}

type Transition int

const (
	NoChange Transition = iota
	Opened              // the target just went down: open an incident
	Resolved            // the target just recovered: resolve the incident
)

// Next folds one check result into the state. It is pure: all the rules about when incidents open
// and close live here, and are tested exhaustively.
func Next(s State, ok bool, t Thresholds) (State, Transition) {
	if ok {
		s.Failures = 0
		s.Successes++
		switch {
		case s.Health == Down && s.Successes >= t.Recovery:
			s.Health = Up
			return s, Resolved
		case s.Health == Unknown:
			s.Health = Up
		}
		return s, NoChange
	}
	s.Successes = 0
	s.Failures++
	if s.Health != Down && s.Failures >= t.Failure {
		s.Health = Down
		return s, Opened
	}
	return s, NoChange
}

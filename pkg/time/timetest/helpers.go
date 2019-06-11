package timetest

import (
	sonotime "github.com/heptio/sonobuoy/pkg/time"
	"time"
)

var (
	// shortDuration is used in tests when we want to sleep, but not for long
	// for the sake of testing time.
	shortDuration = 250 * time.Millisecond
)

// UseShortAfter updates the After method to expedite sleep times for tests. Callers
// should call ResetAfter() when they are done with their test.
func UseShortAfter() {
	sonotime.After = func(time.Duration) <-chan time.Time { return time.After(shortDuration) }
}

// UseNoAfter updates the After method to be a noop for tests. Callers
// should call ResetAfter() when they are done with their test.
func UseNoAfter() {
	sonotime.After = func(time.Duration) <-chan time.Time { return time.After(0) }
}

// ResetAfter is just a test helper to simplify setting the After variable
// for a specific test and then running this cleanup method when done to
// return it to its normal state.
func ResetAfter() {
	sonotime.After = time.After
}

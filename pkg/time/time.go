package time

import (
	"time"
)

// After is a function variable for swapping during tests, allowing
// variable behavior, tracking of calls, etc depending on what the test
// needs.
var After = time.After

// Time is a function that helps convert test time values
// to pointers for the json STATUS
func Time(time time.Time) *time.Time {
	return &time
}

// Duration is a function that helps convert test duration values
// to pointers for the json STATUS
func Duration(duration time.Duration) *time.Duration {
	return &duration
}

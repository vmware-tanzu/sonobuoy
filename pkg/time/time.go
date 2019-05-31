package time

import (
	"time"
)

// After is a function variable for swapping during tests, allowing
// variable behavior, tracking of calls, etc depending on what the test
// needs.
var After = time.After

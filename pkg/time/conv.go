package time

import "time"

func SecsToDuration(secs int) time.Duration {
	return time.Duration(float64(secs) * float64(time.Second))
}


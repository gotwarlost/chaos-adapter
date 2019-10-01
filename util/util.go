package util

import (
	"math/rand"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// NextDelay calculates the delay to use from a normal distribution
// with the supplied duration as mean and std deviation based on the mean
func NextDelay(d time.Duration) time.Duration {
	var mean float64 = float64(d)
	var stddev = mean / 5.0
	n := rand.NormFloat64()*stddev + mean
	if n < 0 {
		n = 0
	}
	max := float64(d) * 2
	if n > max {
		n = max
	}
	return time.Duration(int64(n))
}

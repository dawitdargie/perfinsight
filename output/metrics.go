package output

// humanMetrics holds computed human-readable values for display.
// Filled by computeMetrics on Day 23.
type humanMetrics struct {
	multiplier   float64
	percentage   float64
	delta        int64
	isRegression bool
}

// computeMetrics derives display values from baseline and current.
// Implemented Day 23.
func computeMetrics(baselineMs, currentMs float64) humanMetrics {
	return humanMetrics{}
}
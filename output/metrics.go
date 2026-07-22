// output/metrics.go
package output

import "math"

// humanMetrics holds computed human-readable values for display.
type humanMetrics struct {
	multiplier   float64
	percentage   float64
	delta        int64
	isRegression bool
}

// computeMetrics derives display values from baseline and current.
func computeMetrics(baselineMs, currentMs float64) humanMetrics {
	if baselineMs == 0 {
		return humanMetrics{}
	}
	delta := currentMs - baselineMs
	ratio := currentMs / baselineMs
	percentage := ((currentMs - baselineMs) / baselineMs) * 100

	// Round to one decimal for multiplier
	multiplier := math.Round(ratio*10) / 10
	// Round to nearest whole number for percentage
	percentage = math.Round(percentage)

	return humanMetrics{
		multiplier:   multiplier,
		percentage:   percentage,
		delta:        int64(math.Round(delta)),
		isRegression: currentMs > baselineMs,
	}
}
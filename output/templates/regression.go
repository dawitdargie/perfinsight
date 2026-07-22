//templates/regression.go
package templates

// PerformanceRegression returns fix suggestions for regression issues.
func PerformanceRegression() []string {
	return []string{
		"Review recent code changes or deployments",
		"Check for new or modified database queries",
		"Look for changes in external API call patterns",
		"Profile the endpoint to identify the source of slowdown",
	}
}

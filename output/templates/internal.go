//templates/error_rate.go
package templates

func HighInternalProcessing() []string {
	return []string{
		"Profile CPU usage during request processing",
		"Review serialization/deserialization logic for efficiency",
		"Check for inefficient loops or data transformations",
		"Consider using worker pools for CPU-intensive operations",
	}
}
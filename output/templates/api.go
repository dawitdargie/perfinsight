//templates/api.go
package templates

// ExternalAPIBottleneck returns fix suggestions for external API bottleneck issues.
func ExternalAPIBottleneck() []string {
	return []string{
		"Add timeout limits to external API calls",
		"Cache external API responses where appropriate",
		"Consider moving external API calls to background jobs",
		"Implement circuit breaker pattern for resilience",
	}
}
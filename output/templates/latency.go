package templates

func HighLatency() []string {
	return []string{
		"Consider adding caching for frequently accessed data",
		"Evaluate if synchronous processing can be moved to async/background jobs",
		"Review endpoint logic for unnecessary computation or blocking calls",
		"Consider pagination or streaming for large payloads",
	}
}
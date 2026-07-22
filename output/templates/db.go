//templates/api.go
package templates

// DBBottleneck returns fix suggestions for database bottleneck issues.
func DBBottleneck() []string {
	return []string{
		"Add indexes on frequently queried columns",
		"Reduce SELECT * — select only required columns",
		"Consider caching repeated read queries",
		"Profile slow queries using EXPLAIN ANALYZE",
	}
}
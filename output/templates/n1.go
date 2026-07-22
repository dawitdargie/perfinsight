//templates//n1.go
package templates

// N1Query returns fix suggestions for N+1 query issues.
func N1Query() []string {
	return []string{
		"Use batch loading: replace looped queries with IN clause",
		"Use JOIN to fetch related data in one query",
		"Implement eager loading for related records",
		"Cache query results if data changes infrequently",
	}
}
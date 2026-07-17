package templates

func HighErrorRate() []string {
	return []string{
		"Check for recent code changes or deployments that may have introduced bugs",
		"Review application logs for stack traces and error messages",
		"Verify upstream dependencies are healthy",
		"Add more granular error tracking to identify specific failure modes",
	}
}
// templates/templates_test.go
package templates

import "testing"

func TestDBBottleneck_ReturnsNonEmptySlice(t *testing.T) {
	result := DBBottleneck()
	if len(result) == 0 {
		t.Error("Expected non-empty suggestions")
	}
}

func TestN1Query_ReturnsNonEmptySlice(t *testing.T) {
	result := N1Query()
	if len(result) == 0 {
		t.Error("Expected non-empty suggestions")
	}
}

func TestExternalAPIBottleneck_ReturnsNonEmptySlice(t *testing.T) {
	result := ExternalAPIBottleneck()
	if len(result) == 0 {
		t.Error("Expected non-empty suggestions")
	}
}

func TestPerformanceRegression_ReturnsNonEmptySlice(t *testing.T) {
	result := PerformanceRegression()
	if len(result) == 0 {
		t.Error("Expected non-empty suggestions")
	}
}

func TestDBBottleneck_ContainsIndexSuggestion(t *testing.T) {
	result := DBBottleneck()
	found := false
	for _, s := range result {
		if len(s) > 0 && s[0] == 'A' {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected index suggestion in DB bottleneck template")
	}
}

func TestN1Query_ContainsBatchSuggestion(t *testing.T) {
	result := N1Query()
	found := false
	for _, s := range result {
		for _, word := range []string{"batch", "IN", "JOIN"} {
			if containsWord(s, word) {
				found = true
				break
			}
		}
	}
	if !found {
		t.Error("Expected batch/IN/JOIN suggestion in N+1 template")
	}
}

func containsWord(s, word string) bool {
	for i := 0; i <= len(s)-len(word); i++ {
		if s[i:i+len(word)] == word {
			return true
		}
	}
	return false
}
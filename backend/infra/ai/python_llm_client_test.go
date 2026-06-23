package ai

import (
	"testing"
	"time"
)

func TestLLMRequestTimeoutsAllowPythonUpstreamHeadroom(t *testing.T) {
	t.Parallel()

	tests := map[string]time.Duration{
		"intent":          intentRequestTimeout,
		"answer":          answerRequestTimeout,
		"anaphora":        anaphoraRequestTimeout,
		"semantic_search": semanticRankRequestTimeout,
	}

	for name, got := range tests {
		if got != 8*time.Second {
			t.Fatalf("%s timeout = %s, want 8s", name, got)
		}
	}
}

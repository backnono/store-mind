package ai

import (
	"testing"
	"time"
)

func TestIntentRequestTimeoutAllowsPythonUpstreamHeadroom(t *testing.T) {
	t.Parallel()

	if intentRequestTimeout != 8*time.Second {
		t.Fatalf("intentRequestTimeout = %s, want 8s", intentRequestTimeout)
	}
}

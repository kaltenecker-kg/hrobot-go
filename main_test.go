package hrobot

import (
	"os"
	"testing"
	"time"
)

// TestMain shortens the retry backoff and the polling delay for the whole
// test binary. The tests assert on what is retried and how often, not on how
// long the client waits in between, so real 500ms/2s sleeps only cost time.
func TestMain(m *testing.M) {
	retryBackoff = time.Millisecond
	waitMinDelay = time.Millisecond
	os.Exit(m.Run())
}

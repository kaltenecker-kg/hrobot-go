package hrobot

import (
	"context"
	"fmt"
	"time"
)

// waitMinDelay is the first delay waitForCondition sleeps between polls. It
// is a variable only so the test binary can shorten it (see TestMain).
var waitMinDelay = 2 * time.Second

// waitForCondition polls a condition function with exponential backoff until it returns true or context times out.
func waitForCondition(ctx context.Context, condition func() (bool, error)) error {
	const (
		maxDelay   = 30 * time.Second
		maxRetries = 30 // max ~15 minutes with exponential backoff
	)

	delay := waitMinDelay
	for i := 0; i < maxRetries; i++ {
		// Check if context is cancelled
		select {
		case <-ctx.Done():
			return fmt.Errorf("waiting cancelled: %w", ctx.Err())
		default:
		}

		// Check condition
		ready, err := condition()
		if err != nil {
			return fmt.Errorf("error checking condition: %w", err)
		}
		if ready {
			return nil
		}

		// Wait before retry
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return fmt.Errorf("waiting cancelled: %w", ctx.Err())
		case <-timer.C:
		}

		// Exponential backoff with max delay
		delay *= 2
		if delay > maxDelay {
			delay = maxDelay
		}
	}

	return fmt.Errorf("timeout waiting for condition after %d retries", maxRetries)
}

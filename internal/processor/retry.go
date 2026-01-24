package processor

import (
	"context"
	"fmt"
	"time"
)

// RetryPolicy defines how we should retry failed operations
type RetryPolicy struct {
	MaxRetries     int
	InitialBackoff time.Duration
	MaxBackoff     time.Duration
	Multiplier     float64
}

// DefaultRetryPolicy provides sensible defaults
var DefaultRetryPolicy = RetryPolicy{
	MaxRetries:     3,
	InitialBackoff: 100 * time.Millisecond,
	MaxBackoff:     1 * time.Second,
	Multiplier:     2.0,
}

// DO executes a function with retry logic based on the policy
// It is useful for wrapping specific DB calls or external API requests
func Do(ctx context.Context, policy RetryPolicy, fn func() error) error {
	var err error
	backoff := policy.InitialBackoff

	for attempt := 1; attempt <= policy.MaxRetries; attempt++ {
		// Check if context is cancelled
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Attempt the operation
		if err = fn(); err == nil {
			return nil // Success
		}

		// If it failed, wait before retrying (unless its last attempt)
		if attempt < policy.MaxRetries {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
				// Calculate next backoff duration
				nextBackoff := float64(backoff) * policy.Multiplier
				if nextBackoff > float64(policy.MaxBackoff) {
					backoff = policy.MaxBackoff
				} else {
					backoff = time.Duration(nextBackoff)
				}
			}

		}
	}

	return fmt.Errorf("operation failed after %d attempts: %v", policy.MaxRetries, err)

}

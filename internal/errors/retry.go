package errors

import (
	"fmt"
	"math"
	"math/rand"
	"os"
	"time"
)

// RetryConfig holds configuration for retry logic
type RetryConfig struct {
	MaxAttempts  int
	BaseDelay    time.Duration
	MaxDelay     time.Duration
	Jitter       bool
	OnRetry      func(attempt int, err error, delay time.Duration)
}

// DefaultRetryConfig returns the default retry configuration
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxAttempts: 5,
		BaseDelay:   1 * time.Second,
		MaxDelay:    16 * time.Second,
		Jitter:      true,
		OnRetry: func(attempt int, err error, delay time.Duration) {
			fmt.Fprintf(os.Stderr, "[Retry] Attempt %d failed: %v. Retrying in %v...\n", 
				attempt, err, delay.Round(time.Millisecond))
		},
	}
}

// RetryFunc is a function that can be retried
type RetryFunc func() error

// Retry executes a function with exponential backoff retry logic
func Retry(fn RetryFunc, config *RetryConfig) error {
	if config == nil {
		config = DefaultRetryConfig()
	}

	var lastErr error

	for attempt := 1; attempt <= config.MaxAttempts; attempt++ {
		// Execute the function
		err := fn()
		
		// Success!
		if err == nil {
			if attempt > 1 {
				fmt.Fprintf(os.Stderr, "[Retry] Succeeded on attempt %d\n", attempt)
			}
			return nil
		}

		lastErr = err

		// Check if error is retryable
		if !IsRetryableError(err) {
			fmt.Fprintf(os.Stderr, "[Retry] Non-retryable error: %v\n", err)
			return err
		}

		// Last attempt, don't delay
		if attempt == config.MaxAttempts {
			fmt.Fprintf(os.Stderr, "[Retry] All %d attempts exhausted\n", config.MaxAttempts)
			return err
		}

		// Calculate delay
		delay := calculateDelay(attempt, config, err)

		// Notify before retry
		if config.OnRetry != nil {
			config.OnRetry(attempt, err, delay)
		}

		// Wait before retry
		time.Sleep(delay)
	}

	return lastErr
}

// RetryWithContext executes a function with retry logic and context for cancellation
// Returns (success bool, error)
func RetryWithContext(fn RetryFunc, config *RetryConfig, cancel <-chan bool) error {
	if config == nil {
		config = DefaultRetryConfig()
	}

	var lastErr error

	for attempt := 1; attempt <= config.MaxAttempts; attempt++ {
		// Check for cancellation
		select {
		case <-cancel:
			fmt.Fprintf(os.Stderr, "[Retry] Operation cancelled by user\n")
			return fmt.Errorf("operation cancelled")
		default:
		}

		// Execute the function
		err := fn()
		
		if err == nil {
			if attempt > 1 {
				fmt.Fprintf(os.Stderr, "[Retry] Succeeded on attempt %d\n", attempt)
			}
			return nil
		}

		lastErr = err

		if !IsRetryableError(err) {
			return err
		}

		if attempt == config.MaxAttempts {
			return err
		}

		delay := calculateDelay(attempt, config, err)

		if config.OnRetry != nil {
			config.OnRetry(attempt, err, delay)
		}

		// Wait with cancellation support
		select {
		case <-time.After(delay):
			// Continue to next retry
		case <-cancel:
			fmt.Fprintf(os.Stderr, "[Retry] Operation cancelled during wait\n")
			return fmt.Errorf("operation cancelled")
		}
	}

	return lastErr
}

// calculateDelay computes the delay before next retry using exponential backoff
func calculateDelay(attempt int, config *RetryConfig, err error) time.Duration {
	// Check if error specifies a retry-after duration
	if retryAfter := GetRetryAfter(err); retryAfter > 0 {
		return retryAfter
	}

	// Exponential backoff: baseDelay * 2^(attempt-1)
	delay := config.BaseDelay * time.Duration(math.Pow(2, float64(attempt-1)))

	// Cap at max delay
	if delay > config.MaxDelay {
		delay = config.MaxDelay
	}

	// Add jitter to avoid thundering herd
	if config.Jitter {
		jitter := time.Duration(rand.Int63n(int64(delay) / 10)) // 0-10% jitter
		delay = delay + jitter
	}

	return delay
}

// RetryStatus represents the current status of a retry operation
type RetryStatus struct {
	Attempt      int
	MaxAttempts  int
	LastError    error
	NextRetryIn  time.Duration
	IsRetrying   bool
}

func (s RetryStatus) Message() string {
	if !s.IsRetrying {
		return ""
	}

	if s.LastError != nil {
		if apiErr, ok := s.LastError.(*APIError); ok {
			return fmt.Sprintf("Retrying... (attempt %d/%d) - %s", 
				s.Attempt, s.MaxAttempts, apiErr.GetUserFriendlyMessage())
		}
	}

	return fmt.Sprintf("Retrying... (attempt %d/%d)", s.Attempt, s.MaxAttempts)
}

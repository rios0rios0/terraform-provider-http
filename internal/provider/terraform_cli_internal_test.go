package provider

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// withRetries calls resolve until it succeeds, at most `attempts` times, waiting firstDelay before
// the second attempt and twice as long before each one after it. When no attempt succeeds it
// returns every attempt's error, so the log shows whether the same lookup kept failing or each
// one failed differently.
//
// `sleep` is a parameter so the policy can be tested without waiting for it.
func withRetries(
	ctx context.Context,
	attempts int,
	firstDelay time.Duration,
	sleep func(time.Duration),
	resolve func(context.Context) (string, error),
) (string, error) {
	failures := make([]error, 0, attempts)
	delay := firstDelay

	for attempt := 1; attempt <= attempts; attempt++ {
		result, err := resolve(ctx)
		if err == nil {
			return result, nil
		}

		failures = append(failures, fmt.Errorf("attempt %d of %d: %w", attempt, attempts, err))
		if attempt < attempts {
			sleep(delay)
			delay *= 2
		}
	}

	return "", errors.Join(failures...)
}

// failingResolver fails its first `failures` calls with the error a HashiCorp lookup outage
// produces, then resolves a Terraform CLI.
func failingResolver(failures int, calls *int) func(context.Context) (string, error) {
	return func(context.Context) (string, error) {
		*calls++
		if *calls <= failures {
			return "", errors.New("Unknown status: 502")
		}

		return "/opt/terraform/terraform", nil
	}
}

func TestWithRetries(t *testing.T) {
	t.Parallel()

	t.Run("should return the result as soon as an attempt succeeds", func(t *testing.T) {
		t.Parallel()

		// given
		calls := 0
		var slept []time.Duration
		sleep := func(delay time.Duration) { slept = append(slept, delay) }

		// when
		result, err := withRetries(context.Background(), 5, 2*time.Second, sleep, failingResolver(2, &calls))

		// then
		require.NoError(t, err)
		assert.Equal(t, "/opt/terraform/terraform", result)
		assert.Equal(t, 3, calls)
		assert.Equal(t, []time.Duration{2 * time.Second, 4 * time.Second}, slept,
			"the wait doubles after every failed attempt")
	})

	t.Run("should not wait when the first attempt succeeds", func(t *testing.T) {
		t.Parallel()

		// given
		calls := 0
		var slept []time.Duration
		sleep := func(delay time.Duration) { slept = append(slept, delay) }

		// when
		result, err := withRetries(context.Background(), 5, 2*time.Second, sleep, failingResolver(0, &calls))

		// then
		require.NoError(t, err)
		assert.Equal(t, "/opt/terraform/terraform", result)
		assert.Equal(t, 1, calls)
		assert.Empty(t, slept)
	})

	t.Run("should report every attempt when none succeeds", func(t *testing.T) {
		t.Parallel()

		// given
		calls := 0
		var slept []time.Duration
		sleep := func(delay time.Duration) { slept = append(slept, delay) }

		// when
		result, err := withRetries(context.Background(), 3, time.Second, sleep, failingResolver(3, &calls))

		// then
		require.Error(t, err)
		assert.Empty(t, result)
		assert.Equal(t, 3, calls)
		require.ErrorContains(t, err, "attempt 1 of 3: Unknown status: 502")
		require.ErrorContains(t, err, "attempt 3 of 3: Unknown status: 502")
		assert.Equal(t, []time.Duration{time.Second, 2 * time.Second}, slept,
			"there is nothing to wait for after the last attempt")
	})
}

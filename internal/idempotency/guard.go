package idempotency

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// Executor protects one operation with an idempotency reservation.
type Executor interface {
	Run(context.Context, string, func() error) (bool, error)
}

// Guard executes an operation only for the first delivery of an event to a consumer.
type Guard struct {
	repository Repository
	consumer   string
}

// NewGuard creates an idempotency guard for one logical consumer.
func NewGuard(repository Repository, consumer string) (Guard, error) {
	if repository == nil {
		return Guard{}, errors.New("idempotency repository is required")
	}
	if strings.TrimSpace(consumer) == "" {
		return Guard{}, errors.New("consumer name is required")
	}
	return Guard{repository: repository, consumer: consumer}, nil
}

// Run reserves the event and executes operation. A false executed result is a duplicate.
func (guard Guard) Run(ctx context.Context, eventID string, operation func() error) (executed bool, err error) {
	claimed, err := guard.repository.Claim(ctx, guard.consumer, eventID)
	if err != nil {
		return false, fmt.Errorf("reserve event %q: %w", eventID, err)
	}
	if !claimed {
		return false, nil
	}

	if err := operation(); err != nil {
		if releaseErr := guard.repository.Release(ctx, guard.consumer, eventID); releaseErr != nil {
			return true, errors.Join(err, fmt.Errorf("release event %q after failure: %w", eventID, releaseErr))
		}
		return true, err
	}
	return true, nil
}

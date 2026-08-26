package idempotency

import (
	"context"
	"errors"
	"testing"
)

type fakeRepository struct {
	claimed     bool
	claimErr    error
	releaseErr  error
	claims      []string
	releases    []string
	consumerIDs []string
}

type scopedMemoryRepository struct {
	claims map[string]bool
}

func (repository *scopedMemoryRepository) Claim(_ context.Context, consumer, eventID string) (bool, error) {
	key := consumer + "#" + eventID
	if repository.claims[key] {
		return false, nil
	}
	repository.claims[key] = true
	return true, nil
}

func (repository *scopedMemoryRepository) Release(_ context.Context, consumer, eventID string) error {
	delete(repository.claims, consumer+"#"+eventID)
	return nil
}

func (repository *fakeRepository) Claim(_ context.Context, consumer, eventID string) (bool, error) {
	repository.consumerIDs = append(repository.consumerIDs, consumer)
	repository.claims = append(repository.claims, eventID)
	return repository.claimed, repository.claimErr
}

func (repository *fakeRepository) Release(_ context.Context, consumer, eventID string) error {
	repository.consumerIDs = append(repository.consumerIDs, consumer)
	repository.releases = append(repository.releases, eventID)
	return repository.releaseErr
}

func TestGuardRunExecutesClaimedEvent(t *testing.T) {
	t.Parallel()

	repository := &fakeRepository{claimed: true}
	guard, err := NewGuard(repository, "process-stock")
	if err != nil {
		t.Fatalf("NewGuard() error = %v", err)
	}
	operationCalls := 0

	executed, err := guard.Run(context.Background(), "event-123", func() error {
		operationCalls++
		return nil
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !executed || operationCalls != 1 {
		t.Fatalf("Run() executed = %v, operation calls = %d", executed, operationCalls)
	}
	if len(repository.releases) != 0 {
		t.Fatalf("releases = %v, want none", repository.releases)
	}
}

func TestGuardRunSkipsDuplicateEvent(t *testing.T) {
	t.Parallel()

	repository := &fakeRepository{claimed: false}
	guard, err := NewGuard(repository, "process-stock")
	if err != nil {
		t.Fatalf("NewGuard() error = %v", err)
	}
	operationCalls := 0

	executed, err := guard.Run(context.Background(), "event-123", func() error {
		operationCalls++
		return nil
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if executed || operationCalls != 0 {
		t.Fatalf("Run() executed = %v, operation calls = %d", executed, operationCalls)
	}
}

func TestGuardScopesSameEventIDByConsumer(t *testing.T) {
	t.Parallel()

	repository := &scopedMemoryRepository{claims: make(map[string]bool)}
	stockGuard, err := NewGuard(repository, "process-stock")
	if err != nil {
		t.Fatalf("NewGuard(stock) error = %v", err)
	}
	notificationGuard, err := NewGuard(repository, "send-notification")
	if err != nil {
		t.Fatalf("NewGuard(notification) error = %v", err)
	}
	operationCalls := 0
	operation := func() error {
		operationCalls++
		return nil
	}

	for _, guard := range []Guard{stockGuard, notificationGuard, stockGuard, notificationGuard} {
		if _, err := guard.Run(context.Background(), "event-123", operation); err != nil {
			t.Fatalf("Run() error = %v", err)
		}
	}
	if operationCalls != 2 {
		t.Fatalf("operation calls = %d, want one per consumer", operationCalls)
	}
}

func TestGuardRunReleasesFailedEvent(t *testing.T) {
	t.Parallel()

	repository := &fakeRepository{claimed: true}
	guard, err := NewGuard(repository, "send-notification")
	if err != nil {
		t.Fatalf("NewGuard() error = %v", err)
	}

	executed, err := guard.Run(context.Background(), "event-123", func() error {
		return errors.New("provider unavailable")
	})
	if !executed || err == nil || err.Error() != "provider unavailable" {
		t.Fatalf("Run() executed = %v, error = %v", executed, err)
	}
	if len(repository.releases) != 1 || repository.releases[0] != "event-123" {
		t.Fatalf("releases = %v", repository.releases)
	}
}

func TestGuardRunReturnsRepositoryErrors(t *testing.T) {
	t.Parallel()

	t.Run("claim", func(t *testing.T) {
		t.Parallel()

		repository := &fakeRepository{claimErr: errors.New("DynamoDB unavailable")}
		guard, err := NewGuard(repository, "process-stock")
		if err != nil {
			t.Fatalf("NewGuard() error = %v", err)
		}

		_, err = guard.Run(context.Background(), "event-123", func() error { return nil })
		if err == nil || err.Error() != `reserve event "event-123": DynamoDB unavailable` {
			t.Fatalf("Run() error = %v", err)
		}
	})

	t.Run("release", func(t *testing.T) {
		t.Parallel()

		repository := &fakeRepository{claimed: true, releaseErr: errors.New("DynamoDB unavailable")}
		guard, err := NewGuard(repository, "process-stock")
		if err != nil {
			t.Fatalf("NewGuard() error = %v", err)
		}

		_, err = guard.Run(context.Background(), "event-123", func() error { return errors.New("processing failed") })
		if err == nil || err.Error() != "processing failed\nrelease event \"event-123\" after failure: DynamoDB unavailable" {
			t.Fatalf("Run() error = %v", err)
		}
	})
}

func TestNewGuardValidatesConfiguration(t *testing.T) {
	t.Parallel()

	if _, err := NewGuard(nil, "process-stock"); err == nil || err.Error() != "idempotency repository is required" {
		t.Fatalf("NewGuard(nil) error = %v", err)
	}
	if _, err := NewGuard(&fakeRepository{}, " "); err == nil || err.Error() != "consumer name is required" {
		t.Fatalf("NewGuard(blank consumer) error = %v", err)
	}
}

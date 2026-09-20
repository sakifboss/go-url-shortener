package worker

import (
	"context"
	"errors"
	"testing"
	"time"

	"goshort/model"
)

type fakeClickRepository struct {
	events []model.ClickEvent
}

func (f *fakeClickRepository) Create(
	_ context.Context,
	event model.ClickEvent,
) error {
	f.events = append(f.events, event)
	return nil
}

func TestClickWorkerPoolEnqueue(t *testing.T) {
	repo := &fakeClickRepository{}

	pool := NewClickWorkerPool(
		repo,
		1,
		10,
		3,
		10*time.Millisecond,
	)

	pool.Start()

	event := model.ClickEvent{
		EventKey: "test-event-1",
		URLID:    1,
	}

	if err := pool.Enqueue(context.Background(), event); err != nil {
		t.Fatalf("enqueue failed: %v", err)
	}

	pool.Shutdown()

	if len(repo.events) != 1 {
		t.Fatalf(
			"expected 1 event, got %d",
			len(repo.events),
		)
	}
}
func TestClickWorkerPoolQueueFull(t *testing.T) {
	repo := &fakeClickRepository{}

	pool := NewClickWorkerPool(
		repo,
		1,
		1,
		3,
		10*time.Millisecond,
	)

	event1 := model.ClickEvent{
		EventKey: "event-1",
		URLID:    1,
	}

	event2 := model.ClickEvent{
		EventKey: "event-2",
		URLID:    1,
	}

	if err := pool.Enqueue(context.Background(), event1); err != nil {
		t.Fatalf("first enqueue failed: %v", err)
	}

	if err := pool.Enqueue(context.Background(), event2); err != ErrQueueFull {
		t.Fatalf(
			"expected ErrQueueFull, got %v",
			err,
		)
	}
}

type retryClickRepository struct {
	attempts int
}

func (r *retryClickRepository) Create(
	_ context.Context,
	_ model.ClickEvent,
) error {
	r.attempts++

	if r.attempts < 3 {
		return errors.New("temporary database error")
	}

	return nil
}

func TestClickWorkerPoolRetry(t *testing.T) {
	repo := &retryClickRepository{}

	pool := NewClickWorkerPool(
		repo,
		1,
		10,
		5,
		10*time.Millisecond,
	)

	pool.Start()

	event := model.ClickEvent{
		EventKey: "retry-event-1",
		URLID:    1,
	}

	if err := pool.Enqueue(context.Background(), event); err != nil {
		t.Fatalf("enqueue failed: %v", err)
	}

	pool.Shutdown()

	if repo.attempts != 3 {
		t.Fatalf(
			"expected 3 attempts, got %d",
			repo.attempts,
		)
	}
}

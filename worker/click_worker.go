package worker

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"

	"goshort/model"
	"goshort/repository"
)

var ErrQueueFull = errors.New("click event queue is full")

type ClickWorkerPool struct {
	queue       chan model.ClickEvent
	repository  repository.ClickEventRepository
	workerCount int
	maxRetries  int
	baseBackoff time.Duration

	wg sync.WaitGroup
}

func NewClickWorkerPool(
	repo repository.ClickEventRepository,
	workerCount int,
	queueSize int,
	maxRetries int,
	baseBackoff time.Duration,
) *ClickWorkerPool {
	if workerCount <= 0 {
		workerCount = 4
	}

	if queueSize <= 0 {
		queueSize = 1000
	}

	if maxRetries <= 0 {
		maxRetries = 5
	}

	if baseBackoff <= 0 {
		baseBackoff = 100 * time.Millisecond
	}

	return &ClickWorkerPool{
		queue:       make(chan model.ClickEvent, queueSize),
		repository:  repo,
		workerCount: workerCount,
		maxRetries:  maxRetries,
		baseBackoff: baseBackoff,
	}
}

func (p *ClickWorkerPool) Start() {
	p.wg.Add(p.workerCount)

	for i := 0; i < p.workerCount; i++ {
		go p.worker(i + 1)
	}
}

func (p *ClickWorkerPool) Enqueue(
	ctx context.Context,
	event model.ClickEvent,
) error {
	select {
	case p.queue <- event:
		return nil

	case <-ctx.Done():
		return ctx.Err()

	default:
		return ErrQueueFull
	}
}

func (p *ClickWorkerPool) worker(id int) {
	defer p.wg.Done()

	for event := range p.queue {
		if err := p.process(event); err != nil {
			log.Printf(
				"click worker %d: failed event=%s: %v",
				id,
				event.EventKey,
				err,
			)
		}
	}
}

func (p *ClickWorkerPool) process(event model.ClickEvent) error {
	var lastErr error

	for attempt := 1; attempt <= p.maxRetries; attempt++ {
		ctx, cancel := context.WithTimeout(
			context.Background(),
			5*time.Second,
		)

		err := p.repository.Create(ctx, event)

		cancel()

		if err == nil {
			return nil
		}

		lastErr = err

		if attempt == p.maxRetries {
			break
		}

		backoff := p.baseBackoff * time.Duration(1<<(attempt-1))

		if backoff > 5*time.Second {
			backoff = 5 * time.Second
		}

		time.Sleep(backoff)
	}

	return lastErr
}

func (p *ClickWorkerPool) Shutdown() {
	close(p.queue)
	p.wg.Wait()
}

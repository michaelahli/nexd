package indexing

import (
	"context"
	"fmt"
	"sync"
	"time"
)

const DefaultPollInterval = time.Second

// Service owns background indexing workers.
type Service struct {
	queue        Queue
	processor    *Processor
	workers      int
	pollInterval time.Duration
	sleep        func(context.Context, time.Duration) error

	mu      sync.Mutex
	cancel  context.CancelFunc
	running bool
	done    chan struct{}
}

// Config controls worker pool behavior.
type Config struct {
	Workers      int
	PollInterval time.Duration
}

// NewService creates an indexing service.
func NewService(queue Queue, processor *Processor, cfg Config) *Service {
	if cfg.Workers <= 0 {
		cfg.Workers = 1
	}
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = DefaultPollInterval
	}
	return &Service{queue: queue, processor: processor, workers: cfg.Workers, pollInterval: cfg.PollInterval, sleep: sleepContext}
}

// Start launches indexing workers.
func (s *Service) Start(ctx context.Context) error {
	if s == nil || s.queue == nil || s.processor == nil {
		return fmt.Errorf("indexing service is not configured")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running {
		return fmt.Errorf("indexing service is already running")
	}

	workerCtx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	s.done = make(chan struct{})
	s.running = true

	var wg sync.WaitGroup
	for i := 0; i < s.workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.worker(workerCtx)
		}()
	}

	go func() {
		wg.Wait()
		close(s.done)
	}()
	return nil
}

// Stop asks workers to exit and waits for them.
func (s *Service) Stop(ctx context.Context) error {
	if s == nil {
		return nil
	}

	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return nil
	}
	cancel := s.cancel
	done := s.done
	s.running = false
	s.mu.Unlock()

	cancel()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *Service) worker(ctx context.Context) {
	for {
		job, ok, err := s.queue.Next(ctx)
		if err != nil {
			return
		}
		if !ok {
			if err := s.sleep(ctx, s.pollInterval); err != nil {
				return
			}
			continue
		}

		processed, err := s.processor.Process(ctx, job)
		if err != nil {
			_ = s.queue.Fail(ctx, job.ID, err)
			continue
		}
		_ = s.queue.Complete(ctx, job.ID, processed)
	}
}

func sleepContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

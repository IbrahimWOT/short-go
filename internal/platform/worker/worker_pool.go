package worker

import (
	"context"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"short-go/internal/config"
	"short-go/internal/domain"
)

// AnalyticsWorkerPool manages bounded concurrency and batch persistence for redirection click analytics.
type AnalyticsWorkerPool struct {
	repo          domain.AnalyticsRepository
	cfg           config.WorkerConfig
	events        chan domain.ClickEvent
	wg            sync.WaitGroup
	stopped       atomic.Bool
	stopOnce      sync.Once
	droppedEvents atomic.Uint64
}

// NewAnalyticsWorkerPool creates a new bounded worker pool.
func NewAnalyticsWorkerPool(repo domain.AnalyticsRepository, cfg config.WorkerConfig) *AnalyticsWorkerPool {
	return &AnalyticsWorkerPool{
		repo:   repo,
		cfg:    cfg,
		events: make(chan domain.ClickEvent, cfg.BufferSize),
	}
}

// Enqueue adds a click event into the buffered channel non-blockingly.
// If the buffer is full, it safely increments droppedEvents and returns false.
func (p *AnalyticsWorkerPool) Enqueue(event domain.ClickEvent) bool {
	if p.stopped.Load() {
		return false
	}

	select {
	case p.events <- event:
		return true
	default:
		p.droppedEvents.Add(1)
		log.Printf("[WorkerPool WARNING] Analytics buffer full, dropped click for shortCode: %s", event.ShortCode)
		return false
	}
}

// DroppedCount returns the number of events dropped due to buffer saturation.
func (p *AnalyticsWorkerPool) DroppedCount() uint64 {
	return p.droppedEvents.Load()
}

// Start launches the worker goroutines.
func (p *AnalyticsWorkerPool) Start(ctx context.Context) {
	for i := 0; i < p.cfg.PoolSize; i++ {
		p.wg.Add(1)
		go p.runWorker(ctx, i)
	}
	log.Printf("[WorkerPool] Started %d analytics workers (buffer: %d, batch: %d, flush: %v)",
		p.cfg.PoolSize, p.cfg.BufferSize, p.cfg.BatchSize, p.cfg.FlushInterval)
}

// runWorker processes incoming events, batches them, and flushes periodically or when full.
func (p *AnalyticsWorkerPool) runWorker(ctx context.Context, workerID int) {
	defer p.wg.Done()

	ticker := time.NewTicker(p.cfg.FlushInterval)
	defer ticker.Stop()

	batch := make([]*domain.ClickLog, 0, p.cfg.BatchSize)

	flush := func() {
		if len(batch) == 0 {
			return
		}

		flushCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := p.repo.RecordClicksBatch(flushCtx, batch); err != nil {
			log.Printf("[Worker %d ERROR] Failed to flush %d clicks to db: %v", workerID, len(batch), err)
		}

		// Reset batch keeping underlying slice capacity
		batch = make([]*domain.ClickLog, 0, p.cfg.BatchSize)
	}

	for {
		select {
		case event, ok := <-p.events:
			if !ok {
				// Channel closed, flush remaining items and exit
				flush()
				return
			}

			clickLog := &domain.ClickLog{
				URLID:     event.URLID,
				ShortCode: event.ShortCode,
				ClickedAt: event.ClickedAt,
				IPAddress: event.IPAddress,
				UserAgent: event.UserAgent,
				Referrer:  event.Referrer,
			}
			batch = append(batch, clickLog)

			if len(batch) >= p.cfg.BatchSize {
				flush()
			}

		case <-ticker.C:
			flush()
		}
	}
}

// Stop initiates graceful shutdown by closing the channel and waiting for workers to drain pending events.
func (p *AnalyticsWorkerPool) Stop(ctx context.Context) error {
	var err error
	p.stopOnce.Do(func() {
		p.stopped.Store(true)
		close(p.events)

		done := make(chan struct{})
		go func() {
			p.wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			log.Println("[WorkerPool] All analytics workers drained and exited cleanly.")
		case <-ctx.Done():
			err = ctx.Err()
			log.Printf("[WorkerPool WARNING] Worker shutdown timed out: %v", err)
		}
	})

	return err
}

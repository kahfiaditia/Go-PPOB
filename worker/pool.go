package worker

import (
	"context"
	"sync"
	"time"

	"pulsa-backend/service"
	"pulsa-backend/util"
)

type Job struct {
	TxID string
}

type Pool struct {
	queue   chan Job
	svc     *service.TopupService
	workers int
	mu      sync.Mutex
	queued  map[string]struct{}
}

func NewPool(workers, queueSize int, svc *service.TopupService) *Pool {
	return &Pool{
		queue:   make(chan Job, queueSize),
		svc:     svc,
		workers: workers,
		queued:  make(map[string]struct{}),
	}
}

func (p *Pool) Start(ctx context.Context) {
	for i := 0; i < p.workers; i++ {
		go p.worker(ctx, i)
	}
	go p.recoverPending(ctx)
	util.Log.Infof("[WORKER] pool started with %d workers", p.workers)
}

func (p *Pool) worker(ctx context.Context, id int) {
	util.Log.Infof("[WORKER-%d] ready", id)
	for {
		select {
		case <-ctx.Done():
			util.Log.Infof("[WORKER-%d] shutdown", id)
			return
		case job, ok := <-p.queue:
			if !ok {
				return
			}
			util.Log.Debugf("[WORKER-%d] processing tx=%s", id, job.TxID)
			p.svc.Process(ctx, job.TxID)
			p.removeQueued(job.TxID)
		}
	}
}

// Enqueue — non-blocking, return false jika queue penuh (backpressure)
func (p *Pool) Enqueue(job Job) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, exists := p.queued[job.TxID]; exists {
		return true
	}
	select {
	case p.queue <- job:
		p.queued[job.TxID] = struct{}{}
		return true
	default:
		return false
	}
}

func (p *Pool) recoverPending(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		pendingIDs, err := p.svc.PendingTransactionIDs(ctx, cap(p.queue))
		if err != nil && ctx.Err() == nil {
			util.Log.Errorf("[WORKER] failed to recover pending transactions: %v", err)
		} else if err == nil {
			for _, id := range pendingIDs {
				if !p.Enqueue(Job{TxID: id}) {
					break
				}
			}
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (p *Pool) removeQueued(txID string) {
	p.mu.Lock()
	delete(p.queued, txID)
	p.mu.Unlock()
}

func (p *Pool) QueueSize() int { return len(p.queue) }

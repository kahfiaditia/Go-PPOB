package worker

import (
	"context"
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
}

func NewPool(workers, queueSize int, svc *service.TopupService) *Pool {
	return &Pool{
		queue:   make(chan Job, queueSize),
		svc:     svc,
		workers: workers,
	}
}

func (p *Pool) Start(ctx context.Context) {
	for i := 0; i < p.workers; i++ {
		go p.worker(ctx, i)
	}
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
		}
	}
}

// Enqueue — non-blocking, return false jika queue penuh (backpressure)
func (p *Pool) Enqueue(job Job) bool {
	select {
	case p.queue <- job:
		return true
	default:
		return false
	}
}

func (p *Pool) QueueSize() int { return len(p.queue) }

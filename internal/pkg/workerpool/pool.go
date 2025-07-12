package workerpool

import (
	"context"
	"log"
	"sync"
	"time"
)

type Job func(ctx context.Context)

type WorkerPool struct {
	queue chan Job
	wg    sync.WaitGroup
}

func NewWorkerPool(ctx context.Context, workerCount int, queueSize int) *WorkerPool {
	pool := &WorkerPool{
		queue: make(chan Job, queueSize),
	}

	for range workerCount {
		go pool.worker(ctx)
	}

	return pool
}

func (p *WorkerPool) worker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			log.Println("Worker received shutdown signal")
			return
		case job, ok := <-p.queue:
			if !ok {
				// queue closed
				return
			}
			p.wg.Add(1)
			job(ctx) // Now jobs can watch the cancellation context!
			p.wg.Done()
		}
	}
}

func (p *WorkerPool) Submit(job Job) {
	select {
	case p.queue <- job:
		// job submitted successfully
	default:
		log.Println("Worker pool queue full: job dropped or handled differently")
	}
}

func (p *WorkerPool) Shutdown(ctx context.Context) {
	close(p.queue)

	done := make(chan struct{})

	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-ctx.Done():
		log.Println("Worker pool shutdown timed out")
	case <-done:
		log.Println("Worker pool shutdown complete")
	}
}

func WithRetry(retries int, delay time.Duration, job func() error) func(ctx context.Context) {
	return func(ctx context.Context) {
		for i := range retries {
			if ctx.Err() != nil {
				log.Println("Job canceled before execution")
				return
			}

			err := job()

			if err == nil {
				return // success
			}
			log.Printf("Job failed (attempt %d/%d): %v", i+1, retries, err)
			time.Sleep(delay)
		}
		log.Println("Job failed after max retries")
	}
}

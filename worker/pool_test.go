package worker

import (
	"sync"
	"testing"
)

func TestEnqueueDeduplicatesAndRespectsCapacity(t *testing.T) {
	pool := NewPool(1, 1, nil)

	var wg sync.WaitGroup
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if !pool.Enqueue(Job{TxID: "tx-1"}) {
				t.Error("duplicate transaction should already be accepted")
			}
		}()
	}
	wg.Wait()

	if got := len(pool.queue); got != 1 {
		t.Fatalf("queue length = %d, want 1", got)
	}
	if pool.Enqueue(Job{TxID: "tx-2"}) {
		t.Fatal("enqueue should fail when queue is full")
	}

	<-pool.queue
	pool.removeQueued("tx-1")
	if !pool.Enqueue(Job{TxID: "tx-2"}) {
		t.Fatal("enqueue should succeed after freeing a queue slot")
	}
}

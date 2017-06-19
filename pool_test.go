package pool

import (
	"os"
	"sync"
	"testing"
)

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func TestPool(t *testing.T) {
	pool, _ := NewWorkerPool(4)
	pool.Drain()

	wg := sync.WaitGroup{}
	mu := sync.Mutex{}
	passed := 0
	iterationCount := 4
	wg.Add(iterationCount)
	for i := 0; i < iterationCount; i++ {
		f := func() error {
			defer wg.Done()
			mu.Lock()
			passed += 1
			mu.Unlock()
			return nil
		}
		pool.Add(f)
	}
	wg.Wait()

	if passed != 4 {
		t.Fatalf("expected passed count to be %d but was %d", iterationCount, passed)
	}
}

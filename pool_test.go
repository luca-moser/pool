package pool

import (
	"os"
	"sync"
	"testing"
	"time"
)

func must(err error, t *testing.T) {
	if err != nil {
		t.Fatal(err)
	}
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func TestPool(t *testing.T) {
	pool, _ := NewWorkerPool(40)
	pool.Drain()

	const iterationCount = 500

	wg := sync.WaitGroup{}
	wg.Add(iterationCount)

	mu := sync.Mutex{}
	passed := 0
	for i := 0; i < iterationCount; i++ {
		f := func() error {
			defer wg.Done()
			mu.Lock()
			passed += 1
			mu.Unlock()
			<-time.After(time.Duration(10) * time.Millisecond)
			return nil
		}
		pool.Add(f)
	}
	wg.Wait()

	if passed != iterationCount {
		t.Fatalf("expected passed count to be %d but was %d", iterationCount, passed)
	}
}

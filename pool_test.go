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

	mu := sync.Mutex{}
	passed := 0
	for i := 0; i < iterationCount; i++ {
		f := func() error {
			mu.Lock()
			passed += 1
			mu.Unlock()
			<-time.After(time.Duration(10) * time.Millisecond)
			return nil
		}
		pool.Add(f)
	}
	pool.Wait(500)
	pool.Stop()

	if passed != iterationCount {
		t.Fatalf("expected passed count to be %d but was %d", iterationCount, passed)
	}
}

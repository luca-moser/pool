package pool

import (
	"os"
	"sync/atomic"
	"testing"
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
	testDrives := 50
	for i := 0; i < testDrives; i++ {
		pool, _ := NewWorkerPool(10)
		pool.Drain()

		const iterationCount = 3000

		var passed int32
		for i := 0; i < iterationCount; i++ {
			f := func() error {
				atomic.AddInt32(&passed, 1)
				return nil
			}
			pool.Add(f)
		}
		pool.Wait(iterationCount)
		pool.Stop()

		if passed != iterationCount {
			t.Fatalf("expected passed count to be %d but was %d", iterationCount, passed)
		}
	}
}

package pool

import (
	"os"
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
	testDrives := 100000

	f := func() (interface{}, error) {
		return 1, nil
	}

	for i := 0; i < testDrives; i++ {
		workerPool, err := NewWorkerPool(8)
		if err != nil {
			panic(err)
		}

		const iterationCount = 100
		var processed int

		resultHandler := func(result interface{}) {
			processed += result.(int)
		}

		errorHandler := func(err error) {
			t.Fatal(err)
		}

		workerPool.AddHandlers(resultHandler, errorHandler)

		for i := 0; i < iterationCount; i++ {
			workerPool.AddFuncWithResult(f)
		}

		workerPool.Wait(iterationCount)
		workerPool.Stop()

		if workerPool.jobsDone != workerPool.jobsReceived {
			t.Fatal("expected jobs done (%d) to equal jobs received (%d)", workerPool.jobsDone, workerPool.jobsReceived)
		}

		if processed != iterationCount {
			t.Fatalf("expected passed count to be %d but was %d", iterationCount, processed)
		}
	}

}

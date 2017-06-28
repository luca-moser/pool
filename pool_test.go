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
	testDrives := 1000

	for i := 0; i < testDrives; i++ {
		pool, err := NewWorkerPool(10)
		if err != nil {
			panic(err)
		}

		exit := make(chan interface{})
		const iterationCount = 100
		var processed int

		ready := make(chan struct{})
		go func() {
			readySent := false
			resultsChann := pool.Results()
			errorsChann := pool.Errors()
		exit:
			for {
				if !readySent {
					ready <- struct{}{}
					readySent = true
				}
				select {
				case result := <-resultsChann:
					processed += result.(int)
				case <-errorsChann:
					processed += 1
				case <-exit:
					break exit
				}
			}
		}()
		<-ready

		for i := 0; i < iterationCount; i++ {
			pool.AddFuncWithResult(func() (interface{}, error) {
				return 2, nil
			})
		}
		pool.Wait(iterationCount)
		exit <- struct{}{}
		pool.Stop()

		if pool.jobsDone != pool.jobsReceived {
			t.Fatal("expected jobs done (%d) to equal jobs received (%d)", pool.jobsDone, pool.jobsReceived)
		}

		if processed != iterationCount*2 {
			t.Fatalf("expected passed count to be %d but was %d", iterationCount, processed)
		}
	}

}

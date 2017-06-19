package pool

import (
	"sync"

	"github.com/pkg/errors"
)

var ErrInvalidWorkerCount = errors.New("worker count must be positive and bigger than 0")

type JobFunction func(...interface{}) (interface{}, error)

type Job struct {
	Arguments []interface{}
	Function  JobFunction
}

func NewWorkerPool(numWorker int) (*WorkerPool, error) {
	if numWorker <= 0 {
		return nil, ErrInvalidWorkerCount
	}

	pool := &WorkerPool{workers: []*worker{}}
	pool.in = make(chan Job, numWorker)
	pool.err = make(chan error)
	pool.stop = make(chan struct{})
	pool.results = make(chan interface{})
	pool.done = make(chan struct{})
	pool.resume = make(chan struct{}, numWorker)

	// spawn workers
	for i := 0; i < numWorker; i++ {
		pool.workers = append(pool.workers, newSyncWorker(i, pool.err, pool.results, pool.done, pool.resume))
	}

	pool.init()
	return pool, nil
}

type WorkerPool struct {
	workers      []*worker
	in           chan Job
	err          chan error
	stop         chan struct{}
	results      chan interface{}
	done         chan struct{}
	resume       chan struct{}
	IsRunning    bool
	jobsReceived int64
	jobsDoneMu   sync.Mutex
	jobsDone     int64
	mu           sync.Mutex
}

func (wp *WorkerPool) init() {
	wp.IsRunning = true

	// fill up the drain channel
	for i := 0; i < len(wp.workers); i++ {
		wp.resume <- struct{}{}
	}

	// fill up drain channel
	go func() {
		for wp.IsRunning {
			wp.resume <- struct{}{}
		}
	}()

	go func() {
	exit:
		for {
			select {

			case job := <-wp.in:
				wp.jobsReceived++

				if wp.distributeJob(job) {
					continue
				}

				// wait for a done signal of at least one worker if
				// all workers were busy
				<-wp.done

				// must work, as a worker signaled being done with its current job
				if !wp.distributeJob(job) {
					panic("no worker was available for job distribution but one signaled it was ready")
				}

				if len(wp.workers) == 1 {
					continue
				}

			case <-wp.stop:
				wp.mu.Lock()
				for x := range wp.workers {
					wp.workers[x].Stop()
				}
				wp.mu.Unlock()

				close(wp.err)
				break exit
			}
		}
		wp.IsRunning = false
	}()
}

func (wp *WorkerPool) distributeJob(job Job) bool {
	sent := false
	wp.mu.Lock()
jobSent:
	for x := range wp.workers {
		worker := wp.workers[x]
		// send job to first free worker
		select {
		case worker.in <- job:
			sent = true
			break jobSent
		default:
		}
	}
	wp.mu.Unlock()
	return sent
}

func (wp *WorkerPool) incrementJobsDoneCounter() {
	wp.jobsDoneMu.Lock()
	wp.jobsDone++
	wp.jobsDoneMu.Unlock()
}

// stops any further processing inside the pool
func (wp *WorkerPool) Stop() {
	wp.stop <- struct{}{}
}

// blocks the calling goroutine as long as workers are busy
func (wp *WorkerPool) Wait() {

}

// returns a channel on which errors of workers can be fetched
func (wp *WorkerPool) Errors() <-chan error {
	channel := make(chan error)
	go func() {
		for err := range wp.err {
			wp.incrementJobsDoneCounter()
			channel <- err
		}
	}()
	return channel
}

// returns a channel on which results of workers can be fetched
func (wp *WorkerPool) Results() <-chan interface{} {
	channel := make(chan interface{})
	go func() {
		for result := range wp.results {
			wp.incrementJobsDoneCounter()
			channel <- result
		}
	}()
	return channel
}

// drain error and result channel
func (wp *WorkerPool) Drain() {
	wp.DrainResults()
	wp.DrainErrors()
}

// unblocks result channel bei discarding all worker results
func (wp *WorkerPool) DrainResults() {
	go func() {
		for range wp.Results() {
		}
	}()
}

// unblocks error channel bei discarding all worker errors
func (wp *WorkerPool) DrainErrors() {
	go func() {
		for range wp.Errors() {
		}
	}()
}

func (wp *WorkerPool) Stat() map[string]int64 {
	stat := map[string]int64{}
	wp.mu.Lock()
	for x := range wp.workers {
		w := wp.workers[x]
		stat[w.id] = w.jobProcessed
	}
	wp.mu.Unlock()
	return stat
}

func (wp *WorkerPool) AddJob(j Job) {
	wp.in <- j
}

func (wp *WorkerPool) Add(f func() error) {
	wrapped := func(...interface{}) (interface{}, error) {
		return nil, f()
	}
	wp.in <- Job{Function: wrapped, Arguments: nil}
}

func (wp *WorkerPool) AddWithOut(f func() (interface{}, error)) {
	wrapped := func(...interface{}) (interface{}, error) {
		return f()
	}
	wp.in <- Job{Function: wrapped, Arguments: nil}
}

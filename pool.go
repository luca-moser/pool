package pool

import (
	"sync"

	"reflect"

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
	pool.waitUnblock = make(chan struct{}, 1)

	// spawn workers
	for i := 0; i < numWorker; i++ {
		pool.workers = append(pool.workers, newSyncWorker(i, pool.err, pool.results))
	}

	pool.loop()
	return pool, nil
}

type WorkerPool struct {
	workers      []*worker
	in           chan Job
	err          chan error
	stop         chan struct{}
	results      chan interface{}
	waitUnblock  chan struct{}
	IsRunning    bool
	jobsReceived int64
	jobsDoneMu   sync.Mutex
	jobsDone     int64
	mu           sync.Mutex
}

func (wp *WorkerPool) loop() {
	wp.IsRunning = true

	go func() {
	exit:
		for {
			select {

			case job := <-wp.in:
				wp.jobsReceived++
				wp.assignJob(job)

			case <-wp.stop:
				wp.mu.Lock()
				for x := range wp.workers {
					wp.workers[x].Stop()
				}
				wp.mu.Unlock()

				close(wp.err)
				close(wp.results)
				close(wp.waitUnblock)
				break exit
			}
		}
		wp.IsRunning = false
	}()
}

func (wp *WorkerPool) assignJob(job Job) {
	wp.mu.Lock()
	cases := []reflect.SelectCase{}
	for x := range wp.workers {
		worker := wp.workers[x]
		cases = append(cases, reflect.SelectCase{
			Chan: reflect.ValueOf(worker.in),
			Dir:  reflect.SelectSend, Send: reflect.ValueOf(job),
		})
	}
	wp.mu.Unlock()
	reflect.Select(cases)
}

func (wp *WorkerPool) incrementJobsDoneCounter() {
	wp.jobsDoneMu.Lock()
	wp.jobsDone++
	wp.jobsDoneMu.Unlock()
	select {
	case wp.waitUnblock <- struct{}{}:
	default:
	}

}

// stops any further processing inside the pool
func (wp *WorkerPool) Stop() {
	wp.stop <- struct{}{}
}

// blocks the calling goroutine as long as workers are busy
// respectively the given "jobs done" count is reached
func (wp *WorkerPool) Wait(howMany int64) {
	for true {
		if wp.jobsDone != howMany {
			<-wp.waitUnblock
			continue
		}
		break
	}
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

func (wp *WorkerPool) Stats() map[string]int64 {
	stats := map[string]int64{}
	wp.mu.Lock()
	for x := range wp.workers {
		w := wp.workers[x]
		stats[w.id] = w.jobProcessed
	}
	wp.mu.Unlock()
	return stats
}

func (wp *WorkerPool) AddJob(j Job) {
	wp.in <- j
}

func (wp *WorkerPool) Add(f func() error) {
	wp.in <- Job{Function: func(...interface{}) (interface{}, error) {
		return nil, f()
	}, Arguments: nil}
}

func (wp *WorkerPool) AddWithOut(f func() (interface{}, error)) {
	wp.in <- Job{Function: func(...interface{}) (interface{}, error) {
		return f()
	}, Arguments: nil}
}

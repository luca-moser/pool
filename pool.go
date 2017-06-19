package pool

import (
	"reflect"

	"sync/atomic"

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
	pool.stats = make(chan chan map[string]int64)

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
	stats        chan chan map[string]int64
	IsRunning    bool
	jobsReceived int64
	jobsDone     int64
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
				for x := range wp.workers {
					wp.workers[x].Stop()
				}

				close(wp.err)
				close(wp.results)
				close(wp.waitUnblock)
				break exit

			case back := <-wp.stats:
				stats := map[string]int64{}
				for x := range wp.workers {
					w := wp.workers[x]
					stats[w.id] = w.jobProcessed
				}
				back <- stats
			}
		}
		wp.IsRunning = false
	}()
}

func (wp *WorkerPool) assignJob(job Job) {
	cases := []reflect.SelectCase{}
	for x := range wp.workers {
		worker := wp.workers[x]
		cases = append(cases, reflect.SelectCase{
			Chan: reflect.ValueOf(worker.in),
			Dir:  reflect.SelectSend, Send: reflect.ValueOf(job),
		})
	}
	reflect.Select(cases)
}

func (wp *WorkerPool) incrementJobsDoneCounter() {
	atomic.AddInt64(&wp.jobsDone, 1)
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
	b := make(chan map[string]int64)
	wp.stats <- b
	return <-b
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

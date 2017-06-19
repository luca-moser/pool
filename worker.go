package pool

import (
	"strconv"

	"github.com/pkg/errors"
)

var ErrWorkerPanic = errors.New("worker panicked when executing job function")

type worker struct {
	// identification
	id string
	// to feed in jobs
	in chan Job
	// to send out errors
	err chan<- error
	// to send out results
	results chan<- interface{}
	// to receive a stop signal
	stop chan struct{}
	// to receive/drain from when no done signal is needed
	resume <-chan struct{}
	// jobs count processed
	jobProcessed int64
	// error count
	errorsCount int64
}

func newSyncWorker(id int, errChan chan<- error, resultChann chan<- interface{}) *worker {
	w := &worker{
		id:      strconv.Itoa(id),
		in:      make(chan Job),
		err:     errChan,
		stop:    make(chan struct{}),
		results: resultChann,
	}
	w.init()
	return w
}

func (w *worker) Stop() {
	w.stop <- struct{}{}
}

func (w *worker) init() {
	go func() {
	exit:
		for {
			select {

			case <-w.stop:
				break exit

			case j := <-w.in:

				// call function
				func() {
					defer func() {
						if r := recover(); r != nil {
							w.err <- errors.Wrapf(ErrWorkerPanic, "panic: %v", r)
						}
					}()

					value, err := j.Function(j.Arguments)

					// an err occurred
					if err != nil {
						w.err <- errors.WithStack(err)
						w.errorsCount++
						return
					}

					w.results <- value
				}()

				w.jobProcessed++
			}
		}
	}()
}

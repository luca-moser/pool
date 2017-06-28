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
	errs chan<- error
	// to send out results
	results chan<- interface{}
	// to receive a stop signal
	stop chan struct{}
	// jobs count processed
	jobProcessed int64
	// error count
	errorsCount int64
}

func newSyncWorker(id int, errChan chan<- error, resultChan chan<- interface{}) *worker {
	w := &worker{
		id:      strconv.Itoa(id),
		in:      make(chan Job),
		errs:    errChan,
		stop:    make(chan struct{}),
		results: resultChan,
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

			case j, ok := <-w.in:
				if !ok {
					break exit
				}

				// call function
				func() {
					defer func() {
						if r := recover(); r != nil {
							w.errs <- errors.Wrapf(ErrWorkerPanic, "panic: %v", r)
						}
					}()

					value, err := j.Function(j.Arguments)

					// an errs occurred
					if err != nil {
						w.errs <- errors.WithStack(err)
						w.errorsCount++
						return
					}

					w.results <- value
				}()

				w.jobProcessed++
			}
		}
		close(w.stop)
	}()
}

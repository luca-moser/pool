package pool

import "github.com/pkg/errors"

var ErrWorkerPanic = errors.New("worker panicked when executing job function")

func newSyncWorker(id int, errChan chan<- error, resultChann chan<- interface{}, done chan<- struct{}, drainDone <-chan struct{}) *worker {
	w := &worker{
		id:      id,
		in:      make(chan Job),
		err:     errChan,
		done:    done,
		stop:    make(chan struct{}),
		results: resultChann,
	}
	w.Init()
	return w
}

type worker struct {
	// identification
	id int
	// to feed in jobs
	in chan Job
	// to send out errors
	err chan<- error
	// to send out results
	results chan<- interface{}
	// to receive a stop signal
	stop chan struct{}
	// to send a done signal
	done chan<- struct{}
	// to receive/drain from when no done signal is needed
	drainDone <-chan struct{}
	// jobs count processed
	jobProcessed int64
}

func (w *worker) Stop() {
	w.stop <- struct{}{}
}

func (w *worker) Init() {
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
					}

					w.results <- value
				}()

				w.jobProcessed++
				select {
				case w.done <- struct{}{}:
				case <-w.drainDone:
				}

			}
		}
	}()
}

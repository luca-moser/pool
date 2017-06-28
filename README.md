# Worker Pool [![Build Status](https://travis-ci.org/luca-moser/pool.svg?branch=master)](https://travis-ci.org/luca-moser/pool)

A simple worker pool which receives jobs and distributes them to workers.
Workers can send errors and results back to the pool creator through channels or handler functions.

## Usage
```go
// new pool with 8 worker goroutines
workerPool, err := pool.NewWorkerPool(8)
if err != nil {
    ...
}

const iterationCount = 100
var processed int

// called when a result was produced by a worker
resultHandler := func(result interface{}) {
    // no mutex needed here as the resultHandler is not called concurrently
    processed += result.(int)
}

// called when an error was produced by a worker
errorHandler := func(err error) {
    t.Fatal(err)
}

// define the handler functions on the pool
workerPool.AddHandlers(resultHandler, errorHandler)

// OR discard any results and errors (not to be used with AddHandlers simultaneously(!))
// workerPool.Discard()

// add jobs to the pool
for i := 0; i < iterationCount; i++ {
    // add a function to the pool for execution which returns a result or error
    workerPool.AddFuncWithResult(f)
    
    // OR a function which does not return anything, except maybe an error
    // workerPool.AddFunc(func() error {})
    
    // OR a job
    // workerPool.AddJob(Job{
    //     Function: func(args ...interface{}) (interface{}, error) {
    //         return nil, nil
    //     },
    //     Arguments: []interface{}{1,2,3,4},
    // })
}

// wait for the amount of jobs to complete
workerPool.Wait(iterationCount)

// free up goroutines and close channels
workerPool.Stop()

// returns information about each worker's processed jobs
workerPool.Stats()
```

### Job results and errors
It's important that the error and result channels are being actively used or discarded.
This can be achieved by using the `Errors()` and `Results()` functions which give you channels
to listen for or `DiscardErrors()` and `DiscardResults()` or `Discard()` (errors and results combined).
By using the `AddHandlers` function handlers can be defined which are executed as soon as results and errors are
produced by workers.

### Other
If a worker panics during the execution of the job, an error will be generated, the worker will then resume normally.
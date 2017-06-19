# Worker Pool [![Build Status](https://travis-ci.org/luca-moser/pool.svg?branch=master)](https://travis-ci.org/luca-moser/pool)

```diff
- This is a work in progress!
```

A simple worker pool which receives jobs and distributes them to workers.
Workers can send errors and results back to the pool creator through channels.

## Usage
```go
// a pool with 4 worker goroutines
pool, _ := NewWorkerPool(4)

// auto. drain errors and results
pool.Drain()

const amountOfJobs = 10

for i := 0; i < amountOfJobs; i++ {
    // submit a function to the pool which the next
    // free worker goroutine will execute
    pool.Add(func() error {
         // do something...
         return nil
     })
}
// wait() waits until the given amount of jobs were processed by the pool
pool.Wait(amountOfJobs)

// free up goroutines
pool.Stop()
```

### Results and errors
It's important that the error and result channels are being actively used or drained.
This can be achived by using the `Errors()` and `Results()` functions which give you channels
to listen for or `DrainErrors()` and `DrainResults()` or `Drain()` (errors and results combined).

### Other
If a worker panics during the execution of the job, an error will be generated, the worker will then resume normally.
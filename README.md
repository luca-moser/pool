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

const jobsToCreate = 10
wg := sync.WaitGroup{}
wg.Add(jobsToCreate)

for i := 0; i < jobsToCreate; i++ {
    // submit a function to the pool which the next
    // free worker goroutine will execute
    pool.Add(func() error {
         defer wg.Done()
         // do something expensive
         return nil
     })
}
wg.Wait()

// free up goroutines
pool.Stop()
```

### Results and errors
It's important that the error and result channels are being actively used or drained.
This can be achived by using the `Errors()` and `Results()` functions which give you channels
to listen for or `DrainErrors()` and `DrainResults()` or `Drain()` (errors and results combined).

### Other
If a worker panics during the execution of the job, an error will be generated, the worker will then resume normally.
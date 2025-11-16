package helpers

import (
	"runtime"
	"sync"
)

type Parallelizer struct {
	sem chan struct{}
	wg  sync.WaitGroup
}

func NewParallelizer(workers int) *Parallelizer {
	if workers <= 0 {
		workers = defaultParallelism()
	}
	return &Parallelizer{sem: make(chan struct{}, workers)}
}

func (p *Parallelizer) Go(fn func()) {
	p.wg.Add(1)
	p.sem <- struct{}{}
	go func() {
		defer p.wg.Done()
		defer func() { <-p.sem }()
		fn()
	}()
}

func (p *Parallelizer) Wait() {
	p.wg.Wait()
}

func defaultParallelism() int {
	cpu := runtime.NumCPU()
	if cpu < 2 {
		cpu = 2
	}
	limit := cpu * 2
	if limit > 12 {
		limit = 12
	}
	return limit
}

func DetermineParallelism(total int) int {
	limit := defaultParallelism()
	if total > 0 && total < limit {
		limit = total
	}
	if limit < 2 {
		limit = 2
	}
	return limit
}

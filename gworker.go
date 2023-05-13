package gworker

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

// Pool is a Generic worker pool implementation
type Pool[T any, P any] struct {
	started     bool
	size        int
	batched     bool
	dataSources dS[T]
	workerFunc  func(dataSource T, params []P, errChan chan error)
	errChan     chan error
	ctx         context.Context
	cancel      context.CancelFunc
	doneChan    chan struct{}
}

type dS[T any] struct {
	sync.Mutex
	data []T
}

var missingWorkerFuncError = errors.New("no worker func provided")

// NewPool initializes a new Pool with the provided dataSources, worker func, any value and error channels, then returns it;
// to prevent blocking and allow dataSources whose length is greater than the provided (or default) Size,
// any channels used must be buffered channels
func NewPool[T any, P any](dataSources []T, worker func(dataSource T, params []P, errChan chan error), errorChannel chan error) (*Pool[T, P], error) {
	if worker == nil {
		return nil, missingWorkerFuncError
	}
	p := Pool[T, P]{
		started: false,
		batched: true,
		dataSources: dS[T]{
			data: dataSources,
		},
		size:       5,
		errChan:    errorChannel,
		workerFunc: worker,
	}
	return &p, nil
}

// Stop gracefully stops a Pool if .WithCancel() was called with a context. If Stop is called
// without calling .WithCancel(), nothing will happen and the Pool will continue running
func (p *Pool[T, P]) Stop() {
	if p.cancel != nil {
		p.cancel()
	}
}

func (p *Pool[T, P]) Context() context.Context {
	if p.ctx != nil {
		return p.ctx
	}
	return nil
}

// Size determines the max number of concurrent workers to run at once whenever the Pool has
// been started. If Size is not called, a Pool will default to a size of 5
func (p *Pool[T, P]) Size(size int) *Pool[T, P] {
	if p.started {
		p.Stop()
	}
	p.size = size
	return p
}

// WithAutoRefill switches the Pool from its default "Batched" state (where the Pool
// loops through the data provided to it and runs them in batches matching the Size provided
// until all the dataSources are exhausted) to an "AutoPoolRefill" state (where, as soon as one
// worker goroutine finishes, if any dataSources remain, another worker goroutine will immediately
// be spun up until all dataSources have been exhausted)
func (p *Pool[T, P]) WithAutoRefill() *Pool[T, P] {
	p.batched = false
	return p
}

// WithCancel allows you to provide a Pool. WithCancel returns the Pool and stores a new
// context.Context that wraps the provided context.Context. This new context and its
// context.CancelFunc are also stored within the Pool. To retrieve the context, call .Context()
// and to call the context.CancelFunc, call .Stop()
func (p *Pool[T, P]) WithCancel(ctx context.Context) *Pool[T, P] {
	p.ctx, p.cancel = context.WithCancel(ctx)

	return p
}

// Start starts running the Pool's workers and injects any provided channels, along with the
// provided funcParams, to each worker goroutine
func (p *Pool[T, P]) Start(funcParams []P) {
	if p.ctx == nil &&
		!p.batched {
		p.ctx, p.cancel = context.WithCancel(context.Background())
	}

	dataLen := len(p.dataSources.data)

	if p.size > dataLen {
		p.size = dataLen
	}

	var counter atomic.Uint64

	doneChan := make(chan struct{})

	if p.batched {
		defer close(doneChan)
		batchSlice := p.dataSources.data[:p.size]
		remainingSlice := p.dataSources.data[p.size:]

		wg := &sync.WaitGroup{}

		for i := 0; i < len(batchSlice); i++ {

			wg.Add(1)

			i := i

			if p.cancel != nil {
				go func(p *Pool[T, P], wg *sync.WaitGroup) {
					defer wg.Done()
					for {
						select {
						case <-p.ctx.Done():
							return
						default:
							p.workerFunc(p.dataSources.data[i], funcParams, p.errChan)
						}
					}
				}(p, wg)
			} else {
				go func(p *Pool[T, P]) {
					defer wg.Done()

					p.workerFunc(p.dataSources.data[i], funcParams, p.errChan)
				}(p)
			}
		}

		switch p.cancel == nil {
		case true:
			wg.Wait()
		case false:
			go func(wg *sync.WaitGroup) {
				wg.Wait()
			}(wg)
		}

		p.checkContinueBatch(remainingSlice, funcParams)
	} else {
		var done chan struct{}
		if p.doneChan != nil {
			done = p.doneChan
		} else {
			done = make(chan struct{})
			p.doneChan = done
		}
		if p.ctx != nil {
			for {
				select {
				case <-p.ctx.Done():
					return
				case <-doneChan:
					var l = 0
					if p.dataSources.TryLock() {
						l = len(p.dataSources.data)
					} else {
						p.tryLock(&p.dataSources.data)
						l = len(p.dataSources.data)
					}
					p.dataSources.Unlock()
					counter.Add(^uint64(0))
					if l != 0 &&
						int(counter.Load()) != 0 {
						p.replenishPool(&counter, doneChan, funcParams)
					} else {
						continue
					}
				default:
					var l = 0
					if p.dataSources.TryLock() {
						l = len(p.dataSources.data)
					} else {
						p.tryLock(&p.dataSources.data)
						l = len(p.dataSources.data)
					}
					p.dataSources.Unlock()
					if l != 0 {
						p.replenishPool(&counter, doneChan, funcParams)
					} else {
						return
					}
				}
			}
		}
		done <- struct{}{}
		close(doneChan)
	}
	if !p.batched {
		<-p.doneChan
	}
}

func (p *Pool[T, P]) replenishPool(counter *atomic.Uint64, doneChan chan struct{}, funcParams []P) {
	n := int(counter.Load())
	if n < p.size {
		num := uint64(p.size - n)
		p.runWithAutoRefill(funcParams, int(num), doneChan)
		counter.Add(num)
	}
}

func (p *Pool[T, P]) runWithAutoRefill(params []P, size int, finishedChan chan struct{}) {
	doneChan := make(chan struct{})

	for i := 0; i < size; i++ {

		i := i

		go func(p *Pool[T, P], doneChan chan struct{}) {
			defer func() {
				doneChan <- struct{}{}
			}()

			p.dataSources.Lock()

			if len(p.dataSources.data) == 0 {
				return
			}

			if i >= len(p.dataSources.data) {
				i = len(p.dataSources.data) - 1
			}

			ds := p.dataSources.data[i]
			p.dataSources.Unlock()

			p.removeDataSourceByIndex(i)

			p.workerFunc(ds, params, p.errChan)

			if finishedChan != nil {
				finishedChan <- struct{}{}
			}

		}(p, doneChan)
	}

	wg := &sync.WaitGroup{}
	wg.Add(1)

	go func(p *Pool[T, P], doneChan chan struct{}, params []P) {
		defer wg.Done()
		for {
			select {
			case <-doneChan:
				var d []T

				p.dataSources.Lock()
				d = p.dataSources.data
				p.dataSources.Unlock()
				if len(d) != 0 {
					p.spinUpGoroutine(doneChan, params)
				}
			}
		}
	}(p, doneChan, params)

	go func(wg *sync.WaitGroup, doneChan chan struct{}) {
		wg.Wait()
		close(doneChan)
	}(wg, doneChan)
}

func (p *Pool[T, P]) tryLock(d *[]T) {
	time.Sleep(time.Millisecond)

	locked := p.dataSources.TryLock()

	if locked {
		d = &p.dataSources.data
		return
	}
	p.tryLock(d)
}

func (p *Pool[T, P]) spinUpGoroutine(doneChan chan struct{}, funcParams []P) {
	p.dataSources.Lock()
	ds := p.dataSources.data[0]
	p.dataSources.Unlock()
	p.removeDataSourceByIndex(0)

	go func(p *Pool[T, P], doneChan chan struct{}) {
		defer func() {
			doneChan <- struct{}{}
		}()
		p.workerFunc(ds, funcParams, p.errChan)
	}(p, doneChan)
}

func (p *Pool[T, P]) removeDataSourceByIndex(s int) {

	p.dataSources.Lock()
	defer p.dataSources.Unlock()
	p.dataSources.data = append(p.dataSources.data[:s], p.dataSources.data[s+1:]...)
}

func (p *Pool[T, P]) checkContinueBatch(remainingSlice []T, funcParams []P) {
	if len(remainingSlice) != 0 {
		p.dataSources.data = remainingSlice
		p.Start(funcParams)
		return
	}
}

package gworker

import (
	"context"
	"errors"
	"sync"
	"time"
)

// Pool is a Generic worker pool implementation
type Pool[T any, P any] struct {
	size        int
	batched     bool
	dataSources dS[T]
	workerFunc  func(dataSource T, params []P, errChan chan error)
	errChan     chan error
	ctx         context.Context
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

// Size determines the max number of concurrent workers to run at once whenever the Pool has
// been started. If Size is not called, a Pool will default to a size of 5
func (p *Pool[T, P]) Size(size int) *Pool[T, P] {
	p.size = size
	return p
}

// WithAutoPoolRefill switches the Pool from its default "Batched" state (where the Pool
// loops through the data provided to it and runs them in batches matching the Size provided
// until all the dataSources are exhausted) to an "AutoPoolRefill" state (where, as soon as one
// worker goroutine finishes, if any dataSources remain, another worker goroutine will immediately
// be spun up until all dataSources have been exhausted)
func (p *Pool[T, P]) WithAutoPoolRefill() *Pool[T, P] {
	p.batched = false
	return p
}

// WithCancel allows you to provide a Pool with a context and be returned a context.CancelFunc
// that can be called to prematurely terminated a started Pool
func (p *Pool[T, P]) WithCancel(ctx context.Context) (*Pool[T, P], context.CancelFunc) {
	c, cancel := context.WithCancel(ctx)

	p.ctx = c

	return p, cancel
}

// Start starts running the Pool's workers and injects any provided channels, along with the
// provided funcParams, to each worker goroutine
func (p *Pool[T, P]) Start(funcParams []P) {
	dataLen := len(p.dataSources.data)

	if p.size > dataLen {
		p.size = dataLen
	}

	if p.batched {
		batchSlice := p.dataSources.data[:p.size]
		remainingSlice := p.dataSources.data[p.size:]

		wg := &sync.WaitGroup{}

		for i := 0; i < len(batchSlice); i++ {

			wg.Add(1)

			i := i

			if p.ctx != nil {
				go func(p *Pool[T, P]) {
					defer wg.Done()
					for {
						select {
						case <-p.ctx.Done():
							return
						default:
							p.workerFunc(p.dataSources.data[i], funcParams, p.errChan)
						}
					}
				}(p)
			} else {
				go func(p *Pool[T, P]) {
					defer wg.Done()

					p.workerFunc(p.dataSources.data[i], funcParams, p.errChan)
				}(p)
			}
		}

		wg.Wait()

		p.checkContinueBatch(remainingSlice, funcParams)
	} else {
		if p.ctx != nil {
			for {
				select {
				case <-p.ctx.Done():
					return
				default:
					p.runWithAutoRefill(funcParams)
				}
			}
		} else {
			p.runWithAutoRefill(funcParams)
		}

	}
}

func (p *Pool[T, P]) runWithAutoRefill(params []P) {
	doneChan := make(chan struct{})

	for i := 0; i < p.size; i++ {

		i := i

		go func(p *Pool[T, P], doneChan chan struct{}) {
			defer func() {
				doneChan <- struct{}{}
			}()

			p.dataSources.Lock()
			ds := p.dataSources.data[i]
			p.dataSources.Unlock()

			p.removeDataSourceByIndex(i)

			p.workerFunc(ds, params, p.errChan)
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
				if p.dataSources.TryLock() {
					d = p.dataSources.data
				} else {
					p.tryLock(&d)
				}
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
	}
}

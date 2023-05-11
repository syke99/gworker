package gworker

import (
	"context"
	"errors"
	"sync"
)

// Pool is a Generic worker pool implementation
type Pool[T any] struct {
	size          int
	batched       bool
	dataSources   dS[T]
	workerFunc    func(dataSource any, params []any)
	valueChannels []chan any
	errorChannel  chan error
	ctx           context.Context
}

type dS[T any] struct {
	sync.Mutex
	data []T
}

var missingWorkerFuncError = errors.New("no worker func provided")

// NewPool initializes a new Pool with the provided dataSources, worker func, any value and error channels, then returns it;
// to prevent blocking and allow dataSources whose length is greater than the provided (or default) Size,
// any channels used must be buffered channels
func NewPool[T any](dataSources []T, worker func(dataSource any, params []any), valueChannels []chan any, errorChannel chan error) (*Pool[T], error) {
	if worker == nil {
		return nil, missingWorkerFuncError
	}
	p := Pool[T]{
		batched: true,
		dataSources: dS[T]{
			data: dataSources,
		},
		valueChannels: valueChannels,
		errorChannel:  errorChannel,
		workerFunc:    worker,
	}
	return &p, nil
}

// Size determines the max number of concurrent workers to run at once whenever the Pool has
// been started. If Size is not called, a Pool will default to a size of 5
func (p *Pool[T]) Size(size int) *Pool[T] {
	p.size = size
	return p
}

// WithAutoPoolRefill switches the Pool from its default "Batched" state (where the Pool
// loops through the data provided to it and runs them in batches matching the Size provided
// until all the dataSources are exhausted) to an "AutoPoolRefill" state (where, as soon as one
// worker goroutine finishes, if any dataSources remain, another worker goroutine will immediately
// be spun up until all dataSources have been exhausted)
func (p *Pool[T]) WithAutoPoolRefill() *Pool[T] {
	p.batched = false
	return p
}

// WithCancel allows you to provide a Pool with a context and be returned a context.CancelFunc
// that can be called to prematurely terminated a started Pool
func (p *Pool[T]) WithCancel(ctx context.Context) (*Pool[T], context.CancelFunc) {
	c, cancel := context.WithCancel(ctx)

	p.ctx = c

	return p, cancel
}

// Start starts running the Pool's workers and injects any provided channels, along with the
// provided funcParams, to each worker goroutine
func (p *Pool[T]) Start(funcParams []any) {
	if p.size == 0 {
		p.size = 5
	}

	params := make([]any, len(p.valueChannels)+len(p.errorChannel)+len(funcParams))

	if p.valueChannels != nil {
		for i, vChan := range p.valueChannels {
			params[i] = vChan
		}
	}

	if p.errorChannel != nil {
		if p.valueChannels != nil {
			params[len(p.valueChannels)+1] = p.errorChannel
		}
	}

	for i, param := range funcParams {
		i = i + len(p.valueChannels) + len(p.errorChannel)
		params[i] = param
	}

	if p.batched {
		batchSlice := p.dataSources.data[:p.size]
		remainingSlice := p.dataSources.data[p.size:]

		wg := &sync.WaitGroup{}

		for i := 0; i < len(batchSlice); i++ {

			wg.Add(1)

			i := i

			if p.ctx != nil {
				go func(p *Pool[T]) {
					defer wg.Done()
					for {
						select {
						case <-p.ctx.Done():
							return
						default:
							p.workerFunc(p.dataSources.data[i], params)
						}
					}
				}(p)
			} else {
				go func(p *Pool[T]) {
					defer wg.Done()

					p.workerFunc(p.dataSources.data[i], params)
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
					p.runWithAutoRefill(params)
				}
			}
		} else {
			p.runWithAutoRefill(params)
		}

	}
}

func (p *Pool[T]) runWithAutoRefill(params []any) {
	doneChan := make(chan struct{})

	for i := 0; i < p.size; i++ {
		go func(p *Pool[T], doneChan chan struct{}) {
			defer func() {
				doneChan <- struct{}{}
			}()

			p.dataSources.Lock()
			ds := p.dataSources.data[i]
			p.dataSources.Unlock()

			p.removeDataSourceByIndex(i)

			p.workerFunc(ds, params)
		}(p, doneChan)
	}

	wg := &sync.WaitGroup{}
	wg.Add(1)

	go func(p *Pool[T], doneChan chan struct{}, params []any) {
		defer wg.Done()
		for {
			select {
			case <-doneChan:
				if len(p.dataSources.data) != 0 {
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

func (p *Pool[T]) spinUpGoroutine(doneChan chan struct{}, funcParams []any) {
	p.dataSources.Lock()
	ds := p.dataSources.data[0]
	p.dataSources.Unlock()
	p.removeDataSourceByIndex(0)

	go func(p *Pool[T], doneChan chan struct{}) {
		defer func() {
			doneChan <- struct{}{}
		}()
		p.workerFunc(ds, funcParams)
	}(p, doneChan)
}

func (p *Pool[T]) removeDataSourceByIndex(s int) {

	p.dataSources.Lock()
	defer p.dataSources.Unlock()
	p.dataSources.data = append(p.dataSources.data[:s], p.dataSources.data[s+1:]...)
}

func (p *Pool[T]) checkContinueBatch(remainingSlice []T, funcParams []any) {
	if len(remainingSlice) != 0 {
		p.dataSources.data = remainingSlice
		p.Start(funcParams)
	}
}

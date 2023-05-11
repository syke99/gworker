package gworker

import (
	"context"
	"sync"
)

type Pool[T any] struct {
	size          int
	batched       bool
	dataSources   dataSources[T]
	workerFunc    func(dataSource any, params []any)
	valueChannels []chan any
	errorChannel  chan error
	funcParams    []any
	ctx           context.Context
}

type dataSources[T any] struct {
	sync.Mutex
	data []T
}

func NewPool[T any](data []T, worker func(dataSource any, params []any), valueChannels []chan any, errorChannel chan error, funcParams []any) (*Pool[T], error) {
	p := Pool[T]{
		batched: true,
		dataSources: dataSources[T]{
			data: data,
		},
		valueChannels: valueChannels,
		errorChannel:  errorChannel,
		workerFunc:    worker,
		funcParams:    funcParams,
	}
	return &p, nil
}

func (p *Pool[T]) Size(size int) *Pool[T] {
	p.size = size
	return p
}

func (p *Pool[T]) WithAutoPoolRefill() *Pool[T] {
	p.batched = false
	return p
}

func (p *Pool[T]) WithCancel(ctx context.Context) (*Pool[T], context.CancelFunc) {
	c, cancel := context.WithCancel(ctx)

	p.ctx = c

	return p, cancel
}

func (p *Pool[T]) Start() {
	if p.batched {
		batchSlice := p.dataSources.data[:p.size]
		remainingSlice := p.dataSources.data[p.size:]

		params := make([]any, len(p.valueChannels)+len(p.errorChannel)+len(p.funcParams))

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

		for i, param := range p.funcParams {
			i = i + len(p.valueChannels) + len(p.errorChannel)
			params[i] = param
		}

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

		p.checkContinueBatch(remainingSlice)
	}
}

func (p *Pool[T]) checkContinueBatch(remainingSlice []T) {
	if len(remainingSlice) != 0 {
		p.dataSources.data = remainingSlice
		p.Start()
	}
}

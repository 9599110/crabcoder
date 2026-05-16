package scheduler

import (
	"context"
	"sync"

	"github.com/crabcoder/crabcoder/pkg/model"
)

type TaskExecutor func(ctx context.Context, task *model.Task) *model.TaskResult

type Pool struct {
	workers   int
	executor  TaskExecutor
	taskCh    chan *model.Task
	resultCh  chan taskResult
	wg        sync.WaitGroup
}

type taskResult struct {
	TaskID string
	Result *model.TaskResult
}

func NewPool(workers int, executor TaskExecutor) *Pool {
	return &Pool{
		workers:  workers,
		executor: executor,
		taskCh:   make(chan *model.Task, workers*2),
		resultCh: make(chan taskResult, workers*2),
	}
}

func (p *Pool) Start(ctx context.Context) {
	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go func() {
			defer p.wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case task, ok := <-p.taskCh:
					if !ok {
						return
					}
					result := p.executor(ctx, task)
					select {
					case p.resultCh <- taskResult{TaskID: task.ID, Result: result}:
					case <-ctx.Done():
						return
					}
				}
			}
		}()
	}
}

func (p *Pool) Submit(task *model.Task) {
	p.taskCh <- task
}

func (p *Pool) Results() <-chan taskResult {
	return p.resultCh
}

func (p *Pool) Wait() {
	p.wg.Wait()
	close(p.resultCh)
}

func (p *Pool) Close() {
	close(p.taskCh)
}

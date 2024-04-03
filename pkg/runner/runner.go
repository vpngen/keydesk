package runner

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

type Task struct {
	Func     func(ctx context.Context) error
	Shutdown func(ctx context.Context) error
	ctx      context.Context
	cancel   context.CancelFunc
}

type Runner struct {
	ctx    context.Context
	cancel context.CancelFunc
	tasks  map[string]Task
	errCh  chan error
}

func New(ctx context.Context) *Runner {
	r := &Runner{
		tasks: make(map[string]Task),
	}
	r.ctx, r.cancel = context.WithCancel(ctx)
	return r
}

func (r *Runner) AddTask(name string, task Task) {
	task.ctx, task.cancel = context.WithCancel(r.ctx)
	r.tasks[name] = task
}

func (r *Runner) Run() <-chan error {
	r.errCh = make(chan error, len(r.tasks))
	wg := new(sync.WaitGroup)
	//wg.Add(len(r.tasks))

	for name, task := range r.tasks {
		go func(n string, t Task) {
			fmt.Println("starting", n)
			if err := t.Func(t.ctx); err != nil {
				fmt.Println(n, "error:", err)
			}
		}(name, task)

		if task.Shutdown != nil {
			wg.Add(1)
			go func(n string, t Task) {
				<-t.ctx.Done()
				fmt.Println("shutting down", n)
				if err := t.Shutdown(t.ctx); err != nil && !errors.Is(err, context.Canceled) {
					r.errCh <- fmt.Errorf("%s: shutdown error: %w", n, err)
				}
				wg.Done()
			}(name, task)
		}
	}

	go func() {
		wg.Wait()
		close(r.errCh)
	}()

	return r.errCh
}

func (r *Runner) Stop() (err error) {
	r.cancel()
	for e := range r.errCh {
		err = errors.Join(err, e)
	}
	return
}

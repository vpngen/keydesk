package runner

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestRunner(t *testing.T) {
	r := New(context.Background())

	for i := 0; i < 10; i++ {
		r.AddTask(fmt.Sprintf("task-%d", i), Task{
			Func: func(ctx context.Context) error {
				select {
				case <-ctx.Done():
					return ctx.Err()
				}
			},
		})
	}

	r.Run()

	time.Sleep(3 * time.Second)

	err := r.Stop()
	t.Log(err)
}

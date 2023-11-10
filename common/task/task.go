package task

import (
	"context"

	"github.com/xtls/xray-core/common/signal/semaphore"
)

// OnSuccess executes g() after f() returns nil.
/**
 `OnSuccess`: 这是一个高阶函数（higher-order function），接收两个函数作为参数：`f` 和 `g`。它返回一个函数，该函数会在 `f` 返回 `nil` 时执行 `g`。具体步骤如下：
   - 调用 `f` 函数并检查返回的错误。
   - 如果错误不为 `nil`，直接返回该错误。
   - 如果错误为 `nil`，调用 `g` 函数并返回其结果。
**/
func OnSuccess(f func() error, g func() error) func() error {
	return func() error {
		if err := f(); err != nil {
			return err
		}
		return g()
	}
}

// Run executes a list of tasks in parallel, returns the first error encountered or nil if all tasks pass.
// Run 并行执行任务列表，返回遇到的第一个错误，如果所有任务都通过，则返回 nil。
/**
 `Run`: 这是一个并行执行任务的函数。它接收一个 `context.Context` 对象以及一系列的任务函数（每个任务函数都接收并返回一个 `error`）。函数的执行过程如下：
   - 根据任务数量创建一个信号量（semaphore）对象，初始计数为任务数量。
   - 创建一个用于接收完成任务的通道 `done`。
**/
func Run(ctx context.Context, tasks ...func() error) error {
	n := len(tasks)
	s := semaphore.New(n)
	done := make(chan error, 1)

	/**
	 	- 遍历任务列表，为每个任务启动一个 goroutine 进行异步执行。
	       - 在开始执行任务之前，通过信号量等待一个可用信号量，防止同时执行的任务数量超过设定值。
		   - 执行任务函数。如果任务函数返回的错误为 `nil`，释放一个信号量；否则将错误发送到 `done` 通道中。
		**/
	for _, task := range tasks {
		<-s.Wait()
		go func(f func() error) {
			err := f()
			if err == nil {
				s.Signal() // 释放信号
				return
			}

			select {
			case done <- err:
			default:
			}
		}(task)
	}

	/**
	- 使用 `select` 语句监听三个可能事件：
		- 从 `done` 通道接收到错误：直接返回该错误，表示遇到了第一个出错的任务。
		- `ctx.Done()` 被触发：返回 `ctx.Err()`，表示任务被取消。
		- 从信号量等待通道中接收到信号量：表示有任务已经完成，不进行处理。
	**/
	for i := 0; i < n; i++ {
		select {
		case err := <-done:
			return err
		case <-ctx.Done():
			return ctx.Err()
		case <-s.Wait():
		}
	}

	return nil
}

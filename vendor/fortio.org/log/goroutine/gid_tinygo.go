//go:build tinygo

package goroutine

import (
	"sync"
	"unsafe"
)

const IsTinyGo = true

var (
	counter int64
	mapping = make(map[uintptr]int64)
	// TinyGo at the moment is single threaded, so this is not needed, but it's good to have anyway
	// in case that changes. It does add ~5ns (from 20ns vs 4ns big go) but it's better to be correct.
	// In theory, the mutex could be noop on platforms where everything is single threaded.
	lock sync.Mutex
)

func ID() int64 {
	task := uintptr(currentTask())
	lock.Lock() // explicit minimal critical section without using defer, on purpose.
	if id, ok := mapping[task]; ok {
		lock.Unlock()
		return id
	}
	counter++
	mapping[task] = counter
	lock.Unlock()
	return counter
	// or, super fast but ugly large numbers/pointers:
	//return int64(uintptr(currentTask()))
}

// Call https://github.com/tinygo-org/tinygo/blob/v0.32.0/src/internal/task/task_stack.go#L39
//
//go:linkname currentTask internal/task.Current
func currentTask() unsafe.Pointer

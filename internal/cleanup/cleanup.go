package cleanup

import "sync"

var (
	registered []func()
	mu         sync.Mutex
)

func Register(fn func()) {
	mu.Lock()
	defer mu.Unlock()
	registered = append(registered, fn)
}

func Cleanup() {
	mu.Lock()
	defer mu.Unlock()
	for _, fn := range registered {
		fn()
	}
}

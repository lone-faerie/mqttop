package syncutil

import (
	"sync"
	"sync/atomic"
)

// Once is equivalent to [sync.Once] with the ability to be reset.
type Once struct {
	_ noCopy

	done atomic.Uint32
	m    sync.Mutex
}

// Do calls the function f if and only if Do is being called for the
// first time for this instance of [Once].
func (o *Once) Do(f func()) {
	if o.done.Load() == 0 {
		o.doSlow(f)
	}
}

func (o *Once) doSlow(f func()) {
	o.m.Lock()
	defer o.m.Unlock()

	if o.done.Load() == 0 {
		defer o.done.Store(1)

		f()
	}
}

// Reset resets o to be reused.
func (o *Once) Reset() bool {
	o.m.Lock()
	defer o.m.Unlock()

	return o.done.CompareAndSwap(1, 0)
}

// OnceValue is the struct equivalent of [sync.OnceValue] with the ability to be reset.
type OnceValue[T any] struct {
	Once
	valid  bool
	p      any
	result T
}

func (o *OnceValue[T]) Do(f func() T) T {
	o.Once.Do(func() {
		defer func() {
			o.p = recover()
			if !o.valid {
				panic(o.p)
			}
		}()

		o.result = f()
		f = nil
		o.valid = true
	})

	if !o.valid {
		panic(o.p)
	}

	return o.result
}

// OnceValues is the struct equivalent of [sync.OnceValues] with the ability to be reset.
type OnceValues[T1, T2 any] struct {
	Once
	valid bool
	p     any
	r1    T1
	r2    T2
}

func (o *OnceValues[T1, T2]) Do(f func() (T1, T2)) (T1, T2) {
	o.Once.Do(func() {
		defer func() {
			o.p = recover()

			if !o.valid {
				panic(o.p)
			}
		}()

		o.r1, o.r2 = f()
		f = nil
		o.valid = true
	})

	if !o.valid {
		panic(o.p)
	}

	return o.r1, o.r2
}

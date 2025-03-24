package syncutil

type noCopy struct{}

func (*noCopy) Lock()   {}
func (*noCopy) Unlock() {}

type Once[T any] struct {
	_    noCopy
	done atomic.Uint32
	m    sync.Mutex
	ret  T
}

func (o *Once[T]) Do(f func() T) T {
	if o.done.Load() == 0 {
		o.doSlow(f)
	}
	return o.ret
}

func (o *Once[T]) doSlow(f func() T) {
	o.m.Lock()
	defer o.m.Unlock()
	if o.done.Load() == 0 {
		defer o.done.Store(1)
		o.ret = f()
	}
}

func (o *Once) Reset() {
	o.m.Lock()
	defer o.m.Unlock()
	o.done.Store(0)
}

package syncutil

type Pool[T any] struct {
	pool sync.Pool
	New  func() T
}

func (p *Pool) Get() T {
	return p.pool.Get().(T)
}

func (p *Pool) Put(t T) {
	p.pool.Put(t)
}

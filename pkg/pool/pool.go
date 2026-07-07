package pool

import "sync"

// Resetter is a constraint for types that can reset their state.
type Resetter interface {
	Reset()
}

// Pool is a generic wrapper around sync.Pool for types with a Reset() method.
// On Put the object is automatically reset before being returned to the pool.
type Pool[T Resetter] struct {
	p    sync.Pool
	newF func() T
}

// New creates a Pool. The constructor f is called when the pool is empty.
func New[T Resetter](f func() T) *Pool[T] {
	p := &Pool[T]{newF: f}
	p.p.New = func() any { return f() }
	return p
}

// Get returns an object from the pool or creates a new one.
func (p *Pool[T]) Get() T {
	return p.p.Get().(T)
}

// Put resets the object and returns it to the pool.
func (p *Pool[T]) Put(v T) {
	v.Reset()
	p.p.Put(v)
}

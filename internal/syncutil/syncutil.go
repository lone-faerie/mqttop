// Package syncutil provides synchronization primitives.
//
// Values containing the types defined in this package should not be copied.
package syncutil

type noCopy struct{}

func (*noCopy) Lock()   {}
func (*noCopy) Unlock() {}

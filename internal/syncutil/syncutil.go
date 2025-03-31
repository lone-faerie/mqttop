package syncutil

type noCopy struct{}

func (*noCopy) Lock()   {}
func (*noCopy) Unlock() {}

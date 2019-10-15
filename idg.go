package idg

import "runtime"

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}

// IDG present an internet download manager
// TODO: Implement a feature to support downloading many files in concurrent
type IDG struct {
	MaxWorkers int
}

// Download downloads a file
func (idg *IDG) Download() (*File, error) {
	return nil, nil
}

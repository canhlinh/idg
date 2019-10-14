package idg

import "runtime"

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}

// IDG present an internet download manager
type IDG struct {
	MaxWorkers int
}

// Download downloads a file
func (idg *IDG) Download() (*File, error) {
	return nil, nil
}

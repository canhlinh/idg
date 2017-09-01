package idg

import "testing"
import "github.com/stretchr/testify/assert"

func TestNewWorker(t *testing.T) {
	workerPool := make(chan *Worker, MaxQueueItems)
	w := NewWorker(workerPool)
	assert.NotNil(t, w.WorkerPool)
	assert.NotNil(t, w.JobChannel)
	assert.NotNil(t, w.quit)
}

func TestStartWorker(t *testing.T) {
	workerPool := make(chan *Worker, MaxQueueItems)
	w := NewWorker(workerPool)
	w.start()

	var done bool
	j := func() error {
		done = true
		return nil
	}

	w.JobChannel <- j
	assert.True(t, done)
	w.quit <- true
}

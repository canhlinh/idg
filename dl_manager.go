package idg

const (
	MaxDownloadItems = 2
	MaxQueueItems    = 1000
)

type Downloader interface {
	Start()
	Stop()
	Add(file *File)
	Remove(file *File)
}

type Job func() error

type Worker struct {
	WorkerPool chan *Worker
	JobChannel chan Job
	quit       chan bool
}

type DlManagement struct {
	maxWorkers int
	workerPool chan *Worker
	JobQueue   chan Job
	quit       chan bool
	Workers    []*Worker
}

func NewWorker(pool chan *Worker) *Worker {
	return &Worker{
		WorkerPool: pool,
		JobChannel: make(chan Job),
		quit:       make(chan bool),
	}
}

func (w *Worker) start() {
	go func() {
		for {
			w.WorkerPool <- w
			select {
			case job := <-w.JobChannel:
				job()
			case <-w.quit:
				w.quit <- true
				return
			}
		}
	}()
}

func NewDowloader() Downloader {
	d := &DlManagement{
		maxWorkers: MaxDownloadItems,
		workerPool: make(chan *Worker, MaxQueueItems),
		quit:       make(chan bool),
		JobQueue:   make(chan Job),
	}

	for i := 0; i < d.maxWorkers; i++ {
		worker := NewWorker(d.workerPool)
		d.Workers = append(d.Workers, worker)
	}

	return &DlManagement{}
}

func (d *DlManagement) Start() {

	for _, worker := range d.Workers {
		worker.start()
	}

	go func() {
		for {
			select {
			case job := <-d.JobQueue:
				worker := <-d.workerPool
				worker.JobChannel <- job
			case <-d.quit:
				for i := 0; i < d.maxWorkers; i++ {
					worker := <-d.workerPool
					worker.quit <- true
					<-worker.quit
				}
				d.quit <- true
				return
			}
		}
	}()
}

func (d *DlManagement) Stop() {
	d.quit <- true
}

func (d *DlManagement) Add(file *File) {
	d.JobQueue <- file.StartDownload
}

func (d *DlManagement) Remove(file *File) {
}

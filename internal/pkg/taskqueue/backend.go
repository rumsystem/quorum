package taskqueue

import "fmt"

// taskqueue struct
type Taskqueue struct {
	dispatcher *dispatcher
	queue      *queue
}

// New creates new Blero Backend
func New(dbPath string) *Taskqueue {
	tq := &Taskqueue{}
	pStore := newProcessorsStore()
	tq.dispatcher = newDispatcher(pStore)
	tq.queue = newQueue(queueOpts{DBPath: dbPath})
	return tq
}

// Start Blero
func (tq *Taskqueue) Start() error {
	fmt.Println("Starting Blero ...")
	err := tq.queue.start()
	if err != nil {
		return err
	}
	tq.dispatcher.startLoop(tq.queue)
	return nil
}

// Stop Blero and Release resources
func (tq *Taskqueue) Stop() error {
	fmt.Println("Stopping Blero ...")
	tq.dispatcher.stopLoop()
	return tq.queue.stop()
}

// EnqueueJob enqueues a new Job and returns the job id
func (tq *Taskqueue) EnqueueJob(name string, data []byte) (uint64, error) {
	jID, err := tq.queue.enqueueJob(name, data)
	if err != nil {
		return 0, err
	}

	// signal that a new job was enqueued
	tq.dispatcher.signalLoop()

	return jID, nil
}

// EnqueueJobs enqueues new Jobs
/*func (bl *Blero) EnqueueJobs(names string) (uint64, error) {
}*/

// RegisterProcessor registers a new processor and returns the processor id
func (tq *Taskqueue) RegisterProcessor(p Processor) int {
	return tq.dispatcher.registerProcessor(p)
}

// RegisterProcessorFunc registers a new ProcessorFunc and returns the processor id
func (tq *Taskqueue) RegisterProcessorFunc(f func(j *Job) error) int {
	return tq.dispatcher.registerProcessor(ProcessorFunc(f))
}

// UnregisterProcessor unregisters a processor
// No more jobs will be assigned but if will not cancel a job that already started processing
func (tq *Taskqueue) UnregisterProcessor(pID int) {
	tq.dispatcher.unregisterProcessor(pID)
}

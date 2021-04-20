package taskqueue

import (
	"fmt"
	"io"
	"os"
	"sync"
)

// dispatcher struct
type dispatcher struct {
	dispatchL sync.Mutex
	ch        chan int
	quitCh    chan struct{}
	pStore    *processorsStore
}

// newDispatcher creates new Dispatcher
func newDispatcher(pStore *processorsStore) *dispatcher {
	d := &dispatcher{}
	d.ch = make(chan int, 100)
	d.quitCh = make(chan struct{})
	d.pStore = pStore
	return d
}

// testable stderr
var stdErr io.ReadWriter = os.Stderr

// startLoop starts the dispatcher assignment loop
func (d *dispatcher) startLoop(q *queue) {
	go func() {
		for {
			select {
			case <-d.ch:
				err := d.assignJobs(q)
				if err != nil {
					fmt.Fprintf(stdErr, "Cannot assign jobs: %v", err)
				}
			case <-d.quitCh: // loop was stopped
				return
			}
		}
	}()
}

// signalLoop signals to the dispatcher loop that an assignment check might need to run
func (d *dispatcher) signalLoop() {
	go func() {
		select {
		case <-d.quitCh: // loop was stopped
			return
		case d.ch <- 1:
			return
		}
	}()
}

// stopLoop stops the dispatcher assignment loop
func (d *dispatcher) stopLoop() {
	close(d.quitCh)
}

// registerProcessor registers a new processor
func (d *dispatcher) registerProcessor(p Processor) int {
	d.dispatchL.Lock()
	defer d.dispatchL.Unlock()

	pID := d.pStore.registerProcessor(p)

	// signal that the processor is now available
	d.signalLoop()

	return pID
}

// unregisterProcessor unregisters a processor
// No more jobs will be assigned but if will not cancel a job that already started processing
func (d *dispatcher) unregisterProcessor(pID int) {
	d.dispatchL.Lock()
	defer d.dispatchL.Unlock()

	d.pStore.unregisterProcessor(pID)
}

// assignJobs assigns pending jobs from the queue to free processors
func (d *dispatcher) assignJobs(q *queue) error {
	d.dispatchL.Lock()
	defer d.dispatchL.Unlock()

	pIDs := d.pStore.getAvailableProcessorsIDs()

	for _, pID := range pIDs {
		err := d.assignJob(q, pID)
		if err != nil {
			return err
		}
	}

	return nil
}

// assignJob assigns a pending job processor #pID and starts the run
// NOT THREAD SAFE !! only call from assignJobs
func (d *dispatcher) assignJob(q *queue, pID int) error {
	p := d.pStore.getProcessor(pID)
	if p == nil {
		return fmt.Errorf("Processor %v not found", pID)
	}

	j, err := q.dequeueJob()
	if err != nil {
		return err
	}
	// no jobs to assign
	if j == nil {
		return nil
	}

	if d.pStore.isProcessorBusy(pID) {
		return fmt.Errorf("Cannot assign job %v to Processor %v. Processor busy", j.ID, pID)
	}

	d.pStore.setProcessing(pID, j.ID)

	go d.runJob(q, pID, p, j)

	return nil
}

// unassignJob unmarks a job as assigned to #pID
func (d *dispatcher) unassignJob(pID int) {
	d.dispatchL.Lock()
	defer d.dispatchL.Unlock()

	d.pStore.unsetProcessing(pID)
}

// runJob runs a job on the corresponding processor and moves it to the right queue depending on results
func (d *dispatcher) runJob(q *queue, pID int, p Processor, j *Job) {
	defer d.processorDone(pID)
	err := p.Run(j)
	if err != nil {
		fmt.Printf("Processor: %v. Job %v failed with err: %v\n", pID, j.ID, err)
		err := q.markJobDone(j.ID, jobFailed)
		if err != nil {
			fmt.Printf("markJobDone -> %v jobFailed failed: %v\n", j.ID, err)
		}
		return
	}

	err = q.markJobDone(j.ID, jobComplete)
	if err != nil {
		fmt.Printf("markJobDone -> %v jobComplete failed: %v\n", j.ID, err)
	}
}

func (d *dispatcher) processorDone(pID int) {
	d.unassignJob(pID)

	// signal that the processor might now be available
	d.signalLoop()
}

package taskqueue

// Job represents a Goblero job definition
type Job struct {
	ID   uint64
	Name string
	Data []byte
}

// Processor interface
type Processor interface {
	Run(j *Job) error
}

// ProcessorFunc is a processor function
type ProcessorFunc func(j *Job) error

// Run allows using ProcessorFunc as a Processor
func (pf ProcessorFunc) Run(j *Job) error {
	return pf(j)
}

// processorsStore struct
type processorsStore struct {
	maxProcessorID int
	processors     map[int]Processor
	processing     map[int]uint64
}

// newProcessorsStore creates a new ProcessorsStore
func newProcessorsStore() *processorsStore {
	pStore := &processorsStore{}
	pStore.processors = make(map[int]Processor)
	pStore.processing = make(map[int]uint64)
	return pStore
}

// registerProcessor registers a new processor
func (pStore *processorsStore) registerProcessor(p Processor) int {
	pStore.maxProcessorID++
	pStore.processors[pStore.maxProcessorID] = p

	return pStore.maxProcessorID
}

// unregisterProcessor unregisters a processor
func (pStore *processorsStore) unregisterProcessor(pID int) {
	delete(pStore.processors, pID)
}

// getAvailableProcessorsIDs returns the currently free processors
func (pStore *processorsStore) getAvailableProcessorsIDs() []int {
	var pIDs []int
	for pID := range pStore.processors {
		if _, ok := pStore.processing[pID]; !ok {
			pIDs = append(pIDs, pID)
		}
	}
	return pIDs
}

// getProcessor fetches processors by ID
func (pStore *processorsStore) getProcessor(pID int) Processor {
	return pStore.processors[pID]
}

// isProcessorBusy checks if a processor is already working on a job
func (pStore *processorsStore) isProcessorBusy(pID int) bool {
	_, ok := pStore.processing[pID]
	return ok
}

// setProcessing sets a processor as working on a job
func (pStore *processorsStore) setProcessing(pID int, jID uint64) {
	pStore.processing[pID] = jID
}

// unsetProcessing unsets a processor as working on a job
func (pStore *processorsStore) unsetProcessing(pID int) {
	delete(pStore.processing, pID)
}

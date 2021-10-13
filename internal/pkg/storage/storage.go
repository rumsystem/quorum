package storage

type QuorumStorage interface {
	Init(path string) error
	Close() error
	Set(key []byte, val []byte) error
	Delete(key []byte) error
	Get(key []byte) ([]byte, error)
	PrefixForeach(prefix []byte, fn func([]byte, []byte, error) error) error
	PrefixForeachKey(prefix []byte, valid []byte, reverse bool, fn func([]byte, error) error) error
	Foreach(fn func([]byte, []byte, error) error) error

	// For appdb, atomic batch write
	BatchWrite(keys [][]byte, values [][]byte) error
	GetSequence([]byte, uint64) (Sequence, error)
}

type Sequence interface {
	Next() (uint64, error)
	Release() error
}

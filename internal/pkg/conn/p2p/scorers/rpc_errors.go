package scorers

import "errors"

var (
	ErrWrongForkDigestVersion = errors.New("wrong fork digest version")
	ErrInvalidEpoch           = errors.New("invalid epoch")
	ErrInvalidFinalizedRoot   = errors.New("invalid finalized root")
	ErrInvalidSequenceNum     = errors.New("invalid sequence number provided")
	ErrGeneric                = errors.New("internal service error")
	ErrInvalidParent          = errors.New("mismatched parent root")
	ErrRateLimited            = errors.New("rate limited")
	ErrIODeadline             = errors.New("i/o deadline exceeded")
	ErrInvalidRequest         = errors.New("invalid range, step or count")
)

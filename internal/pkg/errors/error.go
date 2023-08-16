package errors

import "errors"

var (
	// kv db
	ErrEmptyKey    = errors.New("key cannot be empty")
	ErrKeyNotFound = errors.New("key not found")

	ErrInvalidGroupID   = errors.New("Invalid group id")
	ErrGroupNotFound    = errors.New("Group not found")
	ErrJoinGroup        = errors.New("Join group failed")
	ErrClearJoinedGroup = errors.New("Can not clear joined group")
	ErrInvalidGroupData = errors.New("Invalid group data")
	ErrOnlyGroupOwner   = errors.New("Only group owner can do this")
	ErrGroupNotPrivate  = errors.New("Only private group can do this")

	ErrInvalidBlockID  = errors.New("Invalid block id")
	ErrBlockIDNotFound = errors.New("Block id not found")
	ErrBlockExist      = errors.New("Block aleady exist")

	ErrInvalidTrxID     = errors.New("Invalid trx id")
	ErrInvalidTrxIDList = errors.New("Invalid trx id list")
	ErrInvalidTrxData   = errors.New("Invalid trx data")
	ErrTrxIdNotFound    = errors.New("Trx id not found")

	ErrPrivateGroupNotSupported   = errors.New("Private group is not supported")
	ErrEncryptionTypeNotSupported = errors.New("Encryption type is not supported")
	ErrConsensusTypeNotSupported  = errors.New("Consensus type is not supported")

	ErrOpenKeystore         = errors.New("Open keystore failed")
	ErrGetSignPubKey        = errors.New("Get sign public key failed")
	ErrInvalidSignPubKey    = errors.New("Invalid sign public key")
	ErrEncryptAliasNotFound = errors.New("Encrypt alias not found")
	ErrSignAliasNotFound    = errors.New("Sign alias not found")
	ErrInvalidAliasType     = errors.New("Invalid alias type")

	ErrInvalidChainAPIURL = errors.New("Invalid chain api url")

	ErrInvalidJWT = errors.New("Invalid JWT")

	ErrNoPeersAvailable = errors.New("no peers available, waiting for reconnect")

	//syncer
	ErrNotAskedByMe   = errors.New("Error Get Sync Resp but not asked by me")
	ErrSenderMismatch = errors.New("Trx Sender/blocks provider mismatch")
	ErrNoTaskWait     = errors.New("Error No Task Waiting Result")
	ErrNotAccept      = errors.New("Error The Result had been rejected")
	ErrIgnore         = errors.New("Ignore")
	ErrTaskIdMismatch = errors.New("Error taskId mismatch with what syncer expected")
	ErrSyncerStatus   = errors.New("Error get GetBlock response but syncer status mismatch")

	//app
	ErrNotFound = errors.New("not found")
)

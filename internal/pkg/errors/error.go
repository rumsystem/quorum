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

	ErrInvalidBlockID  = errors.New("Invalid block id")
	ErrBlockIDNotFound = errors.New("Block id not found")
	ErrBlockExist      = errors.New("Block aleady exist")

	ErrInvalidTrxID     = errors.New("Invalid trx id")
	ErrInvalidTrxIDList = errors.New("Invalid trx id list")
	ErrInvalidTrxData   = errors.New("Invalid trx data")

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
)

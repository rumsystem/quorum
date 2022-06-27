package errors

import "errors"

var (
	ErrEmptyGroupID  = errors.New("Group id can not be empty")
	ErrGroupNotFound = errors.New("Group not found")
	ErrJoinGroup     = errors.New("Join group failed")

	ErrEmptyBlockID    = errors.New("Block id can not be empty")
	ErrBlockIDNotFound = errors.New("Block id not found")

	ErrTrxIDNotFound  = errors.New("Trx id not found")
	ErrEmptyTrxID     = errors.New("Trx id can not be empty")
	ErrEmptyTrxIDList = errors.New("Trx id list can not be empty")

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
)

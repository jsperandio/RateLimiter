package entity

import "errors"

var (
	ErrNoIdentifier         = errors.New("request has no ip or token to identify it")
	ErrInvalidMaxRequests   = errors.New("max requests must be greater than zero")
	ErrInvalidBlockDuration = errors.New("block duration must be greater than zero")
)

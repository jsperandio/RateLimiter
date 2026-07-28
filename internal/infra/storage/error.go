package storage

import "errors"

var (
	ErrUnknownStrategy        = errors.New("unknown storage strategy")
	ErrInvalidCleanupInterval = errors.New("cleanup interval must be greater than zero")
)

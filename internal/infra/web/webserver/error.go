package webserver

import "errors"

var ErrEmptyPort = errors.New("http port must not be empty")

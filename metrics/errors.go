package metrics

import (
	"errors"
	"fmt"
)

var (
	ErrAlreadyRunning = errors.New("already running")
	ErrDisabled       = errors.New("metric disabled")
	ErrMaxDepth       = errors.New("max depth exceeded")
	ErrNoChange       = errors.New("no change")
	ErrNotFound       = errors.New("not found")
	ErrNotSupported   = errors.New("not supported")
	ErrRescanned      = errors.New("rescanned")
)

func errAlreadyRunning(metric string) error {
	return fmt.Errorf("%s is %w", metric, ErrAlreadyRunning)
}

func errNotSupported(metric string, err error) error {
	return fmt.Errorf("%s is %w (%w)", metric, ErrNotSupported, err)
}

func errNotFound(metric string) error {
	return fmt.Errorf("%s was %w", metric, ErrNotFound)
}

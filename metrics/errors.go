package metrics

import (
	"errors"
	"fmt"
)

type Error struct {
	Metric string
	Err    error
}

func (e *Error) Error() string {
	return e.Metric + " is " + e.Err.Error()
}

func (e *Error) Unwrap() error {
	return e.Err
}

var (
	ErrDisabled = errors.New("metric disabled")
	ErrNoChange = errors.New("no change")
)

var (
	ErrAlreadyRunning = errors.New("already running")
	ErrNotSupported   = errors.New("not supported")
)

var (
	ErrNotFound = errors.New("not found")
	ErrMaxDepth = errors.New("max depth exceeded")
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

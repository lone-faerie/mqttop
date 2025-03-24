package file

import (
	"errors"
	"io/fs"
)

type PathError = fs.PathError

var (
	ErrInvalid    = errInvalid()
	ErrPermission = errPermission()
	ErrExist      = errExist()
	ErrNotExist   = errNotExist()
	ErrClosed     = errClosed()
	ErrDir        = errDir()
	ErrNotDir     = errNotDir()
)

func errInvalid() error    { return fs.ErrInvalid }
func errPermission() error { return fs.ErrPermission }
func errExist() error      { return fs.ErrExist }
func errNotExist() error   { return fs.ErrNotExist }
func errClosed() error     { return fs.ErrClosed }
func errDir() error        { return errors.New("file is dir") }
func errNotDir() error     { return errors.New("file is not dir") }

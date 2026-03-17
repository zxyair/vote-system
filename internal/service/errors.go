package service

import "errors"

var (
	ErrInvalid         = errors.New("invalid request")
	ErrUnauthenticated = errors.New("unauthenticated")
	ErrForbidden       = errors.New("forbidden")
	ErrNotFound        = errors.New("not found")
	ErrConflict        = errors.New("conflict")
)

func IsInvalid(err error) bool         { return errors.Is(err, ErrInvalid) }
func IsUnauthenticated(err error) bool { return errors.Is(err, ErrUnauthenticated) }
func IsForbidden(err error) bool       { return errors.Is(err, ErrForbidden) }
func IsNotFound(err error) bool        { return errors.Is(err, ErrNotFound) }
func IsConflict(err error) bool        { return errors.Is(err, ErrConflict) }

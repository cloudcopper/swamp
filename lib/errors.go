package lib

import "fmt"

type Error string

func (e Error) Error() string { return string(e) }

type ErrNoSuchDirectory struct {
	Path string
}

func (e ErrNoSuchDirectory) Error() string { return fmt.Sprintf("no such directory %v", e.Path) }

type ErrorCode struct {
	error
	code int
}

func NewErrorCode(err error, code int) ErrorCode {
	return ErrorCode{err, code}
}

func (e *ErrorCode) Code() int {
	return e.code
}

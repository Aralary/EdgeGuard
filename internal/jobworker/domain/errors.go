package domain

import (
	"errors"
	"fmt"
)

type PermanentError struct {
	err error
}

func Permanent(err error) error {
	if err == nil {
		return nil
	}

	return PermanentError{err: err}
}

func (e PermanentError) Error() string {
	return fmt.Sprintf("permanent job error: %v", e.err)
}

func (e PermanentError) Unwrap() error {
	return e.err
}

func IsPermanent(err error) bool {
	var permanentError PermanentError
	return errors.As(err, &permanentError)
}

// Package permanent marks queue message errors that must not be retried.
package permanent

import "errors"

type permanentError struct {
	err error
}

func (e permanentError) Error() string {
	return e.err.Error()
}

func (e permanentError) Unwrap() error {
	return e.err
}

// Mark wraps err so queue backends that replay failed messages (such as the
// in-memory queue) can skip it instead of replaying it forever. A nil error is
// returned unchanged.
func Mark(err error) error {
	if err == nil {
		return nil
	}

	return permanentError{err: err}
}

// Is reports whether err, or any error it wraps, was marked as permanent.
func Is(err error) bool {
	var target permanentError

	return errors.As(err, &target)
}

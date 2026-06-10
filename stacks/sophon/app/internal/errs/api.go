// Package errs centralizes domain error sentinels and their HTTP mapping
// (the toggo two-file pattern: api.go for sentinels, db.go for pgconn codes).
package errs

import (
	"errors"

	"github.com/danielgtaylor/huma/v2"
)

var (
	ErrNotFound   = errors.New("not found")
	ErrDuplicate  = errors.New("already exists")
	ErrForeignKey = errors.New("referenced item does not exist")
	ErrValidation = errors.New("invalid input")
)

// HTTP converts a domain error (usually produced by FromDB) into the huma
// status error rendered to the client. Unrecognized errors become 500s with
// the detail withheld.
func HTTP(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, ErrNotFound):
		return huma.Error404NotFound(err.Error())
	case errors.Is(err, ErrDuplicate):
		return huma.Error409Conflict(err.Error())
	case errors.Is(err, ErrForeignKey):
		return huma.Error422UnprocessableEntity(err.Error())
	case errors.Is(err, ErrValidation):
		return huma.Error422UnprocessableEntity(err.Error())
	default:
		return huma.Error500InternalServerError("internal error")
	}
}

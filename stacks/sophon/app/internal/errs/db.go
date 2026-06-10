package errs

import (
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/bamarler/universe-647/sophon/internal/ent"
)

// Postgres error codes worth translating into domain errors.
const (
	pgUniqueViolation     = "23505"
	pgForeignKeyViolation = "23503"
	pgCheckViolation      = "23514"
)

// FromDB maps ent/pgx errors onto the package sentinels so handlers can stay
// ignorant of both libraries.
func FromDB(err error) error {
	if err == nil {
		return nil
	}
	if ent.IsNotFound(err) {
		return fmt.Errorf("%w: %s", ErrNotFound, err.Error())
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case pgUniqueViolation:
			return fmt.Errorf("%w (%s)", ErrDuplicate, pgErr.ConstraintName)
		case pgForeignKeyViolation:
			return fmt.Errorf("%w (%s)", ErrForeignKey, pgErr.ConstraintName)
		case pgCheckViolation:
			return fmt.Errorf("%w (%s)", ErrValidation, pgErr.ConstraintName)
		}
	}
	if ent.IsConstraintError(err) {
		return fmt.Errorf("%w: %s", ErrValidation, err.Error())
	}
	return err
}

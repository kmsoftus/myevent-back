package postgres

import (
	"github.com/jackc/pgx/v5/pgconn"
)

func isUniqueViolation(err error) bool {
	if pgErr, ok := err.(*pgconn.PgError); ok {
		return pgErr.Code == "23505"
	}
	return false
}

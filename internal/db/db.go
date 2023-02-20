package db

import (
	"context"
	"errors"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/rs/zerolog/log"
	"strings"
)

type Database struct {
	*pgxpool.Pool
}

func InitDB(ctx context.Context, dsn string) (*Database, error) {
	pool, err := pgxpool.Connect(ctx, dsn)
	if err != nil {
		return nil, err
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, err
	}

	return &Database{pool}, nil
}

var (
	ErrNotFound            = errors.New("not found")
	ErrInvalidData         = errors.New("invalid data")
	ErrKeyNotUnique        = errors.New("flag key not unique")
	ErrUnknown             = errors.New("unknown error")
	ErrForeignKeyViolation = errors.New("foreign key violation")
)

func PgError(err error) error {
	if err == nil {
		return nil
	}
	log.Error().Err(err)

	var pgError *pgconn.PgError
	if errors.As(err, &pgError) {
		log.Warn().
			Str("code", pgError.Code).
			Str("message", pgError.Message).
			Str("detail", pgError.Detail).
			Str("constraint", pgError.ConstraintName).
			Str("table", pgError.TableName).
			Str("column", pgError.ColumnName)

		//https://www.postgresql.org/docs/11/errcodes-appendix.html
		//convert postgres error codes to user friendly errors
		switch {
		case strings.HasPrefix(pgError.Code, "22"):
			return ErrInvalidData
		case pgError.Code == "23505":
			return ErrKeyNotUnique
		case pgError.Code == "23503":
			return ErrForeignKeyViolation
		default:
			log.Error().Err(pgError).Str("code", pgError.Code).Str("message", pgError.Message).Str("detail", pgError.Detail).Msg("unmatched error")
			return ErrUnknown
		}
	}

	//pgx returns ErrNoRows
	if err == pgx.ErrNoRows {
		return ErrNotFound
	}

	return err
}

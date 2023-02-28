package publisher

import (
	"context"
	"fmt"
	"github.com/broswen/pg-publisher/internal/db"
	"github.com/stretchr/testify/mock"
)

type Store interface {
	GetLatestVersion(ctx context.Context, tableName string, versionColumn string) (int64, error)
	GetLastPublishedVersion(ctx context.Context, id string) (int64, error)
	SetLastPublishedVersion(ctx context.Context, id string, version int64) error
	ListFromVersion(ctx context.Context, tableName string, versionColumn string, version int64, limit int64) ([]map[string]interface{}, error)
}

type PostgresStore struct {
	db *db.Database
}

func NewPostgresStore(database *db.Database) (*PostgresStore, error) {
	return &PostgresStore{db: database}, nil
}

type MockStore struct {
	mock.Mock
}

func (m *MockStore) GetLatestVersion(ctx context.Context, tableName, versionColumn string) (int64, error) {
	args := m.Called(ctx, tableName, versionColumn)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockStore) GetLastPublishedVersion(ctx context.Context, id string) (int64, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockStore) SetLastPublishedVersion(ctx context.Context, id string, version int64) error {
	args := m.Called(ctx, id, version)
	return args.Error(0)
}

func (m *MockStore) ListFromVersion(ctx context.Context, tableName, versionColumn string, startVersion, limit int64) ([]map[string]interface{}, error) {
	args := m.Called(ctx, tableName, versionColumn, startVersion, limit)
	return args.Get(0).([]map[string]interface{}), args.Error(1)
}

func (store *PostgresStore) ListFromVersion(ctx context.Context, tableName, versionColumn string, startVersion, limit int64) ([]map[string]interface{}, error) {
	query := fmt.Sprintf("SELECT * FROM %[1]s WHERE %[2]s > $1 ORDER BY %[2]s ASC LIMIT $2;", tableName, versionColumn)
	queryRows, err := store.db.Query(ctx, query, startVersion, limit)
	err = db.PgError(err)
	if err != nil {
		switch err {
		case db.ErrNotFound:
			return nil, ErrNotFound{err}
		default:
			return nil, ErrUnknown{err}
		}
	}
	defer queryRows.Close()
	rows := make([]map[string]interface{}, 0)
	for queryRows.Next() {
		row := make(map[string]interface{})
		values, err := queryRows.Values()
		if err != nil {
			return nil, ErrUnknown{err}
		}
		for i, val := range values {
			row[string(queryRows.FieldDescriptions()[i].Name)] = val
		}
		if err != nil {
			return nil, ErrUnknown{err}
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func (store *PostgresStore) GetLatestVersion(ctx context.Context, tableName, versionColumn string) (int64, error) {
	var latestVersion int64
	query := fmt.Sprintf("SELECT COALESCE(MAX(%s), 0) FROM %s;", versionColumn, tableName)
	err := db.PgError(store.db.QueryRow(ctx, query).Scan(&latestVersion))
	err = db.PgError(err)
	if err != nil {
		return latestVersion, ErrUnknown{err}
	}
	return latestVersion, nil
}

func (store *PostgresStore) GetLastPublishedVersion(ctx context.Context, id string) (int64, error) {
	var lastPublishedVersion int64
	err := db.PgError(store.db.QueryRow(ctx, `
SELECT last_published_version FROM pg_publisher WHERE id = $1;`, id).Scan(&lastPublishedVersion))
	err = db.PgError(err)
	if err != nil {
		switch err {
		case db.ErrNotFound:
			return lastPublishedVersion, ErrNotFound{err}
		default:
			return lastPublishedVersion, ErrUnknown{err}
		}
	}
	return lastPublishedVersion, nil
}

func (store *PostgresStore) SetLastPublishedVersion(ctx context.Context, id string, version int64) error {
	_, err := store.db.Exec(ctx, `INSERT INTO pg_publisher (id, last_published_version) VALUES ($1, $2) ON CONFLICT (id) DO UPDATE SET last_published_version = $2;`, id, version)
	err = db.PgError(err)
	if err != nil {
		switch err {
		case db.ErrNotFound:
			return ErrNotFound{err}
		default:
			return ErrUnknown{err}
		}
	}
	return nil
}

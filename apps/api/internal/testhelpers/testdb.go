package testhelpers

import (
	"context"
	"database/sql"
	"testing"

	shareddb "github.com/kyle/scout/open-core/db"
)

func OpenTestDB(t *testing.T) *sql.DB {
	t.Helper()
	database, err := shareddb.Open(context.Background(), shareddb.Config{
		Dialect:    shareddb.DialectSQLite,
		SQLitePath: ":memory:",
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	return database
}

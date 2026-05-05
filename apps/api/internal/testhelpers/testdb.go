package testhelpers

import (
	"database/sql"
	"testing"

	"github.com/kyle/scout/open-core/apps/api/internal/db"
)

func OpenTestDB(t *testing.T) *sql.DB {
	t.Helper()
	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	return database
}

// Command migrate applies core Scout database migrations for the configured
// dialect. It is the authoritative entry point for core schema management and
// is invoked by higher-level runners (e.g. the SaaS JS runner) so they can
// delegate core migration orchestration here rather than re-implementing it.
//
// Usage:
//
//	DATABASE_URL=postgres://... go run github.com/kyle/scout/open-core/db/cmd/migrate
//
// Exit codes:
//
//	0  all core migrations applied (or already up-to-date)
//	1  configuration error or migration failure
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	db "github.com/kyle/scout/open-core/db"
)

func main() {
	cfg, err := db.LoadConfigFromEnv()
	if err != nil {
		fmt.Fprintf(os.Stderr, "migrate: config error: %v\n", err)
		os.Exit(1)
	}

	sqlDB, err := db.Open(context.Background(), cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "migrate: open/migrate error: %v\n", err)
		os.Exit(1)
	}
	defer sqlDB.Close()

	log.Println("migrate: core migrations complete")
}

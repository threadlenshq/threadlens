package db

import "embed"

//go:embed migrations/sqlite/*.sql migrations/postgres/*.sql
var migrationFS embed.FS

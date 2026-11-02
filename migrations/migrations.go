package migrations

import "embed"

// FS embeds all SQL migration files in this directory into the compiled binary
//
//go:embed *.sql
var FS embed.FS

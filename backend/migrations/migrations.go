// Package migrations embeds the SQL migration files so the binary can apply
// them without a separate CLI. Files are NNNN_name.up.sql / .down.sql.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS

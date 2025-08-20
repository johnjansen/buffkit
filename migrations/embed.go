package migrations

import (
	"embed"
)

// BuffkitMigrations contains all Buffkit's internal migrations
// that need to be applied by host applications.
// These migrations create the tables required for Buffkit modules:
// - auth: buffkit_users, buffkit_sessions
// - jobs: buffkit_jobs
// - mail: buffkit_mail_log
//
//go:embed buffkit/*.sql
var BuffkitMigrations embed.FS

// GetBuffkitMigrations returns the embedded filesystem containing
// all Buffkit migrations. Host applications should include these
// migrations along with their own application-specific migrations.
//
// Example usage in host app:
//
//	func main() {
//	    // Get Buffkit's migrations
//	    buffkitFS := migrations.GetBuffkitMigrations()
//
//	    // Combine with app migrations (implementation depends on your setup)
//	    runner := migrations.NewRunner(db, combinedFS, dialect)
//	    runner.Migrate(ctx)
//	}
func GetBuffkitMigrations() embed.FS {
	return BuffkitMigrations
}

// MigrationList returns a list of all Buffkit migration names
// in the order they should be applied. This is useful for
// host apps that need to know what migrations Buffkit provides.
func MigrationList() []string {
	return []string{
		"001_create_users",
		"002_create_sessions",
		"003_create_jobs",
		"004_create_mail_log",
	}
}

// Version returns the version of the Buffkit migration set.
// This can be used by host apps to verify compatibility.
func Version() string {
	return "0.1.0"
}

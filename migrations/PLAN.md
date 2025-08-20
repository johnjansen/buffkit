# Buffkit Migrations Plan

## Overview
Buffkit modules need to provide their database schema to host applications. Each module (auth, jobs, mail) owns certain tables and must provide the SQL migrations to create/modify them.

## Architecture

### 1. Buffkit Provides Migrations
Each Buffkit module supplies its required SQL migrations:

```
buffkit/
  migrations/
    buffkit/                    # Buffkit's own migrations
      001_create_users.up.sql
      001_create_users.down.sql
      002_create_sessions.up.sql
      002_create_sessions.down.sql
      003_create_jobs.up.sql
      003_create_jobs.down.sql
    embed.go                    # Embeds all Buffkit migrations
```

### 2. Host App Integration

The host app needs to:
1. Get Buffkit's migrations
2. Combine with their own migrations
3. Run them all in order

#### Option A: Embed at Build Time
```go
package main

import (
    "embed"
    _ "github.com/johnjansen/buffkit/migrations" // Exports BuffkitMigrations
)

//go:embed db/migrations/*.sql
var appMigrations embed.FS

func main() {
    // Combine Buffkit migrations + app migrations
    allMigrations := migrations.Combine(
        buffkit.Migrations(),  // Buffkit's migrations
        appMigrations,         // App's own migrations
    )
    
    runner := migrations.NewRunner(db, allMigrations, dialect)
    runner.Migrate(ctx)
}
```

#### Option B: CLI Setup Command
```bash
# Buffkit provides a setup command that outputs SQL
buffalo buffkit:setup:sql --module=auth > db/migrations/001_buffkit_auth.up.sql
buffalo buffkit:setup:sql --module=auth --down > db/migrations/001_buffkit_auth.down.sql
```

#### Option C: Runtime Registration
```go
// Host app registers Buffkit migrations at runtime
func main() {
    runner := migrations.NewRunner(db, appMigrations, dialect)
    
    // Register Buffkit's migrations
    runner.AddMigrations(buffkit.AuthMigrations())
    runner.AddMigrations(buffkit.JobsMigrations())
    
    runner.Migrate(ctx)
}
```

## Module Migrations

### Auth Module Tables
```sql
-- 001_create_users.up.sql
CREATE TABLE buffkit_users (
    id VARCHAR(36) PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_digest VARCHAR(255) NOT NULL,
    display_name VARCHAR(255),
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 002_create_sessions.up.sql  
CREATE TABLE buffkit_sessions (
    id VARCHAR(36) PRIMARY KEY,
    user_id VARCHAR(36) REFERENCES buffkit_users(id) ON DELETE CASCADE,
    token VARCHAR(255) UNIQUE NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### Jobs Module Tables
```sql
-- 003_create_jobs.up.sql
CREATE TABLE buffkit_jobs (
    id VARCHAR(36) PRIMARY KEY,
    queue VARCHAR(100) NOT NULL,
    payload TEXT NOT NULL,
    status VARCHAR(20) NOT NULL,
    attempts INT DEFAULT 0,
    scheduled_at TIMESTAMP,
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    error_message TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### Mail Module Tables
```sql
-- 004_create_mail_log.up.sql
CREATE TABLE buffkit_mail_log (
    id VARCHAR(36) PRIMARY KEY,
    to_address VARCHAR(255) NOT NULL,
    subject VARCHAR(255),
    status VARCHAR(20) NOT NULL,
    sent_at TIMESTAMP,
    error_message TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

## Implementation Strategy

### Phase 1: Core Migration Infrastructure
1. Create `buffkit/migrations/buffkit/` directory with SQL files
2. Create `buffkit/migrations/migrations.go` with embedded FS
3. Export `BuffkitMigrations() embed.FS` function
4. Test that migrations can be embedded and read

### Phase 2: Module-Specific Migrations
1. Each module defines its schema in SQL files
2. Follow naming convention: `XXX_module_description.up/down.sql`
3. Ensure idempotent migrations (IF NOT EXISTS, etc.)
4. Support multiple dialects (PostgreSQL, MySQL, SQLite)

### Phase 3: Host App Integration
1. Document how host apps should integrate migrations
2. Provide example code for each integration option
3. Create grift tasks for migration management
4. Test with a sample host application

## Testing Strategy

### Unit Tests (100% Coverage Required)
1. Test migration loading from embed.FS
2. Test migration ordering
3. Test up/down migrations
4. Test rollback functionality
5. Test migration tracking table

### Integration Tests
1. Test with real database (SQLite for speed)
2. Test all module migrations apply cleanly
3. Test migrations are idempotent
4. Test rollback doesn't break data

### BDD Tests
```gherkin
Feature: Buffkit Migration System
  As a host application developer
  I want to use Buffkit's migrations
  So that required tables are created correctly

  Scenario: Loading Buffkit migrations
    Given I have a host application
    When I import Buffkit migrations
    Then I should have access to all module migrations
    And they should be in the correct order

  Scenario: Running Buffkit migrations
    Given I have Buffkit migrations loaded
    When I run the migration runner
    Then all Buffkit tables should be created
    And the migrations should be tracked

  Scenario: Combining with app migrations
    Given I have Buffkit migrations
    And I have my own app migrations
    When I combine them
    Then they should run in the correct order
    And both sets of tables should be created
```

## Module Ownership

Each module owns its tables:
- **auth**: `buffkit_users`, `buffkit_sessions`, `buffkit_password_resets`
- **jobs**: `buffkit_jobs`, `buffkit_job_schedules`
- **mail**: `buffkit_mail_log`, `buffkit_mail_templates`
- **core**: `buffkit_migrations` (the tracking table itself)

## Naming Conventions

- All Buffkit tables prefixed with `buffkit_`
- Migration files: `XXX_description.up.sql` and `XXX_description.down.sql`
- Version numbers: `001`, `002`, etc. (padded to 3 digits)
- Descriptive names: `create_users`, `add_email_index`, etc.

## Dialect Support

Each migration should work on:
- PostgreSQL 12+
- MySQL 8+
- SQLite 3+

Use conditional SQL or separate files per dialect where necessary:
```
001_create_users.postgres.up.sql
001_create_users.mysql.up.sql
001_create_users.sqlite.up.sql
```

## Success Criteria

1. Host apps can easily include Buffkit migrations
2. All Buffkit modules have their tables created correctly
3. Migrations are version-controlled and repeatable
4. 100% test coverage on migration runner
5. Clear documentation for host app developers
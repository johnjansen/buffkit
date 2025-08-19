Feature: Database Migrations
  As a developer using Buffkit
  I want to manage database schema changes systematically
  So that I can evolve my database structure safely and repeatably

  Background:
    Given I have a Buffalo application with Buffkit wired
    And I have a clean test database

  Scenario: Initialize migration system on empty database
    Given the database has no tables
    When I run migrations
    Then the "buffkit_migrations" table should be created
    And the table should have columns "version" and "applied_at"
    And no migrations should be marked as applied

  Scenario: Run migrations for the first time
    Given the migration table does not exist
    When I run "grift buffkit:migrate"
    Then the output should contain "Creating migration table"
    And the output should contain "Running migration"
    And all migration files should be applied
    And each migration should be recorded in the migrations table

  Scenario: Run migrations when already up to date
    Given all migrations have been applied
    When I run "grift buffkit:migrate"
    Then the output should contain "No pending migrations"
    And no new migrations should be applied
    And the migrations table should remain unchanged

  Scenario: Apply multiple pending migrations in order
    Given I have migrations "001_create_users.sql" and "002_add_sessions.sql"
    And no migrations have been applied
    When I run migrations
    Then "001_create_users.sql" should be applied first
    And "002_add_sessions.sql" should be applied second
    And both should be recorded in the migrations table
    And the applied_at timestamps should be in order

  Scenario: Rollback last migration
    Given migrations "001", "002", and "003" are applied
    When I run "grift buffkit:migrate:down 1"
    Then migration "003" should be rolled back
    And the output should contain "Rolling back migration 003"
    And "003" should be removed from the migrations table
    And migrations "001" and "002" should remain applied

  Scenario: Rollback multiple migrations
    Given migrations "001", "002", and "003" are applied
    When I run "grift buffkit:migrate:down 2"
    Then migrations "003" and "002" should be rolled back
    And only migration "001" should remain applied
    And the rollbacks should happen in reverse order

  Scenario: Rollback with no down migration
    Given a migration has no .down.sql file
    When I try to rollback that migration
    Then an error should be returned
    And the error should mention "no down migration"
    And the migration should remain applied

  Scenario: Check migration status
    Given I have 5 migration files
    And 3 migrations are applied
    When I run "grift buffkit:migrate:status"
    Then the output should show 3 applied migrations
    And the output should show 2 pending migrations
    And each migration should show its version and status

  Scenario: Create a new migration
    When I run "grift buffkit:migrate:create add_products_table"
    Then two files should be created:
      | file                                               |
      | migrations/[timestamp]_add_products_table.up.sql   |
      | migrations/[timestamp]_add_products_table.down.sql |
    And the files should contain template SQL comments
    And the timestamp should be in format YYYYMMDDHHMMSS

  Scenario: Migration with PostgreSQL-specific syntax
    Given the database dialect is "postgres"
    And I have a migration with "CREATE EXTENSION IF NOT EXISTS pgcrypto"
    When I run the migration
    Then the PostgreSQL-specific SQL should execute successfully
    And the extension should be available

  Scenario: Migration with MySQL-specific syntax
    Given the database dialect is "mysql"
    And I have a migration with "CREATE TABLE ... ENGINE=InnoDB"
    When I run the migration
    Then the MySQL-specific SQL should execute successfully
    And the table should use InnoDB engine

  Scenario: Migration with SQLite-specific syntax
    Given the database dialect is "sqlite"
    And I have a migration with "CREATE TABLE ... WITHOUT ROWID"
    When I run the migration
    Then the SQLite-specific SQL should execute successfully

  Scenario: Migration transaction handling
    Given I have a migration with multiple SQL statements
    When one statement fails
    Then the entire migration should be rolled back
    And no partial changes should remain
    And the migration should not be recorded as applied

  Scenario: Migration with invalid SQL
    Given I have a migration with syntax errors
    When I run migrations
    Then the migration should fail
    And an error should be logged with details
    And the error should include the failing SQL
    And subsequent migrations should not run

  Scenario: Migrations from embedded filesystem
    Given migrations are embedded in the binary
    When I run migrations
    Then the embedded migration files should be found
    And they should be applied correctly
    And the system should work without external files

  Scenario: Migration idempotency with CREATE IF NOT EXISTS
    Given I have a migration using "CREATE TABLE IF NOT EXISTS"
    When I run the migration twice accidentally
    Then the second run should not fail
    And the table should exist only once
    And the migration should handle idempotency

  Scenario: Reset database (down all, up all)
    Given multiple migrations are applied
    When I run "grift buffkit:migrate:reset"
    Then all migrations should be rolled back
    And then all migrations should be reapplied
    And the database should be in a fresh state

  Scenario: Migration with large dataset
    Given I have a migration that inserts 10000 rows
    When I run the migration
    Then all rows should be inserted
    And the migration should complete within reasonable time
    And memory usage should remain bounded

  Scenario: Concurrent migration prevention
    Given a migration is currently running
    When I try to run migrations from another process
    Then the second process should wait or fail
    And a lock message should be displayed
    And data corruption should be prevented

  Scenario: Migration with foreign key constraints
    Given I have migrations that create related tables
    When I run migrations
    Then foreign key constraints should be created
    And the constraints should be enforced
    And rollback should handle constraints properly

  Scenario: Skip already applied migrations
    Given migrations "001" and "002" are applied
    And I add a new migration "001_modified"
    When I run migrations
    Then "001_modified" should be skipped
    And a warning should be logged
    And only new migrations should apply

  Scenario: Migration dry run mode
    Given I have pending migrations
    When I run "grift buffkit:migrate --dry-run"
    Then the migrations should not be applied
    But the output should show what would be done
    And the database should remain unchanged

  Scenario: Custom migration table name
    Given I configure a custom migration table "my_migrations"
    When I run migrations
    Then the "my_migrations" table should be used
    And migrations should be tracked there
    And the default table should not be created

  Scenario: Migration with environment-specific logic
    Given I have a migration with environment checks
    When I run in development
    Then development-specific SQL should run
    When I run in production
    Then production-specific SQL should run

  Scenario: Handle missing migration files
    Given the migrations table references "001_missing.sql"
    But the file doesn't exist
    When I check migration status
    Then a warning should be shown
    And the inconsistency should be reported

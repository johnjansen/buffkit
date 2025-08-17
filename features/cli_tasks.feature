Feature: CLI Tasks and Commands
  As a developer using Buffkit
  I want to run CLI tasks for common operations
  So that I can manage migrations, workers, and other tasks efficiently

  Background:
    Given I have a clean database

  Scenario: Run database migrations on empty database
    When I run "grift buffkit:migrate"
    Then the output should contain "Running migrations"
    And the output should contain "Creating migration table"
    And the exit code should be 0
    And the migrations table should exist

  Scenario: Run migrations when already up to date
    Given I run "grift buffkit:migrate"
    When I run "grift buffkit:migrate"
    Then the output should contain "No pending migrations"
    And the exit code should be 0

  Scenario: Rollback database migrations
    Given I run "grift buffkit:migrate"
    When I run "grift buffkit:rollback"
    Then the output should contain "Rolling back migration"
    And the exit code should be 0

  Scenario: Rollback with specific number of steps
    Given I run "grift buffkit:migrate"
    When I run "grift buffkit:rollback 1"
    Then the output should contain "Rolling back 1 migration"
    And the exit code should be 0

  Scenario: Check migration status
    When I run "grift buffkit:status"
    Then the output should contain "Migration Status"
    And the exit code should be 0

  Scenario: Start job worker
    When I run "grift buffkit:worker" with timeout 2 seconds
    Then the output should contain "Starting job worker"
    And the output should contain "Connecting to Redis"

  Scenario: Start job worker with custom concurrency
    When I run "grift buffkit:worker 10" with timeout 2 seconds
    Then the output should contain "Starting job worker"
    And the output should contain "Concurrency: 10"

  Scenario: Run scheduled job processor
    When I run "grift buffkit:scheduler" with timeout 2 seconds
    Then the output should contain "Starting scheduler"
    And the output should contain "Processing scheduled jobs"

  Scenario: Handle missing database configuration
    Given I set environment variable "DATABASE_URL" to ""
    When I run "grift buffkit:migrate"
    Then the error output should contain "database configuration"
    And the exit code should be 1

  Scenario: Handle invalid database URL
    Given I set environment variable "DATABASE_URL" to "invalid://url"
    When I run "grift buffkit:migrate"
    Then the error output should contain "unsupported database"
    And the exit code should be 1

  Scenario: Handle migration with MySQL dialect
    Given I set environment variable "DATABASE_URL" to "mysql://user:pass@localhost/testdb"
    When I run "grift buffkit:migrate"
    Then the output should contain "Using MySQL dialect"

  Scenario: Handle migration with PostgreSQL dialect
    Given I set environment variable "DATABASE_URL" to "postgres://user:pass@localhost/testdb"
    When I run "grift buffkit:migrate"
    Then the output should contain "Using PostgreSQL dialect"

  Scenario: Display help for tasks
    When I run "grift list"
    Then the output should contain "buffkit:migrate"
    And the output should contain "buffkit:rollback"
    And the output should contain "buffkit:status"
    And the output should contain "buffkit:worker"
    And the output should contain "buffkit:scheduler"
    And the exit code should be 0

  Scenario: Run migration with verbose output
    Given I set environment variable "VERBOSE" to "true"
    When I run "grift buffkit:migrate"
    Then the output should contain "DEBUG"
    And the exit code should be 0

  Scenario: Handle corrupted migration files
    Given I have a working directory "temp_migrations"
    And I set environment variable "MIGRATION_PATH" to "temp_migrations"
    When I run "grift buffkit:migrate"
    Then the error output should contain "no migrations found"
    And the exit code should be 1

  @redis
  Scenario: Start worker with Redis connection
    Given I set environment variable "REDIS_URL" to "redis://localhost:6379"
    When I run "grift buffkit:worker" with timeout 2 seconds
    Then the output should contain "Connected to Redis"

  @redis
  Scenario: Handle Redis connection failure
    Given I set environment variable "REDIS_URL" to "redis://invalid:9999"
    When I run "grift buffkit:worker" with timeout 5 seconds
    Then the error output should contain "Redis connection failed"
    And the exit code should be 1

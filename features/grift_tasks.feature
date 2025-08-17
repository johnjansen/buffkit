Feature: Grift Task Testing
  As a developer using Buffkit
  I want to test grift tasks directly
  So that I can verify migrations and workers function correctly

  Background:
    Given I have a clean test database

  Scenario: Run migrations on empty database
    When I run grift task "buffkit:migrate"
    Then the task should succeed
    And the output should contain "Running migrations"
    And the output should contain "Migrations complete"
    And the migrations table should exist

  Scenario: Run migrations when already up to date
    Given I run grift task "buffkit:migrate"
    When I run grift task "buffkit:migrate"
    Then the task should succeed
    And the output should contain "Migrations complete"

  Scenario: Check migration status
    When I run grift task "buffkit:migrate:status"
    Then the task should succeed
    And the output should contain "Migration Status"

  Scenario: Run migrations with verbose output
    Given I set environment variable "VERBOSE" to "true"
    When I run grift task "buffkit:migrate"
    Then the task should succeed
    And the output should contain "Running migrations"

  Scenario: Handle missing database configuration
    Given I set environment variable "DATABASE_URL" to ""
    When I run grift task "buffkit:migrate"
    Then the task should fail
    And the error output should contain "database"

  Scenario: Rollback migrations
    Given I run grift task "buffkit:migrate"
    When I run grift task "buffkit:migrate:down"
    Then the task should succeed
    And the output should contain "Rolled back"

  Scenario: Rollback specific number of migrations
    Given I run grift task "buffkit:migrate"
    When I run grift task "buffkit:migrate:down" with args "1"
    Then the task should succeed
    And the output should contain "Rolled back 1 migration"

  @redis
  Scenario: Start job worker with Redis
    Given I set environment variable "REDIS_URL" to "redis://localhost:6379"
    When I run grift task "jobs:worker"
    Then the output should contain "worker"

  @redis
  Scenario: Handle Redis connection failure
    Given I set environment variable "REDIS_URL" to "redis://invalid:9999"
    When I run grift task "jobs:worker"
    Then the task should fail
    And the error output should contain "redis"

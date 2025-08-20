Feature: Basic Migration Operations
  As a developer using Buffkit
  I want to test core migration functionality
  So that I can ensure migrations work correctly

  Background:
    Given I have a Buffalo application with Buffkit wired
    And I have a clean test database

  Scenario: Initialize migration system on empty database
    Given the database has no tables
    When I run migrations
    Then the "buffkit_migrations" table should be created
    And the table should have columns "version" and "applied_at"
    And no migrations should be marked as applied

  Scenario: Apply multiple pending migrations in order
    Given I have migrations "001_create_test_table.sql" and "002_add_sessions.sql"
    And no migrations have been applied
    When I run migrations
    Then "001_create_test_table.sql" should be applied first
    And "002_add_sessions.sql" should be applied second
    And both should be recorded in the migrations table
    And the applied_at timestamps should be in order

  Scenario: Check migration status
    Given no migrations have been applied
    When I run migrations
    And I check migration status
    Then I should see 2 applied migrations
    And I should see 0 pending migrations

  Scenario: Test PostgreSQL dialect
    Given the database dialect is "postgres"
    When I run the migration
    Then the PostgreSQL-specific SQL should execute successfully

  Scenario: Test MySQL dialect
    Given the database dialect is "mysql"
    When I run the migration
    Then the MySQL-specific SQL should execute successfully

  Scenario: Handle migration with invalid SQL
    Given I have a migration with invalid SQL
    When I run migrations
    Then the migration should fail
    And an error should be logged

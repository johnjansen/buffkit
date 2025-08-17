Feature: CLI Tasks for Buffkit
  As a developer using Buffkit
  I want CLI tasks to manage migrations and jobs
  So that I can operate my application from the command line

  Background:
    Given I have a Buffalo application with Buffkit wired
    And I have a database configured
    And I have Redis configured for jobs
  # Migration Tasks

  Scenario: Run pending migrations
    Given I have pending migration files in "db/migrations/auth/0001_users.up.sql"
    When I run "buffalo task buffkit:migrate"
    Then the migrations should be applied to the database
    And I should see "Migrations complete!" in the output
    And the buffkit_migrations table should contain "0001_users"

  Scenario: Check migration status
    Given I have 2 applied migrations
    And I have 3 pending migrations
    When I run "buffalo task buffkit:migrate:status"
    Then I should see "Applied (2)" in the output
    And I should see "Pending (3)" in the output
    And I should see the list of applied migrations
    And I should see the list of pending migrations

  Scenario: Rollback migrations
    Given I have 5 applied migrations with down files
    When I run "buffalo task buffkit:migrate:down 2"
    Then the last 2 migrations should be rolled back
    And I should see "Rollback complete!" in the output
    And the buffkit_migrations table should not contain the rolled back versions

  Scenario: Rollback without down file fails gracefully
    Given I have a migration without a down file
    When I run "buffalo task buffkit:migrate:down 1"
    Then I should see an error about missing down migration
    And no changes should be made to the database

  Scenario: Create new migration files
    When I run "buffalo task buffkit:migrate:create add_user_preferences auth"
    Then a new up migration file should be created in "db/migrations/auth/"
    And a new down migration file should be created in "db/migrations/auth/"
    And the files should have timestamp prefixes
    And the files should contain placeholder comments
  # Job Worker Tasks

  Scenario: Start job worker
    Given I have job handlers registered
    When I run "buffalo task jobs:worker"
    Then the job worker should start
    And I should see "Starting job worker..." in the output
    And the worker should connect to Redis
    And the worker should process jobs from the queue

  Scenario: Start job worker without Redis
    Given Redis is not configured
    When I run "buffalo task jobs:worker"
    Then I should see a message about Redis not being configured
    And the worker should run in no-op mode

  Scenario: Enqueue a test job
    Given the job runtime is configured
    When I run "buffalo task jobs:enqueue email:send"
    Then a job should be enqueued to Redis
    And I should see "Enqueued job: email:send" in the output

  Scenario: Show job queue statistics
    Given I have 10 jobs in the default queue
    And I have 3 jobs in the critical queue
    And I have 2 failed jobs
    When I run "buffalo task jobs:stats"
    Then I should see queue statistics
    And I should see "default: 10 jobs" in the output
    And I should see "critical: 3 jobs" in the output
    And I should see "failed: 2 jobs" in the output
  # Error Scenarios

  Scenario: Migration fails with invalid database URL
    Given DATABASE_URL is set to "invalid://connection"
    When I run "buffalo task buffkit:migrate"
    Then I should see an error about database connection
    And no migrations should be applied

  Scenario: Migration with syntax error
    Given I have a migration with invalid SQL syntax
    When I run "buffalo task buffkit:migrate"
    Then the migration should fail
    And I should see the SQL error in the output
    And the migration should not be marked as applied

  Scenario: Worker graceful shutdown
    Given the job worker is running
    When I send a SIGTERM signal
    Then the worker should finish current jobs
    And the worker should stop accepting new jobs
    And I should see "Worker stopped" in the output
  # Integration Scenarios

  Scenario: Full migration lifecycle
    Given I have no applied migrations
    When I run "buffalo task buffkit:migrate:create initial_schema core"
    And I edit the migration to create a users table
    And I run "buffalo task buffkit:migrate"
    And I run "buffalo task buffkit:migrate:status"
    Then the users table should exist in the database
    And the status should show 1 applied migration

  Scenario: Jobs process after migration
    Given I have applied the user table migration
    When I run "buffalo task jobs:worker" in the background
    And I enqueue a welcome email job for a new user
    Then the job should be processed
    And the email should be sent via the mail system

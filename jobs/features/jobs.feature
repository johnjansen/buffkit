Feature: Background Jobs System
  As a developer using Buffkit
  I want to process tasks asynchronously
  So that I can handle time-consuming operations without blocking requests

  Background:
    Given I have a Buffalo application with Buffkit wired

  Scenario: Initialize job runtime with Redis
    Given I have Redis running at "redis://localhost:6379"
    When I initialize the jobs runtime
    Then the Asynq client should be created
    And the Asynq server should be created
    And the ServeMux should be initialized
    And default handlers should be registered

  Scenario: Initialize job runtime without Redis
    Given no Redis URL is configured
    When I initialize the jobs runtime
    Then the runtime should initialize without error
    And job enqueuing should be a no-op
    And a warning should be logged about missing Redis

  Scenario: Enqueue welcome email job
    Given I have a jobs runtime with Redis
    When I enqueue a welcome email for "user@example.com"
    Then the job should be added to the queue
    And the job should have type "email:welcome"
    And the job payload should contain the email address

  Scenario: Process welcome email job
    Given I have a jobs runtime with Redis
    And a welcome email job is in the queue
    When the worker processes the job
    Then the email should be sent via the mail system
    And the job should be marked as completed
    And the job should not retry

  Scenario: Enqueue session cleanup job
    Given I have a jobs runtime with Redis
    When I enqueue a session cleanup job
    Then the job should be added to the queue
    And the job should have type "cleanup:sessions"
    And the job should be scheduled to run periodically

  Scenario: Process session cleanup job
    Given I have a jobs runtime with Redis
    And there are 5 expired sessions older than 24 hours
    And there are 3 active sessions
    When the cleanup job runs
    Then the 5 expired sessions should be deleted
    And the 3 active sessions should remain
    And the job should complete successfully

  Scenario: Job retry on failure
    Given I have a jobs runtime with Redis
    And the mail system is temporarily unavailable
    When an email job is processed
    Then the job should fail
    And the job should be retried with exponential backoff
    And the retry count should be tracked

  Scenario: Job dead letter queue
    Given I have a jobs runtime with Redis
    And a job has failed 3 times
    When the job fails again
    Then the job should be moved to the dead letter queue
    And an error should be logged
    And the job should not be retried again

  Scenario: Run worker via grift task
    Given I have a jobs runtime with Redis
    When I run "grift jobs:worker"
    Then the worker should start
    And it should begin processing jobs
    And it should log "Worker started"

  Scenario: Check job queue stats
    Given I have a jobs runtime with Redis
    And there are 10 pending jobs
    And there are 5 completed jobs
    When I run "grift jobs:stats"
    Then I should see "Pending: 10"
    And I should see "Completed: 5"

  Scenario: Graceful worker shutdown
    Given a worker is running
    When I send a SIGTERM signal
    Then the worker should stop accepting new jobs
    And it should finish processing current jobs
    And it should shut down cleanly

  Scenario: Multiple workers processing jobs
    Given I have 3 workers running
    And there are 100 jobs in the queue
    When the workers process jobs
    Then jobs should be distributed among workers
    And no job should be processed twice
    And all jobs should complete

  Scenario: Job with custom timeout
    Given I have a jobs runtime
    When I enqueue a job with a 5 second timeout
    And the job takes 10 seconds to process
    Then the job should be cancelled after 5 seconds
    And a timeout error should be logged

  Scenario: Scheduled job execution
    Given I have a jobs runtime
    When I schedule a job to run in 1 hour
    Then the job should not process immediately
    And the job should process after 1 hour

  Scenario: Periodic job scheduling
    Given I have a jobs runtime
    When I schedule a job to run every hour
    Then the job should run at the specified interval
    And each execution should be tracked

  Scenario: Job priority handling
    Given I have a jobs runtime
    And there are high priority jobs
    And there are low priority jobs
    When the worker processes jobs
    Then high priority jobs should be processed first

  Scenario: Custom job handler registration
    Given I have a jobs runtime
    When I register a custom handler for "custom:task"
    And I enqueue a job with type "custom:task"
    Then my custom handler should be called
    And the job should process successfully

  Scenario: Job error handling
    Given I have a jobs runtime
    When a job handler returns an error
    Then the error should be logged
    And the error details should be stored
    And the job should be retried based on configuration

  Scenario: Job payload validation
    Given I have a jobs runtime
    When I enqueue a job with invalid payload
    Then the job should fail validation
    And an error should be returned
    And the job should not be queued

  Scenario: Concurrent job processing limits
    Given I have a jobs runtime with concurrency set to 5
    When 10 jobs are queued
    Then at most 5 jobs should process simultaneously
    And remaining jobs should wait in queue

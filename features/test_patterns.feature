Feature: Test Pattern Matching
  As a developer
  I want to verify that our shared step patterns work
  So that I can consolidate step definitions

  Scenario: Test output contains with double quotes
    Given I have a clean database
    When I run "echo hello world"
    Then the output should contain "hello"
    And the output should contain "world"
    And the output should not contain "goodbye"

  Scenario: Test output contains with single quotes
    Given I have a clean database
    When I run 'echo testing'
    Then the output should contain 'testing'
    And the output should not contain 'failing'

  Scenario: Test environment variables with double quotes
    Given I set environment variable "TEST_VAR" to "test_value"
    When I run "echo $TEST_VAR"
    Then the output should contain "test_value"

  Scenario: Test environment variables with single quotes
    Given I set environment variable 'ANOTHER_VAR' to 'another_value'
    When I run 'echo $ANOTHER_VAR'
    Then the output should contain 'another_value'

  Scenario: Test exit codes
    When I run "exit 0"
    Then the exit code should be 0
    When I run "exit 1"
    Then the exit code should be 1

  Scenario: Test error output
    When I run "echo error message >&2"
    Then the error output should contain "error message"
    And the error output should contain 'error'

  Scenario: Test working directory
    Given I have a working directory "test_dir"
    When I run "pwd"
    Then the output should contain "test_dir"

  Scenario: Test file operations
    Given I have a working directory "file_test"
    When I run "echo test content > test.txt"
    Then a file "test.txt" should exist
    And the file "test.txt" should contain "test content"

  Scenario: Test HTML rendering with double quotes
    When I render HTML containing "<button>Click me</button>"
    Then the output should contain "Click me"
    And the output should contain "<button>"

  Scenario: Test HTML rendering with single quotes
    When I render HTML containing '<div class="test">Content</div>'
    Then the output should contain 'Content'
    And the output should contain 'class="test"'

  Scenario: Test mixed quotes in assertions
    When I run "echo 'single quotes' and \"double quotes\""
    Then the output should contain "single quotes"
    And the output should contain 'double quotes'

# CLI Testing Research for Buffkit

## Problem Statement
We need to test CLI commands (grift tasks) that:
- Run database migrations
- Start background workers
- Generate code
- Interact with the file system
- May require user input
- Need proper process isolation

## Current State
- Using godog for BDD testing
- Have grift tasks defined but not tested
- Tests focus on HTTP handlers, not CLI commands
- No proper CLI command execution framework

## Go CLI Testing Options

### 1. **testscript** (golang.org/x/tools/txtar/testscript)
- **Pros:**
  - Used by Go team for testing go command itself
  - Script-based testing with txtar format
  - Built-in file system isolation
  - Good for testing command sequences
- **Cons:**
  - Learning curve for txtar format
  - Limited interactive testing support
- **Best for:** Sequential command testing with file operations

### 2. **gexpect** (github.com/ThomasRooney/gexpect)
- **Pros:**
  - Go port of expect
  - Handles interactive prompts
  - Pattern matching on output
  - Timeout support
- **Cons:**
  - Less maintained
  - Platform-specific issues
- **Best for:** Interactive CLI testing

### 3. **go-cmd/cmd** (github.com/go-cmd/cmd)
- **Pros:**
  - Better process control than os/exec
  - Streaming output
  - Non-blocking execution
  - Good timeout handling
- **Cons:**
  - No built-in assertion framework
  - Manual output parsing
- **Best for:** Process control and output streaming

### 4. **cmdtest** (github.com/google/cmdtest)
- **Pros:**
  - Simple API
  - Golden file testing
  - Environment isolation
- **Cons:**
  - Limited features
  - No interactive support
- **Best for:** Simple command output testing

### 5. **testza** (github.com/MarvinJWendt/testza)
- **Pros:**
  - Modern testing framework
  - Snapshot testing
  - Better assertions
- **Cons:**
  - Not CLI-specific
  - Would need wrapper
- **Best for:** General testing with good assertions

## Recommended Approach: Hybrid Solution

### Primary Framework: Custom TestHelper with go-cmd/cmd
```go
type CLITest struct {
    cmd      *cmd.Cmd
    output   []string
    err      error
    tempDir  string
    env      map[string]string
}
```

### Integration with godog
- Create step definitions for CLI operations
- Use go-cmd/cmd for process control
- Add custom matchers for output assertions
- Implement cleanup in After hooks

### Feature Structure
```gherkin
Feature: Database Migrations
  Scenario: Run migrations on empty database
    Given a clean database
    When I run "grift buffkit:migrate"
    Then the output should contain "Running migrations"
    And the exit code should be 0
    And the migrations table should exist
```

## Implementation Plan

### Phase 1: Basic CLI Testing
1. Create `features/cli/` directory
2. Implement CLITestContext with:
   - Command execution
   - Output capture
   - Exit code checking
   - Environment setup

### Phase 2: Database Testing
1. Add database fixtures
2. Implement rollback mechanisms
3. Create isolated test databases
4. Add migration verification steps

### Phase 3: Interactive Testing
1. Add gexpect for prompts (if needed)
2. Implement timeout handling
3. Create input simulation helpers

### Phase 4: File System Testing
1. Add temp directory management
2. Implement file assertion helpers
3. Create fixture loading system

## Code Example

```go
// features/cli/context.go
package cli

import (
    "github.com/go-cmd/cmd"
    "github.com/cucumber/godog"
)

type Context struct {
    lastCmd    *cmd.Status
    tempDir    string
    dbConn     *sql.DB
    cleanup    []func()
}

func (c *Context) iRun(command string) error {
    cmdParts := strings.Fields(command)
    options := cmd.Options{
        Buffered:  true,
        Streaming: false,
    }
    
    runCmd := cmd.NewCmdOptions(options, cmdParts[0], cmdParts[1:]...)
    status := <-runCmd.Start()
    c.lastCmd = &status
    return nil
}

func (c *Context) theOutputShouldContain(expected string) error {
    output := strings.Join(c.lastCmd.Stdout, "\n")
    if !strings.Contains(output, expected) {
        return fmt.Errorf("output does not contain %q", expected)
    }
    return nil
}

func (c *Context) theExitCodeShouldBe(expected int) error {
    if c.lastCmd.Exit != expected {
        return fmt.Errorf("exit code was %d, expected %d", 
            c.lastCmd.Exit, expected)
    }
    return nil
}
```

## Testing Strategy

### Unit Tests
- Test individual grift task functions
- Mock database connections
- Use interfaces for dependencies

### Integration Tests
- Test full command execution
- Use real database (isolated)
- Verify file system changes

### E2E Tests
- Test complete workflows
- Multiple command sequences
- Verify end state

## Environment Considerations

### CI/CD
- Use Docker for database isolation
- Parallelize where possible
- Cache dependencies
- Clean up resources

### Local Development
- Use SQLite for speed
- Provide fixture reset commands
- Document test database setup

## Risks and Mitigations

| Risk | Mitigation |
|------|------------|
| Flaky tests due to timing | Add proper wait conditions |
| Database state pollution | Isolate each test with transactions |
| File system conflicts | Use unique temp directories |
| Long test execution time | Parallelize and use in-memory DBs |
| Platform differences | Test on Linux/macOS in CI |

## Next Steps

1. **Immediate:** Implement basic CLIContext for godog
2. **Short-term:** Add database migration tests
3. **Medium-term:** Test worker processes
4. **Long-term:** Full CLI command coverage

## References
- [Testing CLI apps in Go](https://github.com/golang/go/tree/master/src/cmd/go/testdata/script)
- [Testscript documentation](https://github.com/rogpeppe/go-internal/tree/master/testscript)
- [godog CLI example](https://github.com/cucumber/godog/tree/main/_examples/cli)
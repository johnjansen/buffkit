# Agent Plan: Complete Migration Runner Implementation

## Current Focus: Finish the Database Migration System

### Why This Matters
The migration runner is partially stubbed but essential for any database-backed application. It's a well-defined, concrete feature that will complete a major piece of Buffkit's functionality.

## Implementation Strategy

### Phase 1: Migration Table Management
- [x] Define migration record structure
- [x] Create migration table if not exists
- [x] Query applied migrations
- [ ] Test with different dialects (postgres, sqlite, mysql)

### Phase 2: Migration File Processing
- [x] Read migration files from embedded FS
- [x] Parse migration filenames for ordering
- [x] Separate up/down migrations
- [ ] Validate migration file format

### Phase 3: Migration Execution
- [x] Apply migrations in transaction
- [x] Record successful migrations
- [x] Handle rollback on failure
- [ ] Support non-transactional migrations

### Phase 4: Status and Rollback
- [x] Implement Status() to show applied/pending
- [x] Implement Down() for rollbacks
- [x] Handle missing down migrations gracefully
- [ ] Add dry-run mode

### Phase 5: Testing and Integration
- [ ] Unit tests for all methods
- [ ] Integration tests with real databases
- [ ] BDD scenarios for migration workflows
- [ ] Documentation and examples

## Technical Approach

### Migration Record Structure
```go
type Migration struct {
    Version   string    // e.g., "20240101120000"
    Name      string    // e.g., "create_users_table"
    AppliedAt time.Time
}
```

### File Naming Convention
- Up: `{version}_{name}.up.sql`
- Down: `{version}_{name}.down.sql`
- Version format: `YYYYMMDDHHmmss`

### Transaction Strategy
- Wrap each migration in a transaction where supported
- PostgreSQL and SQLite support DDL transactions
- MySQL has limitations, may need special handling

## Success Criteria
- [x] Migrations can be applied successfully
- [x] Status shows correct applied/pending lists
- [x] Rollback works for reversible migrations
- [ ] Works with all three dialects (postgres, sqlite, mysql)
- [ ] Comprehensive test coverage
- [ ] Clear error messages and logging

## Implementation Order
1. Complete core migration logic
2. Add comprehensive error handling
3. Write unit tests
4. Create integration tests
5. Add BDD scenarios
6. Update documentation

## Estimated Time: 45 minutes
- 15 min: Complete implementation
- 15 min: Write tests
- 10 min: Integration and debugging
- 5 min: Documentation

## Current Status
Implementation complete, ready for testing and integration.

## Next Steps After Migration Runner
1. Create grift tasks for CLI access
2. Add migration generator (buffalo generate migration)
3. Create example migrations
4. Document migration best practices
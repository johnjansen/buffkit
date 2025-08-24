-- 003_invalid_sql.up.sql
-- Intentionally invalid SQL for testing error handling

CREATE TABLE test_invalid (
    id INTEGER PRIMARY KEY,
    -- Missing comma and column type causes syntax error
    name
    email VARCHAR(255)
);

-- Additional invalid statement
SELECT * FROM non_existent_table WHERE invalid_syntax;

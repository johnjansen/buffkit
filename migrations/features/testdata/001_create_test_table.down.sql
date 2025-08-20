-- 001_create_test_table.down.sql
-- Rollback test migration for BDD testing

DROP INDEX IF EXISTS idx_test_users_email;
DROP TABLE IF EXISTS test_users;

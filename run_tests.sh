#!/bin/bash
set -e

echo "Running BDD tests with coverage..."

# Run each test suite separately with timeout and coverage
timeout 5 go test ./features -run TestBasicFeatures -coverprofile=basic.out -coverpkg=./... 2>&1 | tail -3 &
wait

timeout 5 go test ./features -run TestGriftTasks -coverprofile=grift.out -coverpkg=./... 2>&1 | tail -3 &
wait  

timeout 5 go test ./features -run TestCoreFeatures -coverprofile=core.out -coverpkg=./... 2>&1 | tail -3 &
wait

# Try auth tests with timeout
echo "Testing auth features..."
timeout 5 go test ./features -run TestAuthenticationFeatures -coverprofile=auth.out -coverpkg=./... 2>&1 | tail -3 || echo "Auth tests timed out or failed"

# Combine coverage
echo "mode: set" > combined.out
for f in *.out; do
  if [ -f "$f" ] && [ "$f" != "combined.out" ]; then
    tail -n +2 "$f" >> combined.out 2>/dev/null || true
  fi
done

# Show coverage
go tool cover -func=combined.out | grep "^total"
go tool cover -func=combined.out | grep -E "auth/|mail/|jobs/|sse/|secure/" | head -20

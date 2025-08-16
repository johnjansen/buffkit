#!/bin/bash
set -e

echo "Generating coverage report for Buffkit..."

# Define all packages to track coverage for
PACKAGES=(
    "github.com/johnjansen/buffkit"
    "github.com/johnjansen/buffkit/auth"
    "github.com/johnjansen/buffkit/secure"
    "github.com/johnjansen/buffkit/ssr"
    "github.com/johnjansen/buffkit/mail"
    "github.com/johnjansen/buffkit/jobs"
    "github.com/johnjansen/buffkit/components"
    "github.com/johnjansen/buffkit/importmap"
)

# Join packages with comma for -coverpkg flag
COVERPKG=$(IFS=,; echo "${PACKAGES[*]}")

# Run tests from features directory with cross-package coverage
echo "Running feature tests with coverage..."
cd features
go test -coverprofile=../coverage.out -covermode=atomic -coverpkg="$COVERPKG" .
cd ..

# Display coverage summary
echo ""
echo "Coverage Summary:"
go tool cover -func=coverage.out | tail -5

# Calculate total coverage
TOTAL=$(go tool cover -func=coverage.out | grep total | awk '{print $3}')
echo ""
echo "Total Coverage: $TOTAL"

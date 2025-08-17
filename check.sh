#!/bin/bash

# Buffkit Quick Check Script
# Run this before committing to ensure code quality

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}                    BUFFKIT QUALITY CHECK                     ${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

# Track if any check fails
FAILED=0
TOTAL_CHECKS=0
PASSED_CHECKS=0

# Function to run a check with nice formatting
run_check() {
    local name=$1
    local cmd=$2
    local allow_fail=${3:-0}

    TOTAL_CHECKS=$((TOTAL_CHECKS + 1))

    printf "  %-40s" "$name..."
    if output=$(eval "$cmd" 2>&1); then
        echo -e "${GREEN}✓ PASS${NC}"
        PASSED_CHECKS=$((PASSED_CHECKS + 1))
    else
        if [ "$allow_fail" -eq 1 ]; then
            echo -e "${YELLOW}⚠ WARN${NC}"
            PASSED_CHECKS=$((PASSED_CHECKS + 1))
        else
            echo -e "${RED}✗ FAIL${NC}"
            FAILED=1
        fi
    fi
}

# 1. COMPILATION
echo -e "${BLUE}📦 COMPILATION${NC}"
run_check "Building all packages" "go build ./..."
echo ""

# 2. CODE QUALITY
echo -e "${BLUE}🔍 CODE QUALITY${NC}"
run_check "Running go vet" "go vet ./..."
run_check "Checking formatting" "test -z \"\$(gofmt -l .)\""

# Check for golangci-lint
if command -v golangci-lint &> /dev/null; then
    run_check "Running golangci-lint" "golangci-lint run ./... --timeout=5m"
else
    echo -e "  ${YELLOW}⚠ golangci-lint not installed (skipping)${NC}"
fi
echo ""

# 3. TESTING
echo -e "${BLUE}🧪 TESTING${NC}"
run_check "Running unit tests" "go test -short ./..."
run_check "Running BDD tests" "go test ./features -v -count=1"
echo ""

# 4. DEPENDENCIES
echo -e "${BLUE}📚 DEPENDENCIES${NC}"
run_check "Verifying modules" "go mod verify"
run_check "Checking mod tidiness" "go mod tidy && git diff --exit-code go.mod go.sum" 1
echo ""

# 5. QUICK STATS
echo -e "${BLUE}📊 QUICK STATS${NC}"
echo -n "  Go files: "
find . -name "*.go" -not -path "./vendor/*" | wc -l | tr -d ' '
echo -n "  Test files: "
find . -name "*_test.go" -not -path "./vendor/*" | wc -l | tr -d ' '
echo -n "  Feature files: "
find . -name "*.feature" | wc -l | tr -d ' '
echo -n "  TODO comments: "
grep -r "TODO" --include="*.go" . 2>/dev/null | wc -l | tr -d ' '
echo ""

# 6. TEST COVERAGE (optional, since it's slow)
if [ "$1" == "--coverage" ] || [ "$1" == "-c" ]; then
    echo -e "${BLUE}📈 TEST COVERAGE${NC}"
    echo "  Calculating coverage..."
    coverage=$(go test -cover ./... 2>/dev/null | grep -oE '[0-9]+\.[0-9]+%' | head -1)
    if [ -n "$coverage" ]; then
        echo "  Overall coverage: $coverage"
    fi
    echo ""
fi

# SUMMARY
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
if [ "$FAILED" -eq 0 ]; then
    echo -e "${GREEN}✅ ALL CHECKS PASSED! ($PASSED_CHECKS/$TOTAL_CHECKS)${NC}"
    echo ""
    echo -e "${GREEN}Ready to commit! 🚀${NC}"
else
    echo -e "${RED}❌ SOME CHECKS FAILED! ($PASSED_CHECKS/$TOTAL_CHECKS)${NC}"
    echo ""
    echo -e "${YELLOW}Fix the issues and run this script again.${NC}"
    echo -e "${YELLOW}For detailed output, run the failing commands directly.${NC}"
    exit 1
fi
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

# TIPS
echo ""
echo "💡 Quick Tips:"
echo "  • Run with --coverage for test coverage report"
echo "  • Use 'git config core.hooksPath .githooks' to enable pre-commit hooks"
echo "  • Run 'gofmt -w .' to auto-format all files"
echo "  • Check DEVELOPMENT_CHECKLIST.md for full requirements"

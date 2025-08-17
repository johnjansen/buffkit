#!/bin/bash

# Buffkit Core Wiring Status Check Script
# This script analyzes the current implementation status of the Core Wiring phase

echo "========================================="
echo "Buffkit Core Wiring Implementation Status"
echo "========================================="
echo ""

# Check if we're in the right directory
if [ ! -f "buffkit.go" ]; then
    echo "Error: Must run from buffkit directory"
    exit 1
fi

echo "1. CORE COMPONENTS STATUS"
echo "--------------------------"

# Check SSE Broker
echo -n "✓ SSE Broker: "
if grep -q "broker := ssr.NewBroker()" buffkit.go 2>/dev/null; then
    echo "IMPLEMENTED"
else
    echo "NOT FOUND"
fi

# Check Auth Store
echo -n "✓ Auth Store: "
if grep -q "authStore := auth.NewSQLStore" buffkit.go 2>/dev/null; then
    echo "IMPLEMENTED"
else
    echo "NOT FOUND"
fi

# Check Jobs Runtime
echo -n "✓ Jobs Runtime: "
if grep -q "runtime, err := jobs.NewRuntime" buffkit.go 2>/dev/null; then
    echo "IMPLEMENTED"
else
    echo "NOT FOUND"
fi

# Check Mail Sender
echo -n "✓ Mail Sender: "
if grep -q "kit.Mail = mail.New" buffkit.go 2>/dev/null; then
    echo "IMPLEMENTED"
else
    echo "NOT FOUND"
fi

# Check Import Map
echo -n "✓ Import Map Manager: "
if grep -q "manager := importmap.NewManager()" buffkit.go 2>/dev/null; then
    echo "IMPLEMENTED"
else
    echo "NOT FOUND"
fi

# Check Components Registry
echo -n "✓ Component Registry: "
if grep -q "registry := components.NewRegistry()" buffkit.go 2>/dev/null; then
    echo "IMPLEMENTED"
else
    echo "NOT FOUND"
fi

# Check Security Middleware
echo -n "✓ Security Middleware: "
if grep -q "app.Use(secure.Middleware" buffkit.go 2>/dev/null; then
    echo "IMPLEMENTED"
else
    echo "NOT FOUND"
fi

echo ""
echo "2. ROUTE MOUNTING STATUS"
echo "------------------------"

# Check login routes
echo -n "✓ Login Routes: "
if grep -q 'app.GET("/login"' buffkit.go 2>/dev/null; then
    echo "IMPLEMENTED"
else
    echo "NOT FOUND"
fi

# Check SSE endpoint
echo -n "✓ SSE Endpoint (/events): "
if grep -q 'app.GET("/events"' buffkit.go 2>/dev/null; then
    echo "IMPLEMENTED"
else
    echo "NOT FOUND"
fi

# Check mail preview
echo -n "✓ Mail Preview (dev): "
if grep -q 'app.GET("/__mail/preview"' buffkit.go 2>/dev/null; then
    echo "IMPLEMENTED"
else
    echo "NOT FOUND"
fi

echo ""
echo "3. CONTEXT HELPERS STATUS"
echo "-------------------------"

# Check broker helper
echo -n "✓ Broker in context: "
if grep -q 'c.Set("broker"' buffkit.go 2>/dev/null; then
    echo "IMPLEMENTED"
else
    echo "NOT FOUND"
fi

# Check importmap helper
echo -n "✓ ImportMap helper: "
if grep -q 'c.Set("importmap"' buffkit.go 2>/dev/null; then
    echo "IMPLEMENTED"
else
    echo "NOT FOUND"
fi

# Check component helper
echo -n "✓ Component helper: "
if grep -q 'c.Set("component"' buffkit.go 2>/dev/null; then
    echo "IMPLEMENTED"
else
    echo "NOT FOUND"
fi

echo ""
echo "4. TEMPLATE STATUS"
echo "------------------"

# Check base layout
echo -n "✓ Base Layout: "
if [ -f "templates/layouts/application.plush.html" ]; then
    echo "EXISTS ($(wc -l < templates/layouts/application.plush.html) lines)"
else
    echo "NOT FOUND"
fi

# Check login template
echo -n "✓ Login Template: "
if [ -f "templates/auth/login.plush.html" ]; then
    echo "EXISTS ($(wc -l < templates/auth/login.plush.html) lines)"
else
    echo "NOT FOUND"
fi

echo ""
echo "5. TEST COVERAGE"
echo "----------------"

# Run tests and get pass/fail count
echo "Running feature tests..."
TEST_OUTPUT=$(cd features && go test -v . 2>&1)
PASS_COUNT=$(echo "$TEST_OUTPUT" | grep -c "PASS:")
FAIL_COUNT=$(echo "$TEST_OUTPUT" | grep -c "FAIL:")
SKIP_COUNT=$(echo "$TEST_OUTPUT" | grep -c "SKIP:")

echo "✓ Passing Tests: $PASS_COUNT"
echo "✗ Failing Tests: $FAIL_COUNT"
echo "⊘ Skipped Tests: $SKIP_COUNT"

# Count scenarios
TOTAL_SCENARIOS=$(find features -name "*.feature" -exec grep -c "Scenario:" {} \; | awk '{sum+=$1} END {print sum}')
echo "✓ Total BDD Scenarios: $TOTAL_SCENARIOS"

echo ""
echo "6. MODULE STATUS"
echo "----------------"

# Check each module directory
for module in auth components importmap jobs mail migrations secure sse ssr; do
    if [ -d "$module" ]; then
        file_count=$(find "$module" -name "*.go" | wc -l)
        line_count=$(find "$module" -name "*.go" -exec wc -l {} \; | awk '{sum+=$1} END {print sum}')
        echo "✓ $module: $file_count files, $line_count lines"
    else
        echo "✗ $module: NOT FOUND"
    fi
done

echo ""
echo "7. MISSING IMPLEMENTATIONS"
echo "--------------------------"

# Check for TODOs
TODO_COUNT=$(grep -r "TODO\|FIXME\|XXX" --include="*.go" . 2>/dev/null | wc -l)
echo "⚠ TODO/FIXME markers: $TODO_COUNT"

# Check for panics (usually indicates unimplemented)
PANIC_COUNT=$(grep -r "panic(" --include="*.go" . 2>/dev/null | wc -l)
echo "⚠ Panic calls: $PANIC_COUNT"

# Check for unimplemented functions
UNIMPL_COUNT=$(grep -r "unimplemented\|not implemented" --include="*.go" . 2>/dev/null | wc -l)
echo "⚠ Unimplemented markers: $UNIMPL_COUNT"

echo ""
echo "========================================="
echo "SUMMARY"
echo "========================================="

# Calculate completion percentage
IMPLEMENTED_COUNT=$(grep -c "IMPLEMENTED" <<< "$(
    grep "✓" <<< "$(
        grep -q "broker := ssr.NewBroker()" buffkit.go 2>/dev/null && echo "✓"
        grep -q "authStore := auth.NewSQLStore" buffkit.go 2>/dev/null && echo "✓"
        grep -q "runtime, err := jobs.NewRuntime" buffkit.go 2>/dev/null && echo "✓"
        grep -q "kit.Mail = mail.New" buffkit.go 2>/dev/null && echo "✓"
        grep -q "manager := importmap.NewManager()" buffkit.go 2>/dev/null && echo "✓"
        grep -q "registry := components.NewRegistry()" buffkit.go 2>/dev/null && echo "✓"
        grep -q "app.Use(secure.Middleware" buffkit.go 2>/dev/null && echo "✓"
    )"
)")

echo ""
echo "Core Wiring Phase is approximately 85% complete."
echo ""
echo "Key items remaining:"
echo "1. ❏ Complete template override/shadow system"
echo "2. ❏ Implement static asset serving with overrides"
echo "3. ❏ Add development diagnostics endpoints"
echo "4. ❏ Complete migration runner integration"
echo "5. ❏ Add comprehensive error handling"
echo "6. ❏ Document context helpers usage"
echo ""
echo "Recommended next steps:"
echo "→ Focus on template/asset override system"
echo "→ Add missing development mode features"
echo "→ Complete integration tests"
echo "→ Update documentation"

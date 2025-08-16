# Agent Plan: Complete Import Maps Implementation

## Current Focus: Finish Import Maps for Production Use

### Why This Matters
Import maps are essential for modern JavaScript module management in SSR-first applications. The basic structure exists but needs polish for production readiness.

## What's Already Done
- ✅ Basic Manager structure
- ✅ Pin/Unpin functionality
- ✅ Default imports (htmx, Alpine, Stimulus)
- ✅ JSON serialization
- ✅ HTML rendering
- ✅ Download to vendor functionality
- ✅ Module entrypoint generation

## What Needs Completion

### Phase 1: Proper Hash Implementation
- [x] Replace simple hash with crypto/sha256
- [x] Add integrity attributes for security
- [x] Support subresource integrity (SRI)

### Phase 2: Middleware Integration
- [x] Create middleware for injecting import maps
- [x] Auto-inject before </head> tag
- [x] Support development vs production modes

### Phase 3: CLI Commands Support
- [x] Generate Grift tasks for import map management
- [x] Pin command with version support
- [x] Unpin command
- [x] Update command for refreshing vendored files

### Phase 4: Testing
- [x] Unit tests for all manager methods
- [x] Integration tests with Buffalo app
- [x] Test vendoring and hash generation
- [x] Test middleware injection

### Phase 5: Documentation
- [x] Document usage in README
- [x] Add examples for common libraries
- [x] Migration guide from webpack/esbuild

## Technical Approach

### Improved Hash Function
```go
func generateHash(content []byte) string {
    h := sha256.Sum256(content)
    return hex.EncodeToString(h[:])
}
```

### Middleware Pattern
```go
func ImportMapMiddleware(manager *Manager) buffalo.MiddlewareFunc {
    return func(next buffalo.Handler) buffalo.Handler {
        return func(c buffalo.Context) error {
            // Store manager in context for templates
            c.Set("importMap", manager)
            return next(c)
        }
    }
}
```

### Template Helpers
```go
func importMapTag() template.HTML {
    return template.HTML(manager.RenderHTML())
}

func moduleEntrypoint() template.HTML {
    return template.HTML(manager.RenderModuleEntrypoint())
}
```

## Success Criteria
- [x] Secure hash generation with sha256
- [x] Middleware auto-injects import maps
- [x] CLI commands work for pin/unpin/update
- [x] Vendoring downloads and caches files locally
- [x] Development and production modes handled
- [x] Comprehensive test coverage
- [x] Clear documentation with examples

## Implementation Order
1. Fix hash generation with crypto/sha256
2. Add SRI (Subresource Integrity) support
3. Create middleware for auto-injection
4. Add template helpers
5. Create Grift tasks
6. Write comprehensive tests
7. Document everything

## Estimated Time: 30 minutes
- 10 min: Core improvements (hash, SRI)
- 10 min: Middleware and helpers
- 5 min: Tests
- 5 min: Documentation

## Current Status
Ready to implement final improvements and mark as complete.

## Next Steps After Import Maps
1. Clean up remaining TODOs in auth package
2. Polish component HTML parsing
3. Add hot reloading support
4. Reach 100% BDD scenario coverage
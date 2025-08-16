# Import Maps for Buffkit

Modern JavaScript module management for SSR-first Buffalo applications. This package provides native browser import maps support with vendoring, integrity checking, and development tools.

## Overview

Import maps allow you to control module resolution in the browser without a build step. This package provides:

- 🚀 **Zero Build Step** - Use ES modules directly in the browser
- 📦 **Vendoring Support** - Download and cache dependencies locally
- 🔒 **Subresource Integrity** - Automatic SRI hash generation
- 🛠️ **CLI Management** - Grift tasks for pin/unpin/vendor operations
- 🔄 **Hot Module Loading** - Development mode with automatic reloading
- 🎯 **CDN or Local** - Flexible sourcing of dependencies

## Installation

Import maps are included with Buffkit. Just wire them into your Buffalo app:

```go
import "github.com/johnjansen/buffkit/importmap"

func App() *buffalo.App {
    app := buffalo.New(buffalo.Options{
        Env: ENV,
    })

    // Create import map manager
    manager := importmap.NewManager()
    manager.LoadDefaults()

    // Add middleware
    app.Use(importmap.Middleware(manager))
    app.Use(importmap.DevModeMiddleware(manager))
    app.Use(importmap.VendorMiddleware(manager))

    return app
}
```

## Usage

### In Your Templates

Add these helpers to your application layout:

```html
<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <title>My App</title>
    
    <!-- Import maps and module entry point -->
    <%= raw(importMapTag()) %>
    <%= raw(moduleEntrypoint()) %>
</head>
<body>
    <%= yield %>
</body>
</html>
```

### Default Imports

Buffkit includes these imports by default:

- **htmx.org** - HTML over the wire
- **Alpine.js** - Minimal reactive framework
- **Stimulus** - Modest JavaScript framework
- **app** - Your application entry point
- **controllers/** - Controller modules directory

### CLI Commands

Manage your imports with Buffalo tasks:

```bash
# Initialize with defaults
buffalo task importmap:init

# Pin a new package
buffalo task importmap:pin lodash https://esm.sh/lodash
buffalo task importmap:pin dayjs https://esm.sh/dayjs

# Pin using default CDN (esm.sh)
buffalo task importmap:pin axios axios

# List all pinned packages
buffalo task importmap:list

# Vendor all remote packages locally
buffalo task importmap:vendor

# Update vendored packages
buffalo task importmap:update

# Remove a package
buffalo task importmap:unpin lodash

# Clean unused vendor files
buffalo task importmap:clean
```

### In Your JavaScript

Use imports naturally with the pinned names:

```javascript
// app/assets/js/index.js
import { format } from 'dayjs';
import axios from 'axios';
import _ from 'lodash';

// Your application code
console.log('App initialized at', format(new Date()));
```

### Custom Controllers

Create modular controllers using the controllers/ namespace:

```javascript
// app/assets/js/controllers/search_controller.js
export default class SearchController {
    constructor(element) {
        this.element = element;
        this.setup();
    }

    setup() {
        // Controller logic
    }
}

// Import in your app
import SearchController from 'controllers/search_controller.js';
```

## Configuration

### Custom Vendor Directory

```go
manager := importmap.NewManagerWithOptions(
    "public/vendor", // Custom vendor directory
    true,           // Development mode
)
```

### Loading from Configuration File

```go
// Load from config/importmap.json
manager.LoadFromFile("config/importmap.json")

// Save current configuration
manager.SaveToFile("config/importmap.json")
```

### Programmatic Management

```go
// Pin a package
manager.Pin("react", "https://esm.sh/react@18")

// Download and vendor a package
err := manager.Download("react")

// Get SRI hash for vendored package
integrity := manager.GetIntegrity("react")

// Update all vendored packages
err := manager.UpdateAll()

// Get import map as JSON
jsonData, err := manager.ToJSON()

// Render as HTML
html := manager.RenderHTML()
```

## Security Features

### Subresource Integrity (SRI)

All vendored files automatically get SRI hashes:

```html
<!-- Generated automatically -->
<script type="importmap">
{
  "imports": {
    "lodash": "/assets/vendor/lodash-a1b2c3d4.js"
  }
}
</script>
<!-- lodash integrity: sha256-RlN3DpDvB3tBvS... -->
```

### Content Security Policy

Compatible with strict CSP policies:

```go
// In your security middleware
c.Response().Header().Set("Content-Security-Policy", 
    "script-src 'self' https://esm.sh https://unpkg.com")
```

## Development Mode

In development, the manager provides:

- Debug logging in console
- No caching headers
- Verbose error messages
- Hot module loading support

```javascript
// Automatically injected in development
window.__BUFFKIT_DEV__ = true;
console.log('[Buffkit] Import maps loaded in development mode');
```

## Production Optimization

### Vendoring Strategy

For production, vendor all dependencies:

```bash
# Download all remote packages
buffalo task importmap:vendor

# Commit vendored files
git add public/assets/vendor/
git commit -m "Vendor JavaScript dependencies"
```

### Caching Headers

Vendored files automatically get immutable caching:

```
Cache-Control: public, max-age=31536000, immutable
```

### Preloading

Critical modules are automatically preloaded:

```html
<link rel="modulepreload" href="/assets/vendor/htmx-abc123.js">
<link rel="modulepreload" href="/assets/vendor/alpine-def456.js">
```

## Browser Support

Import maps are supported in all modern browsers:

- Chrome 89+
- Firefox 108+
- Safari 16.4+
- Edge 89+

For older browsers, use the [es-module-shims](https://github.com/guybedford/es-module-shims) polyfill:

```html
<script async src="https://ga.jspm.io/npm:es-module-shims@1.6.3/dist/es-module-shims.js"></script>
```

## Examples

### HTMX with Alpine.js

```html
<div x-data="{ open: false }">
    <button @click="open = !open" 
            hx-get="/api/data" 
            hx-target="#results">
        Load Data
    </button>
    
    <div x-show="open" id="results">
        <!-- HTMX will update this -->
    </div>
</div>
```

### Stimulus Controller

```javascript
// app/assets/js/controllers/hello_controller.js
import { Controller } from '@hotwired/stimulus';

export default class extends Controller {
    static targets = ["name", "output"];
    
    greet() {
        this.outputTarget.textContent = 
            `Hello, ${this.nameTarget.value}!`;
    }
}
```

### Dynamic Import

```javascript
// Lazy load modules
const loadChart = async () => {
    const { Chart } = await import('chart.js');
    return new Chart(ctx, config);
};
```

## Migration Guide

### From Webpack/esbuild

1. Remove build configuration files
2. Move entry points to `app/assets/js/`
3. Update imports to use pinned names
4. Run `buffalo task importmap:init`
5. Pin your dependencies

### From Rails Import Maps

The API is similar to Rails:

```ruby
# Rails
pin "lodash", to: "https://esm.sh/lodash"

# Buffkit (via CLI)
buffalo task importmap:pin lodash https://esm.sh/lodash
```

## Troubleshooting

### Import Not Found

```javascript
// Error: Failed to resolve module specifier "mylib"
```

Solution: Pin the library first:
```bash
buffalo task importmap:pin mylib https://esm.sh/mylib
```

### CORS Issues

If loading from CDN fails, vendor the package:
```bash
buffalo task importmap:vendor
```

### Cache Problems

Clear vendored files and re-download:
```bash
rm -rf public/assets/vendor/
buffalo task importmap:update
```

## Best Practices

1. **Vendor for Production** - Always vendor dependencies for production deployments
2. **Use SRI** - Let the manager generate integrity hashes automatically
3. **Pin Versions** - Use specific versions in URLs (e.g., `@1.2.3`)
4. **Minimize Scopes** - Use scopes sparingly for legacy code
5. **Test Locally** - Use `buffalo task importmap:vendor` in development too

## Contributing

See [CONTRIBUTING.md](../../CONTRIBUTING.md) for guidelines.

## License

Part of the Buffkit framework. See [LICENSE](../../LICENSE) for details.
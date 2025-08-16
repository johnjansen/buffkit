package importmap

import (
	"fmt"
	"html/template"
	"strings"

	"github.com/gobuffalo/buffalo"
)

// Middleware injects import maps into HTML responses
func Middleware(manager *Manager) buffalo.MiddlewareFunc {
	return func(next buffalo.Handler) buffalo.Handler {
		return func(c buffalo.Context) error {
			// Store manager in context for templates
			c.Set("importMapManager", manager)

			// Add template helpers
			c.Set("importMapTag", func() template.HTML {
				return template.HTML(manager.RenderHTML())
			})

			c.Set("moduleEntrypoint", func() template.HTML {
				return template.HTML(manager.RenderModuleEntrypoint())
			})

			// Call the next handler
			err := next(c)
			if err != nil {
				return err
			}

			// Only process HTML responses
			contentType := c.Response().Header().Get("Content-Type")
			if !strings.Contains(contentType, "text/html") {
				return nil
			}

			// Auto-inject import maps if enabled
			if manager.devMode || shouldAutoInject(c) {
				return injectImportMaps(c, manager)
			}

			return nil
		}
	}
}

// shouldAutoInject determines if import maps should be auto-injected
func shouldAutoInject(c buffalo.Context) bool {
	// Check if auto-injection is explicitly disabled
	if disabled, ok := c.Value("disableImportMapInjection").(bool); ok && disabled {
		return false
	}

	// Check if we're in an API route
	if strings.HasPrefix(c.Request().URL.Path, "/api/") {
		return false
	}

	// Check if we're in a partial render
	if c.Request().Header.Get("HX-Request") == "true" {
		return false
	}

	return true
}

// injectImportMaps injects the import map and module entrypoint into the HTML response
func injectImportMaps(c buffalo.Context, manager *Manager) error {
	// Buffalo doesn't expose the response body directly, so we can't modify it after the fact
	// Instead, we should set the import map in context for templates to use
	// This is a limitation of the current approach

	// For now, just ensure the manager is available in context
	c.Set("importMapHTML", template.HTML(manager.RenderHTML()))
	c.Set("moduleEntrypointHTML", template.HTML(manager.RenderModuleEntrypoint()))

	return nil
}

// DevModeMiddleware sets development mode based on environment
func DevModeMiddleware(manager *Manager) buffalo.MiddlewareFunc {
	return func(next buffalo.Handler) buffalo.Handler {
		return func(c buffalo.Context) error {
			app := c.Value("app").(*buffalo.App)
			if app != nil {
				manager.SetDevMode(app.Env == "development")
			}
			return next(c)
		}
	}
}

// VendorMiddleware serves vendored JavaScript files with proper caching headers
func VendorMiddleware(manager *Manager) buffalo.MiddlewareFunc {
	return func(next buffalo.Handler) buffalo.Handler {
		return func(c buffalo.Context) error {
			// Check if this is a vendor asset request
			if !strings.HasPrefix(c.Request().URL.Path, "/assets/vendor/") {
				return next(c)
			}

			// Extract the filename
			filename := strings.TrimPrefix(c.Request().URL.Path, "/assets/vendor/")

			// Look up integrity hash if available
			for name := range manager.List() {
				if strings.Contains(filename, sanitizeName(name)) {
					integrity := manager.GetIntegrity(name)
					if integrity != "" {
						c.Response().Header().Set("X-Content-Integrity", integrity)
					}
					break
				}
			}

			// Set caching headers for vendored files (they're content-hashed)
			if !manager.devMode {
				c.Response().Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			} else {
				c.Response().Header().Set("Cache-Control", "no-cache")
			}

			return next(c)
		}
	}
}

// PreloadMiddleware adds preload link headers for critical modules
func PreloadMiddleware(manager *Manager) buffalo.MiddlewareFunc {
	return func(next buffalo.Handler) buffalo.Handler {
		return func(c buffalo.Context) error {
			// Add preload headers for critical modules
			criticalModules := []string{"htmx.org", "alpinejs", "app"}

			for _, module := range criticalModules {
				if url, exists := manager.imports[module]; exists {
					// Add preload link header
					link := fmt.Sprintf(`<%s>; rel="modulepreload"`, url)
					c.Response().Header().Add("Link", link)

					// Add integrity if available
					if integrity := manager.GetIntegrity(module); integrity != "" {
						link = fmt.Sprintf(`<%s>; rel="modulepreload"; integrity="%s"`, url, integrity)
						c.Response().Header().Set("Link", link)
					}
				}
			}

			return next(c)
		}
	}
}

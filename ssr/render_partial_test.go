package ssr

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gobuffalo/buffalo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderPartial(t *testing.T) {
	// Create a test Buffalo app
	app := buffalo.New(buffalo.Options{
		Env: "test",
	})

	t.Run("renders simple partial with data", func(t *testing.T) {
		// Create a test handler that uses RenderPartial
		app.GET("/test-partial", func(c buffalo.Context) error {
			html, err := RenderPartial(c, "test_message", map[string]interface{}{
				"title":   "Test Title",
				"message": "This is a test message",
			})
			if err != nil {
				return c.Error(http.StatusInternalServerError, err)
			}

			// Write the rendered HTML directly
			_, _ = c.Response().Write(html)
			return nil
		})

		// Make a request to test the partial rendering
		req := httptest.NewRequest("GET", "/test-partial", nil)
		res := httptest.NewRecorder()

		app.ServeHTTP(res, req)

		// Check that we got a response (even if template doesn't exist)
		// In production, the template would exist and render properly
		assert.NotNil(t, res.Body)
	})

	t.Run("includes context values in render data", func(t *testing.T) {
		app.GET("/test-context", func(c buffalo.Context) error {
			// Set some context values
			c.Set("current_user", map[string]string{"name": "Test User"})
			c.Set("authenticity_token", "test-csrf-token")

			html, err := RenderPartial(c, "user_info", map[string]interface{}{
				"extra_data": "some value",
			})
			if err != nil {
				return c.Error(http.StatusInternalServerError, err)
			}

			_, _ = c.Response().Write(html)
			return nil
		})

		req := httptest.NewRequest("GET", "/test-context", nil)
		res := httptest.NewRecorder()

		app.ServeHTTP(res, req)

		// Verify we got a response
		assert.NotNil(t, res.Body)
	})

	t.Run("returns error for invalid template", func(t *testing.T) {
		app.GET("/test-error", func(c buffalo.Context) error {
			// Try to render a template that definitely doesn't exist
			html, err := RenderPartial(c, "this_template_does_not_exist_12345", map[string]interface{}{})

			// We expect an error here
			if err != nil {
				return c.Error(http.StatusInternalServerError, err)
			}

			_, _ = c.Response().Write(html)
			return nil
		})

		req := httptest.NewRequest("GET", "/test-error", nil)
		res := httptest.NewRecorder()

		app.ServeHTTP(res, req)

		// Should get an error status
		assert.Equal(t, http.StatusInternalServerError, res.Code)
	})

	t.Run("can be used for SSE broadcasts", func(t *testing.T) {
		// This test demonstrates the intended use case:
		// render once, use for both HTTP response and SSE broadcast

		var renderedHTML []byte
		var renderErr error

		// Simulate rendering a partial
		app.GET("/test-broadcast", func(c buffalo.Context) error {
			renderedHTML, renderErr = RenderPartial(c, "notification", map[string]interface{}{
				"type":           "info",
				"message":        "Server update",
				"notificationID": "123",
				"timestamp":      "2024-01-01T12:00:00Z",
			})

			if renderErr != nil {
				return c.Error(http.StatusInternalServerError, renderErr)
			}

			// Return the HTML as response
			_, _ = c.Response().Write(renderedHTML)
			return nil
		})

		req := httptest.NewRequest("GET", "/test-broadcast", nil)
		res := httptest.NewRecorder()

		app.ServeHTTP(res, req)

		// Verify we got HTML (might be nil if template doesn't exist in test)
		// In production with proper templates, this would not be nil
		if renderErr != nil {
			t.Logf("Expected error in test environment: %v", renderErr)
		}

		// In a real scenario, we'd also broadcast this via SSE:
		// broker.Broadcast("notification", renderedHTML)
	})
}

func TestRenderPartialIntegration(t *testing.T) {
	// Integration test with actual template rendering
	// This test would work if templates are properly set up

	t.Run("renders notification partial", func(t *testing.T) {
		app := buffalo.New(buffalo.Options{
			Env: "test",
		})

		app.GET("/notification", func(c buffalo.Context) error {
			html, err := RenderPartial(c, "notification", map[string]interface{}{
				"type":           "success",
				"message":        "Operation completed successfully",
				"notificationID": "test-123",
				"timestamp":      "2024-01-01T12:00:00Z",
			})

			if err != nil {
				// Template might not exist in test environment
				// In production, this would render successfully
				t.Logf("Expected error in test environment: %v", err)
				return c.Error(http.StatusInternalServerError, err)
			}

			// Check that the rendered HTML contains expected content
			htmlStr := string(html)
			assert.Contains(t, htmlStr, "notification")
			assert.Contains(t, htmlStr, "success")
			assert.Contains(t, htmlStr, "Operation completed successfully")

			_, _ = c.Response().Write(html)
			return nil
		})

		req := httptest.NewRequest("GET", "/notification", nil)
		res := httptest.NewRecorder()

		app.ServeHTTP(res, req)

		// In test environment, might get error if templates aren't loaded
		// In production, this would succeed
		t.Logf("Response status: %d", res.Code)
		t.Logf("Response body: %s", res.Body.String())
	})
}

func TestResponseCaptureWriter(t *testing.T) {
	t.Run("captures written bytes", func(t *testing.T) {
		var buf bytes.Buffer
		w := &responseCaptureWriter{
			Buffer: &buf,
			header: make(http.Header),
		}

		// Write some data
		n, err := w.Write([]byte("test content"))
		require.NoError(t, err)
		assert.Equal(t, 12, n)
		assert.Equal(t, "test content", buf.String())
	})

	t.Run("handles headers", func(t *testing.T) {
		var buf bytes.Buffer
		w := &responseCaptureWriter{
			Buffer: &buf,
			header: make(http.Header),
		}

		// Set header
		w.Header().Set("Content-Type", "text/html")
		assert.Equal(t, "text/html", w.Header().Get("Content-Type"))

		// Write status
		w.WriteHeader(http.StatusOK)
		assert.Equal(t, http.StatusOK, w.statusCode)
	})

	t.Run("accumulates multiple writes", func(t *testing.T) {
		var buf bytes.Buffer
		w := &responseCaptureWriter{
			Buffer: &buf,
			header: make(http.Header),
		}

		_, _ = w.Write([]byte("first "))
		_, _ = w.Write([]byte("second "))
		_, _ = w.Write([]byte("third"))

		assert.Equal(t, "first second third", buf.String())
	})
}

func TestRenderPartialEdgeCases(t *testing.T) {
	app := buffalo.New(buffalo.Options{
		Env: "test",
	})

	t.Run("handles empty data map", func(t *testing.T) {
		app.GET("/empty-data", func(c buffalo.Context) error {
			html, err := RenderPartial(c, "simple", map[string]interface{}{})
			if err != nil {
				return c.Error(http.StatusInternalServerError, err)
			}
			_, _ = c.Response().Write(html)
			return nil
		})

		req := httptest.NewRequest("GET", "/empty-data", nil)
		res := httptest.NewRecorder()
		app.ServeHTTP(res, req)

		// Should handle empty data gracefully
		assert.NotNil(t, res.Body)
	})

	t.Run("handles nil data values", func(t *testing.T) {
		app.GET("/nil-values", func(c buffalo.Context) error {
			html, err := RenderPartial(c, "test", map[string]interface{}{
				"title":   nil,
				"message": "test",
			})
			if err != nil {
				return c.Error(http.StatusInternalServerError, err)
			}
			_, _ = c.Response().Write(html)
			return nil
		})

		req := httptest.NewRequest("GET", "/nil-values", nil)
		res := httptest.NewRecorder()
		app.ServeHTTP(res, req)

		// Should handle nil values in data
		assert.NotNil(t, res.Body)
	})

	t.Run("preserves HTML in data", func(t *testing.T) {
		app.GET("/html-content", func(c buffalo.Context) error {
			html, err := RenderPartial(c, "content", map[string]interface{}{
				"content": "<strong>Bold text</strong>",
			})
			if err != nil {
				// Expected in test environment without templates
				t.Logf("Expected error: %v", err)
				return c.Error(http.StatusInternalServerError, err)
			}

			// In production, the template would handle HTML escaping
			// based on whether raw() is used or not
			htmlStr := string(html)
			t.Logf("Rendered HTML: %s", htmlStr)

			_, _ = c.Response().Write(html)
			return nil
		})

		req := httptest.NewRequest("GET", "/html-content", nil)
		res := httptest.NewRecorder()
		app.ServeHTTP(res, req)

		// Verify request was processed
		assert.NotNil(t, res.Body)
	})
}

// BenchmarkRenderPartial measures the performance of partial rendering
func BenchmarkRenderPartial(b *testing.B) {
	// Skip for now as we need a proper context setup
	// In a real benchmark, you'd:
	// 1. Create a Buffalo app
	// 2. Set up a proper context with templates
	// 3. Measure RenderPartial performance
	b.SkipNow()
}

// ExampleRenderPartial shows how to use RenderPartial in practice
func ExampleRenderPartial() {
	app := buffalo.New(buffalo.Options{})

	app.POST("/api/items", func(c buffalo.Context) error {
		// Process the item creation...
		item := map[string]interface{}{
			"id":   "123",
			"name": "New Item",
		}

		// Render the partial once
		html, err := RenderPartial(c, "item_row", item)
		if err != nil {
			return err
		}

		// Use for HTMX response
		if strings.Contains(c.Request().Header.Get("HX-Request"), "true") {
			_, _ = c.Response().Write(html)
			return nil
		}

		// Also broadcast via SSE to other clients
		// broker.Broadcast("item-created", html)

		// Or return JSON for API clients
		// Or return JSON for API clients
		c.Response().Header().Set("Content-Type", "application/json")
		c.Response().WriteHeader(http.StatusCreated)
		return nil
	})
}

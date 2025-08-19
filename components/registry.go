// Package components provides server-side custom elements for Buffalo applications.
// It implements a component system similar to Web Components but rendered server-side,
// allowing you to create reusable UI components with a custom tag syntax.
//
// Components are defined as <bk-*> tags in your HTML templates and are expanded
// server-side before sending to the client. This provides the benefits of components
// (reusability, encapsulation, composition) without requiring JavaScript.
//
// Example usage in templates:
//
//	<bk-button variant="primary" href="/save">Save Changes</bk-button>
//
//	<bk-card>
//	    <bk-slot name="header">Card Title</bk-slot>
//	    <p>Card content goes here</p>
//	</bk-card>
//
// The system supports:
//   - Custom attributes passed to components
//   - Named slots for content distribution
//   - Nested components
//   - Shadowing (apps can override built-in components)
//
// Components are registered with the Registry and expanded by middleware
// that processes HTML responses before sending to clients.
package components

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"

	"github.com/gobuffalo/buffalo"
	"golang.org/x/net/html"
)

// Renderer is a function that renders a component.
// It receives attributes from the component tag and any slot content,
// then returns the expanded HTML.
//
// Example renderer:
//
//	func renderButton(attrs map[string]string, slots map[string]string) ([]byte, error) {
//	    variant := attrs["variant"] // Get variant attribute
//	    content := slots["default"]  // Get default slot content
//	    return []byte(fmt.Sprintf(`<button class="btn-%s">%s</button>`, variant, content)), nil
//	}
//
// WHY: This signature allows components to be pure functions that transform
// attributes and content into HTML, making them easy to test and reason about.
type Renderer func(attrs map[string]string, slots map[string]string) ([]byte, error)

// Registry manages server-side components.
// It's the central repository for all registered components in the application.
// Components are registered by name (e.g., "bk-button") with their renderer function.
//
// The registry is used by the expansion middleware to look up and render components
// when processing HTML responses.
type Registry struct {
	// components maps component names to their renderer functions.
	// Names should follow the pattern "bk-*" to avoid conflicts with HTML elements.
	components map[string]Renderer
}

// NewRegistry creates a new component registry.
// This is typically called once during app initialization:
//
//	registry := components.NewRegistry()
//	registry.RegisterDefaults()
//	app.Use(components.ExpanderMiddleware(registry))
func NewRegistry() *Registry {
	return &Registry{
		components: make(map[string]Renderer),
	}
}

// Register adds a component to the registry.
// The name should follow the pattern "bk-*" to clearly identify it as a Buffkit component.
//
// Example:
//
//	registry.Register("bk-avatar", func(attrs, slots map[string]string) ([]byte, error) {
//	    user := attrs["user"]
//	    size := attrs["size"]
//	    return []byte(fmt.Sprintf(`<img class="avatar-%s" src="/avatars/%s.jpg">`, size, user)), nil
//	})
//
// Components can be overridden by registering a new renderer with the same name.
// This allows apps to customize built-in components.
func (r *Registry) Register(name string, renderer Renderer) {
	r.components[name] = renderer
}

// RegisterDefaults is deprecated and does nothing.
// Apps should register their own components using Register().
//
// Example:
//
//	registry.Register("my-button", func(attrs map[string]string, slots map[string]string) ([]byte, error) {
//	    // Your component rendering logic here
//	    return []byte("<button>" + slots["default"] + "</button>"), nil
//	})
func (r *Registry) RegisterDefaults() {
	// No default components - apps define their own
}

// Render renders a component by name.
// This looks up the component's renderer and calls it with the provided
// attributes and slots.
//
// If the component doesn't exist, an error is returned and the original
// tag is preserved in the HTML (graceful degradation).
//
// This method is called by the expansion middleware when it encounters
// a <bk-*> tag in the HTML.
func (r *Registry) Render(name string, attrs map[string]string, slots map[string]string) ([]byte, error) {
	renderer, exists := r.components[name]
	if !exists {
		// Return error so the original tag is preserved
		// This allows graceful degradation if a component isn't registered
		return nil, fmt.Errorf("component %s not found", name)
	}

	return renderer(attrs, slots)
}

// ExpanderMiddleware returns middleware that expands server-side components.
// This middleware intercepts HTML responses and processes any <bk-*> tags,
// replacing them with their rendered HTML before sending to the client.
//
// How it works:
//  1. Wraps the response writer to capture the HTML output
//  2. Lets the handler generate its response
//  3. If response is HTML, parses it and expands components
//  4. Writes the expanded HTML to the real response writer
//
// The middleware only processes text/html responses to avoid breaking
// JSON APIs, file downloads, etc.
//
// When devMode is true, component boundary comments are added to help
// with debugging (e.g., <!-- bk-button --> ... <!-- /bk-button -->).
//
// Usage:
//
//	app.Use(components.ExpanderMiddleware(registry, devMode))
//
// WHY middleware: This approach allows components to work transparently
// with any template engine or HTML generation method. Templates don't need
// to know about component expansion - they just write <bk-*> tags.
func ExpanderMiddleware(registry *Registry, devMode bool) buffalo.MiddlewareFunc {
	return func(next buffalo.Handler) buffalo.Handler {
		return func(c buffalo.Context) error {
			// Create a response wrapper to capture output.
			// We need to buffer the response so we can process it
			// before sending to the client.
			wrapper := &responseWrapper{
				ResponseWriter: c.Response(),
				body:           &bytes.Buffer{},
				statusCode:     http.StatusOK,
			}

			// Replace response writer with our wrapper
			oldWriter := c.Response()
			c.Set("res", wrapper)

			// Call the actual handler
			err := next(c)

			// Restore original writer for final output
			c.Set("res", oldWriter)

			if err != nil {
				return err
			}

			// Only process HTML responses.
			// Skip JSON, images, downloads, etc.
			contentType := wrapper.Header().Get("Content-Type")
			if !strings.Contains(contentType, "text/html") {
				// Write original content unchanged
				oldWriter.WriteHeader(wrapper.statusCode)
				_, writeErr := oldWriter.Write(wrapper.body.Bytes())
				return writeErr
			}

			// Expand components in the captured HTML
			expanded, err := expandComponents(wrapper.body.Bytes(), registry, devMode)
			if err != nil {
				// On error, send original HTML
				// Better to show unexpanded components than error page
				oldWriter.WriteHeader(wrapper.statusCode)
				_, writeErr := oldWriter.Write(wrapper.body.Bytes())
				return writeErr
			}

			// Write the expanded HTML to the client
			oldWriter.WriteHeader(wrapper.statusCode)
			_, err = oldWriter.Write(expanded)
			return err
		}
	}
}

// expandComponents expands all <bk-*> tags in HTML.
// This function parses the HTML, finds all component tags, and replaces them
// with their rendered output.
//
// The process:
//  1. Parse HTML into a DOM tree
//  2. Walk the tree looking for <bk-*> elements
//  3. Extract attributes and slot content from each component
//  4. Call the component's renderer
//  5. Replace the component tag with rendered HTML
//  6. Serialize the modified tree back to HTML
//
// Components can be nested - inner components are expanded first.
// If a component fails to render, it's left unchanged (graceful degradation).
//
// TODO: This is a simplified implementation. Production version should:
//   - Handle component recursion limits
//   - Preserve HTML comments and doctype
//   - Optimize for large documents
func expandComponents(htmlContent []byte, registry *Registry, devMode bool) ([]byte, error) {
	doc, err := html.Parse(bytes.NewReader(htmlContent))
	if err != nil {
		return htmlContent, err
	}

	// Walk the tree and expand components.
	// This is a recursive function that processes nodes depth-first.
	var expand func(*html.Node) error
	expand = func(n *html.Node) error {
		if n.Type == html.ElementNode && strings.HasPrefix(n.Data, "bk-") {
			// Found a component tag - extract its data
			componentName := n.Data

			// Extract attributes from the component tag
			attrs := make(map[string]string)
			for _, attr := range n.Attr {
				attrs[attr.Key] = attr.Val
			}

			// Extract slot content (named and default slots)
			slots := extractSlots(n)

			// Render the component
			rendered, err := registry.Render(n.Data, attrs, slots)
			if err != nil {
				// Keep original tag if rendering fails
				// This allows the page to still work even if a component breaks
				return nil
			}

			// Parse the rendered HTML fragment
			renderedDoc, err := html.ParseFragment(bytes.NewReader(rendered), &html.Node{
				Type: html.ElementNode,
				Data: "div",
			})
			if err != nil {
				return nil
			}

			// Add component boundary comments in development mode
			if devMode {
				// Add start comment
				startComment := &html.Node{
					Type: html.CommentNode,
					Data: fmt.Sprintf(" %s ", componentName),
				}
				n.Parent.InsertBefore(startComment, n)
			}

			// Replace the component node with rendered nodes
			for _, newNode := range renderedDoc {
				n.Parent.InsertBefore(newNode, n)
			}

			// Add end comment in development mode
			if devMode {
				endComment := &html.Node{
					Type: html.CommentNode,
					Data: fmt.Sprintf(" /%s ", componentName),
				}
				n.Parent.InsertBefore(endComment, n)
			}

			n.Parent.RemoveChild(n)

			return nil
		}

		// Not a component - recurse to children
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if err := expand(c); err != nil {
				return err
			}
		}

		return nil
	}

	if err := expand(doc); err != nil {
		return htmlContent, err
	}

	// Render the modified tree back to HTML
	var buf bytes.Buffer
	if err := html.Render(&buf, doc); err != nil {
		return htmlContent, err
	}

	return buf.Bytes(), nil
}

// extractSlots extracts named slots from a component node.
// Slots allow components to accept content in specific locations,
// similar to Vue.js or Web Components slots.
//
// Example component usage:
//
//	<bk-card>
//	    <bk-slot name="header">Card Title</bk-slot>
//	    <p>This goes in default slot</p>
//	    <bk-slot name="footer">Card Footer</bk-slot>
//	</bk-card>
//
// This would produce:
//
//	slots["header"] = "Card Title"
//	slots["default"] = "<p>This goes in default slot</p>"
//	slots["footer"] = "Card Footer"
//
// The component renderer can then place this content appropriately.
func extractSlots(n *html.Node) map[string]string {
	slots := make(map[string]string)
	var defaultSlot bytes.Buffer

	// Iterate through the component's children
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == "bk-slot" {
			// This is a named slot - extract its name
			slotName := "default"
			for _, attr := range c.Attr {
				if attr.Key == "name" {
					slotName = attr.Val
					break
				}
			}

			// Extract the slot's content
			var slotBuf bytes.Buffer
			for sc := c.FirstChild; sc != nil; sc = sc.NextSibling {
				_ = html.Render(&slotBuf, sc)
			}
			slots[slotName] = slotBuf.String()
		} else {
			// Not a slot - this goes in the default slot
			_ = html.Render(&defaultSlot, c)
		}
	}

	// Set default slot if it has content
	if defaultSlot.Len() > 0 {
		slots["default"] = defaultSlot.String()
	}

	return slots
}

// responseWrapper captures response for processing.
// This allows the middleware to buffer the entire response before
// processing it for component expansion.
//
// WHY: We need the complete HTML document before we can parse and
// modify it. This wrapper intercepts writes and stores them.
type responseWrapper struct {
	http.ResponseWriter               // Embed the original ResponseWriter
	body                *bytes.Buffer // Buffer to capture response body
	statusCode          int           // HTTP status code to preserve
}

func (w *responseWrapper) WriteHeader(statusCode int) {
	w.statusCode = statusCode
}

func (w *responseWrapper) Write(b []byte) (int, error) {
	return w.body.Write(b)
}

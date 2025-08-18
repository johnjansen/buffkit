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
	"strconv"
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

// RegisterDefaults registers the default Buffkit components.
// These provide a base set of UI components that apps can use immediately:
//   - bk-button: Styled buttons with variants
//   - bk-card: Card containers with header/footer slots
//   - bk-dropdown: Dropdown menus with Alpine.js integration
//   - bk-alert: Alert messages with variants and dismissible option
//   - bk-form: Forms with automatic CSRF token
//   - bk-input: Form inputs with labels and validation
//
// All default components can be overridden by the app by registering
// a new renderer with the same name after calling RegisterDefaults.
func (r *Registry) RegisterDefaults() {
	// Register button component
	r.Register("bk-button", renderButton)

	// Register card component
	r.Register("bk-card", renderCard)

	// Register dropdown component
	r.Register("bk-dropdown", renderDropdown)

	// Register alert component
	r.Register("bk-alert", renderAlert)

	// Register form components
	r.Register("bk-form", renderForm)
	r.Register("bk-input", renderInput)

	// Register text component (safe HTML escaping)
	r.Register("bk-text", renderText)

	// Register modal component
	r.Register("bk-modal", renderModal)

	// Register tabs component
	r.Register("bk-tabs", renderTabs)

	// Register code component
	r.Register("bk-code", renderCode)

	// Register icon component
	r.Register("bk-icon", renderIcon)

	// Register avatar component
	r.Register("bk-avatar", renderAvatar)

	// Register feature flag component
	r.Register("bk-feature", renderFeatureFlag)

	// Register progress bar component
	r.Register("bk-progress-bar", renderProgressBar)

	// Register form field component
	r.Register("bk-form-field", renderFormField)
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

// Default component renderers
// These functions implement the built-in Buffkit components.
// Each follows the Renderer signature and transforms attributes/slots into HTML.

// renderButton renders the bk-button component.
// Supports variants (primary, danger, etc.) and can render as either
// a <button> or <a> tag depending on whether href is provided.
//
// Attributes:
//   - variant: Visual style (primary, danger, warning, success, default)
//   - href: If provided, renders as a link instead of button
//   - class: Additional CSS classes to apply
//
// Example:
//
//	<bk-button variant="primary" href="/save">Save</bk-button>
//	Renders: <a href="/save" class="bk-button bk-button-primary">Save</a>
func renderButton(attrs map[string]string, slots map[string]string) ([]byte, error) {
	variant := attrs["variant"]
	if variant == "" {
		variant = "default"
	}

	size := attrs["size"]
	href := attrs["href"]
	class := attrs["class"]
	content := slots["default"]

	// Build class list
	classes := []string{"bk-button", "bk-button-" + variant}
	if size != "" {
		classes = append(classes, "bk-button-"+size)
	}
	if class != "" {
		classes = append(classes, class)
	}

	// Build attributes string for additional attributes
	var attrStr strings.Builder
	for key, value := range attrs {
		// Skip attributes we've already handled
		if key == "variant" || key == "size" || key == "class" || key == "href" {
			continue
		}
		// Skip onclick and other potentially dangerous attributes
		if strings.HasPrefix(key, "on") {
			continue
		}
		attrStr.WriteString(` `)
		attrStr.WriteString(html.EscapeString(key))
		attrStr.WriteString(`="`)
		attrStr.WriteString(html.EscapeString(value))
		attrStr.WriteString(`"`)
	}

	// Render as link or button based on href presence
	if href != "" {
		return []byte(fmt.Sprintf(
			`<a href="%s" class="%s"%s>%s</a>`,
			html.EscapeString(href), strings.Join(classes, " "), attrStr.String(), content,
		)), nil
	}

	return []byte(fmt.Sprintf(
		`<button class="%s"%s>%s</button>`,
		strings.Join(classes, " "), attrStr.String(), content,
	)), nil
}

// renderCard renders the bk-card component.
// A card is a flexible container with optional header and footer slots.
//
// Slots:
//   - header: Content for the card header
//   - default: Main card body content
//   - footer: Content for the card footer
//
// Example:
//
//	<bk-card>
//	  <bk-slot name="header">User Profile</bk-slot>
//	  <p>User details here</p>
//	  <bk-slot name="footer">Last updated: today</bk-slot>
//	</bk-card>
func renderCard(attrs map[string]string, slots map[string]string) ([]byte, error) {
	class := attrs["class"]
	header := slots["header"]
	footer := slots["footer"]
	content := slots["default"]

	var buf bytes.Buffer
	buf.WriteString(`<div class="bk-card`)
	if class != "" {
		buf.WriteString(" " + class)
	}
	buf.WriteString(`">`)

	if header != "" {
		buf.WriteString(`<div class="bk-card-header">`)
		buf.WriteString(header)
		buf.WriteString(`</div>`)
	}

	buf.WriteString(`<div class="bk-card-body">`)
	buf.WriteString(content)
	buf.WriteString(`</div>`)

	if footer != "" {
		buf.WriteString(`<div class="bk-card-footer">`)
		buf.WriteString(footer)
		buf.WriteString(`</div>`)
	}

	buf.WriteString(`</div>`)

	return buf.Bytes(), nil
}

func renderDropdown(attrs map[string]string, slots map[string]string) ([]byte, error) {
	class := attrs["class"]
	trigger := slots["trigger"]
	if trigger == "" {
		trigger = "Menu"
	}
	content := slots["default"]

	// Generate unique ID
	id := fmt.Sprintf("dropdown-%d", hashString(trigger+content))

	var buf bytes.Buffer
	buf.WriteString(`<div class="bk-dropdown`)
	if class != "" {
		buf.WriteString(" " + class)
	}
	buf.WriteString(`"`)

	// Add data-component attribute
	buf.WriteString(` data-component="dropdown"`)
	buf.WriteString(` data-state="closed"`)

	// Add x-data if not already present
	if xData, ok := attrs["x-data"]; ok {
		buf.WriteString(` x-data="`)
		buf.WriteString(html.EscapeString(xData))
		buf.WriteString(`"`)
	} else {
		buf.WriteString(` x-data="{ open: false }"`)
	}

	// Add any other data attributes
	for key, value := range attrs {
		if key != "class" && key != "x-data" {
			// Preserve all data attributes and other safe attributes
			if strings.HasPrefix(key, "data-") || strings.HasPrefix(key, "aria-") || key == "id" {
				buf.WriteString(` `)
				buf.WriteString(html.EscapeString(key))
				buf.WriteString(`="`)
				buf.WriteString(html.EscapeString(value))
				buf.WriteString(`"`)
			}
		}
	}

	buf.WriteString(`>`)

	// Trigger button
	buf.WriteString(fmt.Sprintf(
		`<button class="bk-dropdown-trigger" @click="open = !open" aria-haspopup="true" aria-expanded="false" :aria-expanded="open">%s</button>`,
		trigger,
	))

	// Dropdown menu
	buf.WriteString(fmt.Sprintf(
		`<div class="bk-dropdown-menu" x-show="open" @click.away="open = false" x-transition id="%s">`,
		id,
	))
	buf.WriteString(content)
	buf.WriteString(`</div>`)

	buf.WriteString(`</div>`)

	return buf.Bytes(), nil
}

func renderAlert(attrs map[string]string, slots map[string]string) ([]byte, error) {
	variant := attrs["variant"]
	if variant == "" {
		variant = "info"
	}
	dismissible := attrs["dismissible"] == "true"
	class := attrs["class"]
	content := slots["default"]

	var buf bytes.Buffer
	buf.WriteString(`<div class="bk-alert bk-alert-` + variant)
	if class != "" {
		buf.WriteString(" " + class)
	}
	buf.WriteString(`"`)

	if dismissible {
		buf.WriteString(` x-data="{ show: true }" x-show="show"`)
	}
	buf.WriteString(`>`)

	buf.WriteString(content)

	if dismissible {
		buf.WriteString(`<button type="button" class="bk-alert-close" @click="show = false">&times;</button>`)
	}

	buf.WriteString(`</div>`)

	return buf.Bytes(), nil
}

func renderForm(attrs map[string]string, slots map[string]string) ([]byte, error) {
	action := attrs["action"]
	method := attrs["method"]
	if method == "" {
		method = "POST"
	}
	class := attrs["class"]
	content := slots["default"]

	var buf bytes.Buffer
	buf.WriteString(`<form`)
	if action != "" {
		buf.WriteString(fmt.Sprintf(` action="%s"`, action))
	}
	buf.WriteString(fmt.Sprintf(` method="%s"`, method))
	if class != "" {
		buf.WriteString(fmt.Sprintf(` class="bk-form %s"`, class))
	} else {
		buf.WriteString(` class="bk-form"`)
	}
	buf.WriteString(`>`)

	// Add CSRF token for non-GET forms
	if method != "GET" {
		buf.WriteString(`<input type="hidden" name="authenticity_token" value="{{ .authenticity_token }}">`)
	}

	buf.WriteString(content)
	buf.WriteString(`</form>`)

	return buf.Bytes(), nil
}

func renderInput(attrs map[string]string, slots map[string]string) ([]byte, error) {
	inputType := attrs["type"]
	if inputType == "" {
		inputType = "text"
	}
	name := attrs["name"]
	label := attrs["label"]
	placeholder := attrs["placeholder"]
	// Boolean attributes in HTML are present without a value or with any value
	_, required := attrs["required"]
	_, disabled := attrs["disabled"]
	_, readonly := attrs["readonly"]
	_, checked := attrs["checked"]
	class := attrs["class"]
	value := attrs["value"]

	var buf bytes.Buffer
	buf.WriteString(`<div class="bk-input-group">`)

	if label != "" {
		buf.WriteString(fmt.Sprintf(`<label for="%s" class="bk-label">%s`, name, label))
		if required {
			buf.WriteString(` <span class="bk-required">*</span>`)
		}
		buf.WriteString(`</label>`)
	}

	buf.WriteString(`<input`)
	buf.WriteString(fmt.Sprintf(` type="%s"`, inputType))
	buf.WriteString(fmt.Sprintf(` id="%s"`, name))
	buf.WriteString(fmt.Sprintf(` name="%s"`, name))

	// Add aria-label for accessibility
	if label != "" {
		buf.WriteString(fmt.Sprintf(` aria-label="%s"`, html.EscapeString(label)))
	} else if placeholder != "" {
		buf.WriteString(fmt.Sprintf(` aria-label="%s"`, html.EscapeString(placeholder)))
	} else if name != "" {
		// Generate aria-label from name if no label or placeholder
		// Convert snake_case or kebab-case to readable format
		ariaLabel := strings.ReplaceAll(name, "_", " ")
		ariaLabel = strings.ReplaceAll(ariaLabel, "-", " ")
		buf.WriteString(fmt.Sprintf(` aria-label="%s"`, html.EscapeString(ariaLabel)))
	} else {
		// Fall back to input type
		buf.WriteString(fmt.Sprintf(` aria-label="%s input"`, inputType))
	}

	if placeholder != "" {
		buf.WriteString(fmt.Sprintf(` placeholder="%s"`, placeholder))
	}
	if value != "" {
		buf.WriteString(fmt.Sprintf(` value="%s"`, value))
	}
	if required {
		buf.WriteString(` required`)
	}
	if disabled {
		buf.WriteString(` disabled`)
	}
	if readonly {
		buf.WriteString(` readonly`)
	}
	if checked && (inputType == "checkbox" || inputType == "radio") {
		buf.WriteString(` checked`)
	}
	if class != "" {
		buf.WriteString(fmt.Sprintf(` class="bk-input %s"`, class))
	} else {
		buf.WriteString(` class="bk-input"`)
	}
	buf.WriteString(`>`)

	buf.WriteString(`</div>`)

	return buf.Bytes(), nil
}

// renderText renders a text component with safe HTML escaping.
// This component escapes all HTML to prevent XSS attacks.
func renderText(attrs map[string]string, slots map[string]string) ([]byte, error) {
	var buf strings.Builder

	buf.WriteString(`<span class="bk-text"`)

	// Add any custom attributes
	for key, value := range attrs {
		if key != "class" {
			buf.WriteString(` `)
			buf.WriteString(html.EscapeString(key))
			buf.WriteString(`="`)
			buf.WriteString(html.EscapeString(value))
			buf.WriteString(`"`)
		}
	}

	buf.WriteString(`>`)
	// Escape the content to prevent XSS
	buf.WriteString(html.EscapeString(slots["default"]))
	buf.WriteString(`</span>`)

	return []byte(buf.String()), nil
}

// renderModal renders a modal dialog component with ARIA attributes.
func renderModal(attrs map[string]string, slots map[string]string) ([]byte, error) {
	var buf strings.Builder

	title := attrs["title"]
	if title == "" {
		title = "Modal"
	}

	modalID := fmt.Sprintf("modal-%d", hashString(title))
	titleID := fmt.Sprintf("modal-title-%d", hashString(title))

	buf.WriteString(`<div class="bk-modal"`)

	// Add ARIA attributes for accessibility
	buf.WriteString(` role="dialog"`)
	buf.WriteString(` aria-modal="true"`)
	buf.WriteString(` aria-labelledby="`)
	buf.WriteString(titleID)
	buf.WriteString(`"`)

	// Add other attributes
	for key, value := range attrs {
		if key != "title" && key != "class" {
			buf.WriteString(` `)
			buf.WriteString(html.EscapeString(key))
			buf.WriteString(`="`)
			buf.WriteString(html.EscapeString(value))
			buf.WriteString(`"`)
		}
	}

	buf.WriteString(` id="`)
	buf.WriteString(modalID)
	buf.WriteString(`">`)

	// Modal header
	buf.WriteString(`<div class="bk-modal-header">`)
	buf.WriteString(`<h2 id="`)
	buf.WriteString(titleID)
	buf.WriteString(`">`)
	buf.WriteString(html.EscapeString(title))
	buf.WriteString(`</h2>`)

	if headerSlot, ok := slots["header"]; ok {
		buf.WriteString(headerSlot)
	}

	buf.WriteString(`</div>`)

	// Modal body
	buf.WriteString(`<div class="bk-modal-body">`)
	buf.WriteString(slots["default"])
	if bodySlot, ok := slots["body"]; ok {
		buf.WriteString(bodySlot)
	}
	buf.WriteString(`</div>`)

	// Modal footer (if provided)
	if footerSlot, ok := slots["footer"]; ok {
		buf.WriteString(`<div class="bk-modal-footer">`)
		buf.WriteString(footerSlot)
		buf.WriteString(`</div>`)
	}

	buf.WriteString(`</div>`)

	return []byte(buf.String()), nil
}

// renderTabs renders a tabs component with state management attributes.
func renderTabs(attrs map[string]string, slots map[string]string) ([]byte, error) {
	var buf strings.Builder

	defaultTab := attrs["default-tab"]
	if defaultTab == "" {
		defaultTab = "1"
	}

	buf.WriteString(`<div class="bk-tabs"`)
	buf.WriteString(` data-component="tabs"`)
	buf.WriteString(` data-initial-tab="`)
	buf.WriteString(html.EscapeString(defaultTab))
	buf.WriteString(`"`)

	// Add other attributes
	for key, value := range attrs {
		if key != "default-tab" && key != "class" {
			buf.WriteString(` `)
			buf.WriteString(html.EscapeString(key))
			buf.WriteString(`="`)
			buf.WriteString(html.EscapeString(value))
			buf.WriteString(`"`)
		}
	}

	buf.WriteString(`>`)

	// Tab navigation
	if navSlot, ok := slots["nav"]; ok {
		buf.WriteString(`<div class="bk-tabs-nav">`)
		buf.WriteString(navSlot)
		buf.WriteString(`</div>`)
	}

	// Tab content
	buf.WriteString(`<div class="bk-tabs-content">`)
	buf.WriteString(slots["default"])
	buf.WriteString(`</div>`)

	buf.WriteString(`</div>`)

	return []byte(buf.String()), nil
}

// renderCode renders a code block component with syntax highlighting support.
func renderCode(attrs map[string]string, slots map[string]string) ([]byte, error) {
	var buf strings.Builder

	language := attrs["language"]
	if language == "" {
		language = "plaintext"
	}

	buf.WriteString(`<pre class="bk-code"`)

	// Add other attributes
	for key, value := range attrs {
		if key != "language" && key != "class" {
			buf.WriteString(` `)
			buf.WriteString(html.EscapeString(key))
			buf.WriteString(`="`)
			buf.WriteString(html.EscapeString(value))
			buf.WriteString(`"`)
		}
	}

	buf.WriteString(`><code class="language-`)
	buf.WriteString(html.EscapeString(language))
	buf.WriteString(`">`)

	// Preserve whitespace and escape HTML
	content := slots["default"]
	// Don't escape if it looks like it's already escaped or is meant to be raw
	if strings.Contains(content, "&lt;") || strings.Contains(content, "&gt;") {
		buf.WriteString(content)
	} else {
		buf.WriteString(html.EscapeString(content))
	}

	buf.WriteString(`</code></pre>`)

	return []byte(buf.String()), nil
}

// renderIcon renders an icon component.
func renderIcon(attrs map[string]string, slots map[string]string) ([]byte, error) {
	var buf strings.Builder

	name := attrs["name"]
	if name == "" {
		name = "default"
	}

	buf.WriteString(`<span class="bk-icon bk-icon-`)
	buf.WriteString(html.EscapeString(name))
	buf.WriteString(`"`)

	// Add ARIA label for accessibility
	if label, ok := attrs["aria-label"]; ok {
		buf.WriteString(` aria-label="`)
		buf.WriteString(html.EscapeString(label))
		buf.WriteString(`"`)
	} else {
		buf.WriteString(` aria-hidden="true"`)
	}

	// Add other attributes
	for key, value := range attrs {
		if key != "name" && key != "class" && key != "aria-label" {
			buf.WriteString(` `)
			buf.WriteString(html.EscapeString(key))
			buf.WriteString(`="`)
			buf.WriteString(html.EscapeString(value))
			buf.WriteString(`"`)
		}
	}

	buf.WriteString(`>`)

	// Icon content (could be SVG, font icon, etc.)
	if slots["default"] != "" {
		buf.WriteString(slots["default"])
	} else {
		// Default icon representation
		buf.WriteString(`<svg viewBox="0 0 24 24" fill="currentColor">`)
		switch name {
		case "check":
			buf.WriteString(`<path d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z"/>`)
		case "close":
			buf.WriteString(`<path d="M19 6.41L17.59 5 12 10.59 6.41 5 5 6.41 10.59 12 5 17.59 6.41 19 12 13.41 17.59 19 19 17.59 13.41 12z"/>`)
		default:
			buf.WriteString(`<circle cx="12" cy="12" r="10"/>`)
		}
		buf.WriteString(`</svg>`)
	}

	buf.WriteString(`</span>`)

	return []byte(buf.String()), nil
}

// renderAvatar renders a user avatar component.
func renderAvatar(attrs map[string]string, slots map[string]string) ([]byte, error) {
	var buf strings.Builder

	userID := attrs["user-id"]
	size := attrs["size"]
	if size == "" {
		size = "medium"
	}

	// In a real app, this would fetch user data from the database
	// For testing, we'll generate a placeholder
	avatarURL := fmt.Sprintf("/avatars/user-%s.jpg", userID)
	userName := fmt.Sprintf("User %s", userID)

	if url, ok := attrs["src"]; ok {
		avatarURL = url
	}
	if name, ok := attrs["alt"]; ok {
		userName = name
	}

	buf.WriteString(`<div class="bk-avatar bk-avatar-`)
	buf.WriteString(html.EscapeString(size))
	buf.WriteString(`"`)

	// Add data attributes
	if userID != "" {
		buf.WriteString(` data-user-id="`)
		buf.WriteString(html.EscapeString(userID))
		buf.WriteString(`"`)
	}

	// Add other attributes
	for key, value := range attrs {
		if key != "user-id" && key != "size" && key != "src" && key != "alt" && key != "class" {
			buf.WriteString(` `)
			buf.WriteString(html.EscapeString(key))
			buf.WriteString(`="`)
			buf.WriteString(html.EscapeString(value))
			buf.WriteString(`"`)
		}
	}

	buf.WriteString(`>`)
	buf.WriteString(`<img src="`)
	buf.WriteString(html.EscapeString(avatarURL))
	buf.WriteString(`" alt="`)
	buf.WriteString(html.EscapeString(userName))
	buf.WriteString(`" class="bk-avatar-img">`)

	// Optional status indicator
	if status, ok := attrs["status"]; ok {
		buf.WriteString(`<span class="bk-avatar-status bk-avatar-status-`)
		buf.WriteString(html.EscapeString(status))
		buf.WriteString(`"></span>`)
	}

	buf.WriteString(`</div>`)

	return []byte(buf.String()), nil
}

// Global feature flags for testing (in production, this would come from config)
var featureFlags = make(map[string]bool)

// SetFeatureFlag sets a feature flag for testing purposes.
func SetFeatureFlag(flag string, enabled bool) {
	featureFlags[flag] = enabled
}

// renderFeatureFlag renders content conditionally based on feature flags.
func renderFeatureFlag(attrs map[string]string, slots map[string]string) ([]byte, error) {
	flag := attrs["flag"]
	if flag == "" {
		// If no flag specified, don't render anything
		return []byte(""), nil
	}

	// Check if the flag is enabled
	if enabled, ok := featureFlags[flag]; ok && enabled {
		// Flag is enabled, render the content
		return []byte(slots["default"]), nil
	}

	// Flag is disabled or not set, check for fallback slot
	if fallback, ok := slots["fallback"]; ok {
		return []byte(fallback), nil
	}

	// No content to render
	return []byte(""), nil
}

// renderProgressBar renders a progress bar component.
func renderProgressBar(attrs map[string]string, slots map[string]string) ([]byte, error) {
	var buf strings.Builder

	value := attrs["value"]
	if value == "" {
		value = "0"
	}

	max := attrs["max"]
	if max == "" {
		max = "100"
	}

	buf.WriteString(`<div class="bk-progress-bar"`)
	buf.WriteString(` role="progressbar"`)
	buf.WriteString(` aria-valuenow="`)
	buf.WriteString(html.EscapeString(value))
	buf.WriteString(`"`)
	buf.WriteString(` aria-valuemin="0"`)
	buf.WriteString(` aria-valuemax="`)
	buf.WriteString(html.EscapeString(max))
	buf.WriteString(`"`)

	// Add other attributes
	for key, val := range attrs {
		if key != "value" && key != "max" && key != "class" {
			buf.WriteString(` `)
			buf.WriteString(html.EscapeString(key))
			buf.WriteString(`="`)
			buf.WriteString(html.EscapeString(val))
			buf.WriteString(`"`)
		}
	}

	buf.WriteString(`>`)

	// Calculate percentage
	valueInt, _ := strconv.Atoi(value)
	maxInt, _ := strconv.Atoi(max)
	percentage := 0
	if maxInt > 0 {
		percentage = (valueInt * 100) / maxInt
	}

	buf.WriteString(`<div class="bk-progress-bar-fill" style="width: `)
	buf.WriteString(strconv.Itoa(percentage))
	buf.WriteString(`%">`)

	// Optional label
	if label, ok := slots["default"]; ok && label != "" {
		buf.WriteString(`<span class="bk-progress-bar-label">`)
		buf.WriteString(label)
		buf.WriteString(`</span>`)
	}

	buf.WriteString(`</div>`)
	buf.WriteString(`</div>`)

	return []byte(buf.String()), nil
}

// renderFormField renders a form field wrapper with label, input, and error slots.
func renderFormField(attrs map[string]string, slots map[string]string) ([]byte, error) {
	var buf strings.Builder

	label := attrs["label"]
	name := attrs["name"]
	if name == "" {
		name = fmt.Sprintf("field-%d", hashString(label))
	}

	fieldID := fmt.Sprintf("field-%s", name)
	errorID := fmt.Sprintf("error-%s", name)

	buf.WriteString(`<div class="bk-form-field"`)

	// Add other attributes
	for key, value := range attrs {
		if key != "label" && key != "name" && key != "class" {
			buf.WriteString(` `)
			buf.WriteString(html.EscapeString(key))
			buf.WriteString(`="`)
			buf.WriteString(html.EscapeString(value))
			buf.WriteString(`"`)
		}
	}

	buf.WriteString(`>`)

	// Label
	if label != "" {
		buf.WriteString(`<label for="`)
		buf.WriteString(fieldID)
		buf.WriteString(`" class="bk-form-field-label">`)
		buf.WriteString(html.EscapeString(label))

		// Add required indicator if present
		if required, ok := attrs["required"]; ok && required == "true" {
			buf.WriteString(`<span class="bk-form-field-required" aria-label="required">*</span>`)
		}

		buf.WriteString(`</label>`)
	}

	// Input area (from default slot or input slot)
	buf.WriteString(`<div class="bk-form-field-input" id="`)
	buf.WriteString(fieldID)
	buf.WriteString(`"`)

	// Add ARIA describedby if there's an error slot
	if _, hasError := slots["error"]; hasError {
		buf.WriteString(` aria-describedby="`)
		buf.WriteString(errorID)
		buf.WriteString(`"`)
	}

	buf.WriteString(`>`)

	if inputSlot, ok := slots["input"]; ok {
		buf.WriteString(inputSlot)
	} else {
		buf.WriteString(slots["default"])
	}

	buf.WriteString(`</div>`)

	// Help text (if provided)
	if helpSlot, ok := slots["help"]; ok {
		buf.WriteString(`<div class="bk-form-field-help">`)
		buf.WriteString(helpSlot)
		buf.WriteString(`</div>`)
	}

	// Error message (if provided)
	if errorSlot, ok := slots["error"]; ok {
		buf.WriteString(`<div class="bk-form-field-error" id="`)
		buf.WriteString(errorID)
		buf.WriteString(`" role="alert">`)
		buf.WriteString(errorSlot)
		buf.WriteString(`</div>`)
	}

	buf.WriteString(`</div>`)

	return []byte(buf.String()), nil
}

// hashString generates a simple hash for creating unique IDs.
// Used for generating unique IDs for dropdowns and other components
// that need to reference elements.
//
// TODO: Use a proper hash function or UUID generator in production
func hashString(s string) int {
	h := 0
	for _, c := range s {
		h = h*31 + int(c)
	}
	if h < 0 {
		h = -h
	}
	return h
}

// ComponentsCSS returns default CSS for components.
// This provides basic styling for all built-in components.
// Apps should include this CSS in their layout or override with custom styles.
//
// Usage in layout:
//
//	<style><%= componentsCSS() %></style>
//
// Or save to a file and include as a stylesheet.
// All classes are prefixed with "bk-" to avoid conflicts.
func ComponentsCSS() string {
	return `
/* Buffkit Components Default Styles */
.bk-button {
	display: inline-block;
	padding: 0.5rem 1rem;
	border: 1px solid #ddd;
	border-radius: 0.25rem;
	background: white;
	cursor: pointer;
	text-decoration: none;
	color: inherit;
}

.bk-button-primary {
	background: #007bff;
	color: white;
	border-color: #007bff;
}

.bk-button-danger {
	background: #dc3545;
	color: white;
	border-color: #dc3545;
}

.bk-card {
	border: 1px solid #ddd;
	border-radius: 0.25rem;
	margin-bottom: 1rem;
}

.bk-card-header {
	padding: 0.75rem 1rem;
	background: #f7f7f7;
	border-bottom: 1px solid #ddd;
}

.bk-card-body {
	padding: 1rem;
}

.bk-card-footer {
	padding: 0.75rem 1rem;
	background: #f7f7f7;
	border-top: 1px solid #ddd;
}

.bk-dropdown {
	position: relative;
	display: inline-block;
}

.bk-dropdown-menu {
	position: absolute;
	top: 100%;
	left: 0;
	z-index: 1000;
	min-width: 10rem;
	background: white;
	border: 1px solid #ddd;
	border-radius: 0.25rem;
	box-shadow: 0 2px 4px rgba(0,0,0,0.1);
}

.bk-alert {
	padding: 0.75rem 1rem;
	margin-bottom: 1rem;
	border: 1px solid transparent;
	border-radius: 0.25rem;
}

.bk-alert-info {
	background: #d1ecf1;
	border-color: #bee5eb;
	color: #0c5460;
}

.bk-alert-success {
	background: #d4edda;
	border-color: #c3e6cb;
	color: #155724;
}

.bk-alert-warning {
	background: #fff3cd;
	border-color: #ffeeba;
	color: #856404;
}

.bk-alert-danger {
	background: #f8d7da;
	border-color: #f5c6cb;
	color: #721c24;
}

.bk-form {
	margin-bottom: 1rem;
}

.bk-input-group {
	margin-bottom: 1rem;
}

.bk-label {
	display: block;
	margin-bottom: 0.25rem;
	font-weight: 500;
}

.bk-input {
	display: block;
	width: 100%;
	padding: 0.375rem 0.75rem;
	border: 1px solid #ced4da;
	border-radius: 0.25rem;
}

.bk-required {
	color: #dc3545;
}
`
}

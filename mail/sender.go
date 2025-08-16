package mail

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/smtp"
	"strings"
	"time"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/buffalo/render"
)

// Message represents an email message
type Message struct {
	From    string   // Optional, uses default if empty
	To      string   // Required recipient email
	Cc      []string // Optional CC recipients
	Bcc     []string // Optional BCC recipients
	Subject string   // Email subject
	Text    string   // Plain text body
	HTML    string   // HTML body (optional)
}

// Sender is the interface for sending emails
type Sender interface {
	Send(ctx context.Context, msg Message) error
}

// SMTPConfig holds SMTP server configuration
type SMTPConfig struct {
	Addr     string // Host:port (e.g., "smtp.gmail.com:587")
	User     string // SMTP username
	Password string // SMTP password
	From     string // Default sender email
}

// SMTPSender sends emails via SMTP
type SMTPSender struct {
	config SMTPConfig
}

// NewSMTPSender creates a new SMTP sender
func NewSMTPSender(config SMTPConfig) *SMTPSender {
	return &SMTPSender{
		config: config,
	}
}

// Send sends an email via SMTP
func (s *SMTPSender) Send(ctx context.Context, msg Message) error {
	// Use default from if not specified
	from := msg.From
	if from == "" {
		from = s.config.From
	}
	if from == "" {
		from = s.config.User // Fallback to username
	}

	// Build recipient list
	recipients := []string{msg.To}
	recipients = append(recipients, msg.Cc...)
	recipients = append(recipients, msg.Bcc...)

	// Build email headers and body
	var headers strings.Builder
	headers.WriteString(fmt.Sprintf("From: %s\r\n", from))
	headers.WriteString(fmt.Sprintf("To: %s\r\n", msg.To))

	if len(msg.Cc) > 0 {
		headers.WriteString(fmt.Sprintf("Cc: %s\r\n", strings.Join(msg.Cc, ", ")))
	}

	headers.WriteString(fmt.Sprintf("Subject: %s\r\n", msg.Subject))
	headers.WriteString(fmt.Sprintf("Date: %s\r\n", time.Now().Format(time.RFC1123Z)))
	headers.WriteString("MIME-Version: 1.0\r\n")

	// Determine content type and body
	var body string
	if msg.HTML != "" {
		headers.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n")
		body = msg.HTML
	} else {
		headers.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n")
		body = msg.Text
	}

	headers.WriteString("\r\n")
	fullMessage := headers.String() + body

	// Setup authentication
	var auth smtp.Auth
	if s.config.User != "" && s.config.Password != "" {
		host := strings.Split(s.config.Addr, ":")[0]
		auth = smtp.PlainAuth("", s.config.User, s.config.Password, host)
	}

	// Send the email
	err := smtp.SendMail(
		s.config.Addr,
		auth,
		from,
		recipients,
		[]byte(fullMessage),
	)

	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	log.Printf("Mail: Sent email to %s: %s", msg.To, msg.Subject)
	return nil
}

// DevSender logs emails instead of sending them (for development)
type DevSender struct {
	messages []Message // Store messages for preview
}

// NewDevSender creates a new development sender
func NewDevSender() *DevSender {
	return &DevSender{
		messages: make([]Message, 0),
	}
}

// Send logs the email instead of sending it
func (d *DevSender) Send(ctx context.Context, msg Message) error {
	log.Printf("Mail (Dev): Would send email to %s", msg.To)
	log.Printf("  Subject: %s", msg.Subject)
	if msg.Text != "" {
		log.Printf("  Text: %s", truncate(msg.Text, 100))
	}
	if msg.HTML != "" {
		log.Printf("  HTML: %s", truncate(msg.HTML, 100))
	}

	// Store for preview
	d.messages = append(d.messages, msg)

	return nil
}

// GetMessages returns stored messages (for preview)
func (d *DevSender) GetMessages() []Message {
	return d.messages
}

// NoOpSender does nothing (for testing)
type NoOpSender struct{}

// Send does nothing
func (n *NoOpSender) Send(ctx context.Context, msg Message) error {
	return nil
}

// Global sender instance
var globalSender Sender

// UseSender sets the global mail sender
func UseSender(s Sender) {
	globalSender = s
}

// GetSender returns the current mail sender
func GetSender() Sender {
	if globalSender == nil {
		return NewDevSender()
	}
	return globalSender
}

// Send sends an email using the global sender
func Send(ctx context.Context, msg Message) error {
	return GetSender().Send(ctx, msg)
}

// PreviewHandler shows sent emails in development mode
func PreviewHandler(c buffalo.Context) error {
	// Get dev sender
	sender := GetSender()
	devSender, ok := sender.(*DevSender)
	if !ok {
		html := `
<!DOCTYPE html>
<html>
<head>
    <title>Mail Preview</title>
    <style>
        body { font-family: system-ui, sans-serif; padding: 20px; }
        .error { color: red; }
    </style>
</head>
<body>
    <h1>Mail Preview</h1>
    <p class="error">Mail preview is only available with DevSender</p>
</body>
</html>
		`
		return c.Render(http.StatusOK, mailRenderer{html: html})
	}

	// Build preview HTML
	messages := devSender.GetMessages()
	var preview strings.Builder
	preview.WriteString(`
<!DOCTYPE html>
<html>
<head>
    <title>Mail Preview</title>
    <style>
        body { font-family: system-ui, sans-serif; padding: 20px; }
        .message { border: 1px solid #ddd; margin: 20px 0; padding: 15px; }
        .header { background: #f5f5f5; padding: 10px; margin: -15px -15px 15px; }
        .subject { font-weight: bold; font-size: 1.2em; }
        .meta { color: #666; font-size: 0.9em; margin: 5px 0; }
        .body { margin-top: 15px; padding: 10px; background: #fafafa; }
        pre { white-space: pre-wrap; word-wrap: break-word; }
    </style>
</head>
<body>
    <h1>Mail Preview (Development)</h1>
    <p>Showing ` + fmt.Sprintf("%d", len(messages)) + ` message(s)</p>
`)

	if len(messages) == 0 {
		preview.WriteString(`<p><em>No messages sent yet</em></p>`)
	}

	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		preview.WriteString(`
    <div class="message">
        <div class="header">
            <div class="subject">` + msg.Subject + `</div>
            <div class="meta">To: ` + msg.To + `</div>
        </div>
`)
		if msg.HTML != "" {
			preview.WriteString(`
        <div class="body">
            <strong>HTML Body:</strong>
            <div style="border: 1px solid #ccc; padding: 10px; margin-top: 5px;">
                ` + msg.HTML + `
            </div>
        </div>
`)
		}
		if msg.Text != "" {
			preview.WriteString(`
        <div class="body">
            <strong>Text Body:</strong>
            <pre>` + msg.Text + `</pre>
        </div>
`)
		}
		preview.WriteString(`</div>`)
	}

	preview.WriteString(`
</body>
</html>
`)

	return c.Render(http.StatusOK, mailRenderer{html: preview.String()})
}

// Helper functions

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

// Simple HTML renderer for stub
type mailRenderer struct {
	html string
}

func (mailRenderer) HTML(s string) mailRenderer {
	return mailRenderer{html: s}
}

func (r mailRenderer) ContentType() string {
	return "text/html; charset=utf-8"
}

func (r mailRenderer) Render(w io.Writer, data render.Data) error {
	if hw, ok := w.(http.ResponseWriter); ok {
		hw.Header().Set("Content-Type", r.ContentType())
	}
	_, err := w.Write([]byte(r.html))
	return err
}

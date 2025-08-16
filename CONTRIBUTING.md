# Contributing to Buffkit

Thank you for your interest in contributing to Buffkit! We're building an opinionated SSR-first stack for Buffalo (Go) applications, and we welcome contributions that align with our vision of simplicity, clarity, and practical minimalism.

## 🚧 Project Status

Buffkit is in early development. The API is still evolving, and we're establishing core patterns. Your patience and understanding during this phase is appreciated.

## 📋 Before You Contribute

1. **Check existing issues** - Someone might already be working on it
2. **Read the documentation** - Familiarize yourself with the project structure
3. **Understand our philosophy** - We favor clarity over cleverness, simplicity over complexity

## 🛠️ Development Setup

### Prerequisites

- Go 1.21 or higher
- Redis (for testing)
- Make
- Git

### Getting Started

```bash
# Clone the repository
git clone https://github.com/johnjansen/buffkit.git
cd buffkit

# Install dependencies
go mod download

# Run tests
make test

# Run specific feature tests
make test-focus FOCUS="@your-tag"

# Run linting
make lint
```

## 🧪 Testing Philosophy

We use Behavior-Driven Development (BDD) with Gherkin feature files:

1. **Write feature files first** - Document the intended behavior
2. **Implement step definitions** - Make the tests pass
3. **Keep tests fast** - Our entire suite runs in under a second
4. **Test real behavior** - Not implementation details

### Running Tests

```bash
# Run all tests
make test

# Run with coverage
go test -v ./... -coverprofile=coverage.txt -covermode=atomic

# Run specific scenarios
make test-focus FOCUS="@authentication"
```

## 📝 Code Style

### General Principles

- **Clean and idiomatic** - Follow language best practices
- **Comment aggressively** - Explain the why, not just the what
- **Small functions** - Aim for <20 lines, low cyclomatic complexity
- **One file, one concern** - Each major component gets its own file
- **Consistent returns** - No exceptions for control flow

### Go Specific

- Follow standard Go formatting (`gofmt`)
- Use meaningful variable names
- Handle errors explicitly
- Prefer concrete types over interfaces until abstraction is needed

### Example

```go
// BadgeRenderer generates HTML for status badges.
// It follows the shields.io standard for maximum compatibility.
type BadgeRenderer struct {
    // Style determines the visual appearance (flat, flat-square, etc)
    Style string
    
    // BaseURL for badge generation service
    BaseURL string
}

// Render creates an HTML img tag for the given badge parameters.
// Returns an error if the parameters are invalid.
func (br *BadgeRenderer) Render(label, message, color string) (string, error) {
    // Validate inputs first
    if label == "" || message == "" {
        return "", fmt.Errorf("badge requires both label and message")
    }
    
    // Build the badge URL
    url := br.buildURL(label, message, color)
    
    // Generate clean HTML
    return fmt.Sprintf(`<img src="%s" alt="%s: %s">`, url, label, message), nil
}
```

## 🎯 What We're Looking For

### High Priority

- SSE (Server-Sent Events) improvements
- Authentication system enhancements
- Component registry features
- Documentation improvements
- Test coverage expansion

### Good First Issues

Look for issues labeled `good first issue` - these are well-defined tasks suitable for newcomers.

## 📤 Submitting Changes

### 1. Fork and Branch

```bash
git checkout -b feature/your-feature-name
```

### 2. Make Your Changes

- Write tests first (TDD/BDD approach)
- Keep commits focused and atomic
- Update documentation as needed

### 3. Commit Messages

Follow conventional commits:

```
feat: add SSE reconnection logic
fix: prevent memory leak in broker
docs: update component examples
test: add coverage for auth middleware
```

### 4. Push and PR

```bash
git push origin feature/your-feature-name
```

Then open a Pull Request with:
- Clear description of the change
- Link to related issues
- Screenshots/examples if applicable

## 🚀 Pull Request Process

1. **Tests must pass** - All CI checks green
2. **Coverage maintained** - Don't decrease test coverage
3. **Documentation updated** - If you changed behavior
4. **Review addressed** - Respond to feedback constructively

## 🐛 Reporting Issues

### Bug Reports Should Include

- Go version
- Buffalo version  
- Minimal reproduction steps
- Expected vs actual behavior
- Error messages/stack traces

### Feature Requests Should Include

- Use case description
- Proposed API/interface
- Alternative solutions considered
- Impact on existing functionality

## 💡 Design Decisions

### Our Approach

- **SSR-first** - Server-side rendering is the default
- **Progressive enhancement** - JavaScript enhances, doesn't replace
- **Batteries included** - Common needs solved out of the box
- **Zero config start** - Sensible defaults that just work
- **Escape hatches** - Override when needed

### What We Avoid

- Premature abstraction
- Overengineering
- Windows-specific accommodations (we target macOS/Linux)
- Feature creep beyond core mission

## 📚 Documentation

Documentation lives in several places:

- **README.md** - Project overview and quick start
- **Feature files** - Behavior documentation
- **Code comments** - Implementation details
- **Examples directory** - Working examples

When contributing, update relevant documentation.

## 🤝 Code of Conduct

### Be Respectful

- Welcome newcomers
- Be patient with questions
- Provide constructive feedback
- Respect different perspectives

### Be Professional

- Stay on topic
- No harassment or discrimination
- Keep discussions technical
- Assume good intentions

## 📮 Getting Help

- **GitHub Issues** - For bugs and features
- **Discussions** - For questions and ideas
- **Code Review** - Request feedback on drafts

## 🙏 Recognition

Contributors are recognized in:
- Release notes
- Contributors file
- Project documentation

## 📄 License

By contributing, you agree that your contributions will be licensed under the MIT License.

---

Thank you for helping make Buffkit better! 🎉
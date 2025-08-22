# Buffkit Generators

Buffkit provides Rails-like code generators to quickly scaffold models, controllers, migrations, and buffkit-specific components. All generators are accessible via `buffalo task` commands.

## Quick Reference

```bash
# Standard MVC generators
buffalo task buffkit:generate:model user name:string email:string age:int
buffalo task buffkit:generate:action users index show create update destroy
buffalo task buffkit:generate:resource post title:string content:text published:bool
buffalo task buffkit:generate:migration create_users name:string email:string

# Buffkit-specific generators
buffalo task buffkit:generate:component card
buffalo task buffkit:generate:job email_processor
buffalo task buffkit:generate:mailer user welcome password_reset
buffalo task buffkit:generate:sse notification

# Shorthand aliases (g instead of buffkit:generate)
buffalo task g:model user name:string
buffalo task g:action users
buffalo task g:component button
```

## Model Generator

Generates a model struct with CRUD methods and optionally creates a migration.

### Usage
```bash
buffalo task buffkit:generate:model <name> [field:type ...]
```

### Field Types
- `string` / `text` → `string`
- `int` / `integer` → `int`
- `bigint` → `int64`
- `float` / `decimal` → `float64`
- `bool` / `boolean` → `bool`
- `date` / `datetime` / `time` → `time.Time`
- `uuid` → `uuid.UUID`
- `json` / `jsonb` → `json.RawMessage`

### Nullable Fields
Add `:nullable` to make a field nullable:
```bash
buffalo task g:model user name:string bio:text:nullable
```

### Example
```bash
buffalo task g:model user name:string email:string age:int active:bool
```

This generates:
- `models/user.go` - Model struct with CRUD methods
- `db/migrations/core/[timestamp]_create_users.up.sql` - CREATE TABLE migration
- `db/migrations/core/[timestamp]_create_users.down.sql` - DROP TABLE migration

Generated model includes:
- `Create()` - Insert record
- `Update()` - Update record
- `Delete()` - Delete record
- `FindUser()` - Find by ID
- `AllUsers()` - List all records

## Action Generator

Generates Buffalo action handlers (controllers in Rails terminology).

### Usage
```bash
buffalo task buffkit:generate:action <resource> [actions...]
```

### Default Actions
If no actions specified, generates all RESTful actions:
- `index` - List all resources
- `show` - Show single resource
- `new` - New resource form
- `create` - Create resource
- `edit` - Edit form
- `update` - Update resource
- `destroy` - Delete resource

### Example
```bash
# Generate all RESTful actions
buffalo task g:action users

# Generate specific actions only
buffalo task g:action posts index show create

# Custom actions
buffalo task g:action users profile settings update_password
```

Generates `actions/users.go` with all specified handlers.

## Resource Generator

Generates a complete resource: model + actions + views.

### Usage
```bash
buffalo task buffkit:generate:resource <name> [field:type ...]
```

### Example
```bash
buffalo task g:resource article title:string content:text published:bool author_id:int
```

This generates:
- Model with migration
- All RESTful actions
- View templates:
  - `templates/articles/index.plush.html`
  - `templates/articles/show.plush.html`
  - `templates/articles/new.plush.html`
  - `templates/articles/edit.plush.html`
  - `templates/articles/_form.plush.html`

## Migration Generator

Enhanced migration generator with field support.

### Usage
```bash
buffalo task buffkit:generate:migration <name> [field:type ...]
```

### Migration Types
The generator detects migration type from the name:
- `create_*` - Generates CREATE TABLE
- `add_*_to_*` - Generates ALTER TABLE ADD COLUMN
- `remove_*_from_*` - Generates ALTER TABLE DROP COLUMN
- Others - Generates empty migration template

### Examples
```bash
# Create table migration
buffalo task g:migration create_products name:string price:decimal stock:int

# Add columns migration
buffalo task g:migration add_description_to_products description:text

# Remove columns migration  
buffalo task g:migration remove_stock_from_products stock:int

# Custom migration
buffalo task g:migration update_user_indexes
```

## Component Generator

Generates server-side components for buffkit's component system.

### Usage
```bash
buffalo task buffkit:generate:component <name>
```

### Example
```bash
buffalo task g:component modal
```

Generates:
- `components/modal.go` - Component implementation
- `assets/css/components/modal.css` - Component styles

The component can be used in templates:
```html
<bk-modal variant="primary" id="my-modal">
  <bk-slot name="header">
    <h2>Modal Title</h2>
  </bk-slot>
  
  Modal content goes here
  
  <bk-slot name="footer">
    <button>Close</button>
  </bk-slot>
</bk-modal>
```

Register in your app:
```go
kit.Components.Register("modal", components.ModalComponent)
```

## Job Generator

Generates background job handlers for async processing.

### Usage
```bash
buffalo task buffkit:generate:job <name>
```

### Example
```bash
buffalo task g:job email_sender
```

Generates `jobs/email_sender.go` with:
- Job payload struct
- Handler function
- Enqueue helper
- Registration function

Register in your app:
```go
jobs.RegisterEmailSenderHandler(kit.Jobs.Mux)
```

Enqueue jobs:
```go
jobs.EnqueueEmailSender(kit.Jobs.Client, "user@example.com")
```

## Mailer Generator

Generates email handlers and templates.

### Usage
```bash
buffalo task buffkit:generate:mailer <name> [actions...]
```

### Example
```bash
buffalo task g:mailer user welcome password_reset confirmation
```

Generates:
- `mailers/user.go` - Mailer handler with methods for each action
- `templates/mail/user/welcome.html` - Email template
- `templates/mail/user/password_reset.html` - Email template
- `templates/mail/user/confirmation.html` - Email template

Use in your app:
```go
mailer := mailers.NewUserMailer(kit.Mail)
err := mailer.SendWelcome(ctx, "user@example.com", map[string]interface{}{
    "Name": "John",
    "ActivationLink": "https://...",
})
```

## SSE Generator

Generates Server-Sent Events handlers for real-time updates.

### Usage
```bash
buffalo task buffkit:generate:sse <name>
```

### Example
```bash
buffalo task g:sse notification
```

Generates `sse/notification.go` with:
- Event struct
- Handler function
- Broadcast helper
- Route setup function

Set up in your app:
```go
sse.SetupNotificationRoutes(app, kit.Broker)
```

Broadcast events:
```go
sse.BroadcastNotification(kit.Broker, "New message!")
```

Client-side:
```html
<div hx-sse="connect:/events/notification" hx-sse-swap="message">
  <!-- Updates appear here -->
</div>
```

## Name Transformations

Generators automatically handle name transformations:

| Input | snake_case | CamelCase | plural | kebab-case |
|-------|------------|-----------|--------|------------|
| user | user | User | users | user |
| UserProfile | user_profile | UserProfile | user_profiles | user-profile |
| blog-post | blog_post | BlogPost | blog_posts | blog-post |
| person | person | Person | people | person |

## Best Practices

### 1. Start with Resources
For full CRUD operations, use the resource generator:
```bash
buffalo task g:resource product name:string price:decimal inventory:int
```

### 2. Use Consistent Naming
- Models: singular (user, product, article)
- Actions/Controllers: plural (users, products, articles)
- Tables: plural (automatically handled)

### 3. Add Fields Later
You can always add fields with migrations:
```bash
buffalo task g:migration add_description_to_products description:text
```

### 4. Leverage Components
Create reusable UI components:
```bash
buffalo task g:component alert
buffalo task g:component data-table
buffalo task g:component form-field
```

### 5. Background Jobs for Heavy Work
Move expensive operations to background jobs:
```bash
buffalo task g:job image_processor
buffalo task g:job report_generator
buffalo task g:job data_importer
```

## Customization

### Templates Location
Generator templates are embedded in the generator code but can be overridden by placing custom templates in:
```
generators/templates/
├── model.go.tmpl
├── action.go.tmpl
├── component.go.tmpl
└── ...
```

### Adding Custom Generators

Create a new generator by adding to `generators/grifts.go`:

```go
_ = grift.Add("my_generator", func(c *grift.Context) error {
    // Your generator logic
    return nil
})
```

## Troubleshooting

### Generator Not Found
Ensure buffkit is imported in your app:
```go
import _ "github.com/johnjansen/buffkit"
```

### Files Already Exist
Generators won't overwrite existing files. Remove or rename existing files first.

### Invalid Field Types
Check the field type mapping in the Field Types section above.

### Migration Fails
Ensure your database is configured and accessible:
```bash
buffalo task buffkit:migrate:status
```

## Examples

### Blog Application
```bash
# Generate blog post resource
buffalo task g:resource post title:string slug:string content:text published:bool author_id:int

# Generate comment model
buffalo task g:model comment post_id:int author:string content:text approved:bool

# Generate admin actions
buffalo task g:action admin/posts index approve reject

# Generate email notifications
buffalo task g:mailer comment_notification new_comment comment_approved

# Generate real-time updates
buffalo task g:sse comment_stream
```

### E-commerce Application
```bash
# Products
buffalo task g:resource product name:string description:text price:decimal stock:int

# Orders
buffalo task g:resource order user_id:int total:decimal status:string

# Order items
buffalo task g:model order_item order_id:int product_id:int quantity:int price:decimal

# Background jobs
buffalo task g:job order_processor
buffalo task g:job inventory_updater
buffalo task g:job email_invoice

# Components
buffalo task g:component product-card
buffalo task g:component shopping-cart
buffalo task g:component checkout-form
```

## Next Steps

After generating code:

1. **Run migrations**: `buffalo task buffkit:migrate`
2. **Register routes**: Add to your `app.go`
3. **Customize templates**: Edit generated views
4. **Add validation**: Enhance model validation
5. **Write tests**: Test your generated code

## Related Documentation

- [Buffalo Documentation](https://gobuffalo.io)
- [Buffkit README](README.md)
- [WARP Guide](WARP.md)
- [Architecture](HOW_IT_WORKS.md)

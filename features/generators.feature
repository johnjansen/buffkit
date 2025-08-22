Feature: Buffkit Generators
  As a developer
  I want to use Rails-like generators
  So that I can quickly scaffold code

  Background:
    Given I have a test project directory
    And buffkit is available

  @generators @model
  Scenario: Generate a model with fields
    When I run "buffalo task buffkit:generate:model user name:string email:string age:int"
    Then the file "models/user.go" should exist
    And the file "models/user.go" should contain "type User struct"
    And the file "models/user.go" should contain "Name string"
    And the file "models/user.go" should contain "Email string"
    And the file "models/user.go" should contain "Age int"
    And the file "models/user.go" should contain "func (user *User) Create"
    And the file "models/user.go" should contain "func FindUser"
    And the file "models/user.go" should contain "func AllUsers"
    And a migration file matching "db/migrations/core/*_create_users.up.sql" should exist
    And a migration file matching "db/migrations/core/*_create_users.down.sql" should exist

  @generators @action
  Scenario: Generate actions for a resource
    When I run "buffalo task buffkit:generate:action posts index show create"
    Then the file "actions/posts.go" should exist
    And the file "actions/posts.go" should contain "func PostsIndex"
    And the file "actions/posts.go" should contain "func PostsShow"
    And the file "actions/posts.go" should contain "func PostsCreate"
    And the file "actions/posts.go" should not contain "func PostsEdit"

  @generators @resource
  Scenario: Generate a complete resource
    When I run "buffalo task buffkit:generate:resource article title:string content:text"
    Then the file "models/article.go" should exist
    And the file "actions/articles.go" should exist
    And the file "templates/articles/index.plush.html" should exist
    And the file "templates/articles/show.plush.html" should exist
    And the file "templates/articles/new.plush.html" should exist
    And the file "templates/articles/edit.plush.html" should exist
    And the file "templates/articles/_form.plush.html" should exist

  @generators @migration
  Scenario: Generate a create table migration
    When I run "buffalo task buffkit:generate:migration create_products name:string price:decimal"
    Then a migration file matching "db/migrations/core/*_create_products.up.sql" should exist
    And the migration up file should contain "CREATE TABLE products"
    And the migration up file should contain "name VARCHAR(255)"
    And the migration up file should contain "price DECIMAL(10,2)"
    And a migration file matching "db/migrations/core/*_create_products.down.sql" should exist
    And the migration down file should contain "DROP TABLE IF EXISTS products"

  @generators @migration
  Scenario: Generate an add columns migration
    When I run "buffalo task buffkit:generate:migration add_description_to_products description:text"
    Then a migration file matching "db/migrations/core/*_add_description_to_products.up.sql" should exist
    And the migration up file should contain "ALTER TABLE products"
    And the migration up file should contain "ADD COLUMN description VARCHAR(255)"

  @generators @component
  Scenario: Generate a server-side component
    When I run "buffalo task buffkit:generate:component card"
    Then the file "components/card.go" should exist
    And the file "components/card.go" should contain "func CardComponent"
    And the file "components/card.go" should contain "bk-card"
    And the file "assets/css/components/card.css" should exist
    And the file "assets/css/components/card.css" should contain ".bk-card"

  @generators @job
  Scenario: Generate a background job
    When I run "buffalo task buffkit:generate:job email_processor"
    Then the file "jobs/email_processor.go" should exist
    And the file "jobs/email_processor.go" should contain "type EmailProcessorJob struct"
    And the file "jobs/email_processor.go" should contain "func EmailProcessorHandler"
    And the file "jobs/email_processor.go" should contain "func EnqueueEmailProcessor"
    And the file "jobs/email_processor.go" should contain "func RegisterEmailProcessorHandler"

  @generators @mailer
  Scenario: Generate a mailer with actions
    When I run "buffalo task buffkit:generate:mailer user welcome reset"
    Then the file "mailers/user.go" should exist
    And the file "mailers/user.go" should contain "type UserMailer struct"
    And the file "mailers/user.go" should contain "func (m *UserMailer) SendWelcome"
    And the file "mailers/user.go" should contain "func (m *UserMailer) SendReset"
    And the file "templates/mail/user/welcome.html" should exist
    And the file "templates/mail/user/reset.html" should exist

  @generators @sse
  Scenario: Generate an SSE handler
    When I run "buffalo task buffkit:generate:sse notification"
    Then the file "sse/notification.go" should exist
    And the file "sse/notification.go" should contain "type NotificationEvent struct"
    And the file "sse/notification.go" should contain "func NotificationHandler"
    And the file "sse/notification.go" should contain "func BroadcastNotification"

  @generators @shorthand
  Scenario: Use shorthand generator aliases
    When I run "buffalo task g:model product name:string"
    Then the file "models/product.go" should exist
    And the file "models/product.go" should contain "type Product struct"

  @generators @nullable
  Scenario: Generate model with nullable fields
    When I run "buffalo task g:model profile bio:text:nullable website:string:nullable"
    Then the file "models/profile.go" should exist
    And the file "models/profile.go" should contain "*string"

  @generators @pluralization
  Scenario: Handle irregular pluralization
    When I run "buffalo task g:model person name:string"
    Then the file "models/person.go" should exist
    And the file "models/person.go" should contain "func AllPeople"
    And a migration file matching "db/migrations/core/*_create_people.up.sql" should exist

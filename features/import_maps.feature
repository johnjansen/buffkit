Feature: Import Maps System
  As a developer using Buffkit
  I want to manage JavaScript dependencies without a bundler
  So that I can use modern ES modules with simple tooling

  Background:
    Given I have a Buffalo application with Buffkit wired

  Scenario: Default import map includes essential libraries
    When I create a new import map manager
    Then the import map should include "htmx.org"
    And the import map should include "alpinejs"
    And the import map should include "app" pointing to "/assets/js/index.js"

  Scenario: Pin a new JavaScript dependency
    Given I have an import map manager
    When I pin "lodash" to "https://esm.sh/lodash@4.17.21"
    Then the import map should include "lodash"
    And "lodash" should map to "https://esm.sh/lodash@4.17.21"

  Scenario: Pin with vendor download option
    Given I have an import map manager
    When I pin "lodash" to "https://esm.sh/lodash@4.17.21" with download flag
    Then the file should be downloaded to "/assets/vendor/"
    And the import map should point to the local vendored file
    And the vendored file should have a content hash in its name

  Scenario: Unpin a dependency
    Given I have an import map manager
    And I have pinned "lodash"
    When I unpin "lodash"
    Then "lodash" should be removed from the import map
    And the import map should not contain "lodash"

  Scenario: List all pinned dependencies
    Given I have an import map manager
    And I have pinned:
      | name   | url                   |
      | lodash | https://esm.sh/lodash |
      | dayjs  | https://esm.sh/dayjs  |
      | axios  | https://esm.sh/axios  |
    When I list all pins
    Then I should see "lodash -> https://esm.sh/lodash"
    And I should see "dayjs -> https://esm.sh/dayjs"
    And I should see "axios -> https://esm.sh/axios"

  Scenario: Generate import map HTML tag
    Given I have an import map manager with pins
    When I call ToHTML
    Then I should get a valid <script type="importmap"> tag
    And the content should be valid JSON
    And it should include all pinned dependencies

  Scenario: Import map middleware adds to context
    Given import map middleware is configured
    When I make a request to any page
    Then the import map should be available in the context
    And templates should be able to access the import map

  Scenario: Import map in base layout
    Given I have a base layout template
    When the page is rendered
    Then the HTML should include <script type="importmap">
    And it should appear before any module scripts
    And the import map should be properly formatted

  Scenario: Pin with integrity hash for security
    Given I have an import map manager
    When I pin a dependency with an integrity hash
    Then the import map should include the integrity attribute
    And the hash should be verified when loading

  Scenario: Scoped imports for organization
    Given I have an import map manager
    When I pin "@myapp/utils" to "/assets/js/utils/"
    Then imports starting with "@myapp/utils" should resolve correctly
    And I should be able to import "@myapp/utils/helpers.js"

  Scenario: Override existing pin
    Given I have an import map manager
    And "lodash" is pinned to "https://esm.sh/lodash@4.17.20"
    When I pin "lodash" to "https://esm.sh/lodash@4.17.21"
    Then "lodash" should map to "https://esm.sh/lodash@4.17.21"
    And the old mapping should be replaced

  Scenario: Save import map to file
    Given I have an import map manager with pins
    When I save the import map
    Then it should be written to "importmap.json"
    And the file should be valid JSON
    And it should preserve all pins

  Scenario: Load import map from file
    Given I have an "importmap.json" file with pins
    When I create an import map manager
    Then it should load the pins from the file
    And all pins should be available

  Scenario: Import map with development and production URLs
    Given DevMode is true
    When I pin "react" with dev and prod URLs
    Then the development URL should be used
    Given DevMode is false
    Then the production URL should be used

  Scenario: Vendor directory cleanup
    Given I have vendored files from old pins
    When I run vendor cleanup
    Then unused vendored files should be removed
    And currently pinned vendor files should remain

  Scenario: Pin from package.json
    Given I have a package.json with dependencies
    When I run import map sync
    Then dependencies should be pinned automatically
    And versions should match package.json

  Scenario: CDN fallback for vendored files
    Given I have vendored a file locally
    When the local file fails to load
    Then the browser should fall back to the CDN URL
    And a warning should be logged

  Scenario: Import map caching headers
    When import map assets are served
    Then appropriate cache headers should be set
    And vendored files should have long cache times
    And the import map itself should have shorter cache

  Scenario: Preload hints for critical dependencies
    Given I have critical dependencies marked
    When the import map is rendered
    Then <link rel="modulepreload"> tags should be added
    And critical modules should be preloaded

  Scenario: Import maps in development mode
    Given DevMode is true
    When I modify the import map
    Then changes should be reflected immediately
    And no caching should interfere

  Scenario: Import maps in production mode
    Given DevMode is false
    When the import map is served
    Then it should be minified
    And appropriate caching headers should be set

  Scenario: Handle missing vendored files
    Given a vendored file is referenced but missing
    When the import map is loaded
    Then a warning should be logged
    And the CDN URL should be used as fallback

  Scenario: Validate import map syntax
    When I create an invalid import map
    Then validation should fail
    And helpful error messages should be shown

  Scenario: Import map with trailing slashes for directories
    When I pin "utils/" to "/assets/js/utils/"
    Then directory imports should work correctly
    And "utils/helper.js" should resolve to "/assets/js/utils/helper.js"

package importmap

import (
	"fmt"
	"os"
	"strings"

	"github.com/markbates/grift/grift"
)

// RegisterTasks registers import map management tasks with Grift
func RegisterTasks(manager *Manager) {
	_ = grift.Namespace("importmap", func() {
		_ = grift.Desc("pin", "Pin a JavaScript package to the import map")
		_ = grift.Add("pin", func(c *grift.Context) error {
			if len(c.Args) < 2 {
				return fmt.Errorf("usage: buffalo task importmap:pin <name> <url>")
			}

			name := c.Args[0]
			url := c.Args[1]

			// Check if URL or local path
			if !strings.HasPrefix(url, "http") && !strings.HasPrefix(url, "/") {
				// Assume it's a package name, use default CDN
				url = fmt.Sprintf("https://esm.sh/%s", url)
			}

			manager.Pin(name, url)
			fmt.Printf("✓ Pinned %s to %s\n", name, url)

			// Save to file
			if err := manager.SaveToFile("config/importmap.json"); err != nil {
				return fmt.Errorf("failed to save import map: %w", err)
			}

			return nil
		})

		_ = grift.Desc("unpin", "Remove a package from the import map")
		_ = grift.Add("unpin", func(c *grift.Context) error {
			if len(c.Args) < 1 {
				return fmt.Errorf("usage: buffalo task importmap:unpin <name>")
			}

			name := c.Args[0]
			manager.Unpin(name)
			fmt.Printf("✓ Unpinned %s\n", name)

			// Save to file
			if err := manager.SaveToFile("config/importmap.json"); err != nil {
				return fmt.Errorf("failed to save import map: %w", err)
			}

			return nil
		})

		_ = grift.Desc("list", "List all pinned packages")
		_ = grift.Add("list", func(c *grift.Context) error {
			imports := manager.List()

			if len(imports) == 0 {
				fmt.Println("No packages pinned")
				return nil
			}

			fmt.Println("Pinned packages:")
			fmt.Println("================")

			maxNameLen := 0
			for name := range imports {
				if len(name) > maxNameLen {
					maxNameLen = len(name)
				}
			}

			for name, url := range imports {
				integrity := manager.GetIntegrity(name)
				if integrity != "" {
					fmt.Printf("  %-*s → %s (vendored, integrity: %s...)\n",
						maxNameLen, name, url, integrity[:20])
				} else {
					fmt.Printf("  %-*s → %s\n", maxNameLen, name, url)
				}
			}

			return nil
		})

		_ = grift.Desc("vendor", "Download all remote packages to local vendor directory")
		_ = grift.Add("vendor", func(c *grift.Context) error {
			fmt.Println("Vendoring remote packages...")

			// Load current import map
			if err := manager.LoadFromFile("config/importmap.json"); err != nil {
				fmt.Printf("Warning: Could not load import map: %v\n", err)
			}

			// Download all remote packages
			imports := manager.List()
			vendored := 0

			for name, url := range imports {
				if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
					fmt.Printf("  Downloading %s from %s...\n", name, url)
					if err := manager.Download(name); err != nil {
						fmt.Printf("    ✗ Failed: %v\n", err)
					} else {
						fmt.Printf("    ✓ Vendored with integrity hash\n")
						vendored++
					}
				}
			}

			// Save updated import map with local paths
			if err := manager.SaveToFile("config/importmap.json"); err != nil {
				return fmt.Errorf("failed to save import map: %w", err)
			}

			fmt.Printf("\n✓ Vendored %d packages\n", vendored)
			return nil
		})

		_ = grift.Desc("update", "Update all vendored packages to latest versions")
		_ = grift.Add("update", func(c *grift.Context) error {
			fmt.Println("Updating vendored packages...")

			// Load current import map
			if err := manager.LoadFromFile("config/importmap.json"); err != nil {
				fmt.Printf("Warning: Could not load import map: %v\n", err)
			}

			// Update all packages
			if err := manager.UpdateAll(); err != nil {
				return fmt.Errorf("failed to update packages: %w", err)
			}

			// Save updated import map
			if err := manager.SaveToFile("config/importmap.json"); err != nil {
				return fmt.Errorf("failed to save import map: %w", err)
			}

			fmt.Println("✓ All packages updated")
			return nil
		})

		_ = grift.Desc("init", "Initialize import map with default packages")
		_ = grift.Add("init", func(c *grift.Context) error {
			fmt.Println("Initializing import map with defaults...")

			// Load defaults
			manager.LoadDefaults()

			// Create config directory if it doesn't exist
			if err := os.MkdirAll("config", 0755); err != nil {
				return fmt.Errorf("failed to create config directory: %w", err)
			}

			// Save to file
			if err := manager.SaveToFile("config/importmap.json"); err != nil {
				return fmt.Errorf("failed to save import map: %w", err)
			}

			fmt.Println("✓ Import map initialized with defaults:")
			imports := manager.List()
			for name, url := range imports {
				fmt.Printf("  %s → %s\n", name, url)
			}

			return nil
		})

		_ = grift.Desc("clean", "Remove unused vendored files")
		_ = grift.Add("clean", func(c *grift.Context) error {
			fmt.Println("Cleaning vendor directory...")

			// This would remove files not referenced in the current import map
			// For now, just report what would be cleaned

			vendorDir := "public/assets/vendor"
			entries, err := os.ReadDir(vendorDir)
			if err != nil {
				if os.IsNotExist(err) {
					fmt.Println("No vendor directory found")
					return nil
				}
				return err
			}

			fmt.Printf("Found %d files in vendor directory\n", len(entries))
			fmt.Println("✓ Clean complete (dry run - no files removed)")

			return nil
		})
	})
}

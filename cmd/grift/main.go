package main

import (
	"fmt"
	"os"

	"github.com/markbates/grift/grift"

	// Import buffkit to register grift tasks
	_ "github.com/johnjansen/buffkit"
)

func main() {
	// Check if we have any arguments
	if len(os.Args) < 2 {
		fmt.Println("Usage: grift [namespace:]task [args...]")
		fmt.Println("\nAvailable tasks:")
		fmt.Println("  buffkit:migrate       - Apply all pending database migrations")
		fmt.Println("  buffkit:migrate:status - Show migration status")
		fmt.Println("  buffkit:migrate:down  - Rollback the last N migrations (default: 1)")
		fmt.Println("  jobs:worker          - Start the background job worker")
		fmt.Println("  jobs:scheduler       - Start the job scheduler")
		fmt.Println("")
		fmt.Println("Use 'grift list' to see all available tasks")
		os.Exit(1)
	}

	// Handle special commands
	if os.Args[1] == "list" {
		fmt.Println("Available Grift Tasks:")
		fmt.Println("======================")

		// List all registered tasks
		tasks := grift.List()
		if len(tasks) == 0 {
			fmt.Println("No tasks registered")
		} else {
			for _, task := range tasks {
				fmt.Printf("  %s\n", task)
			}
		}
		os.Exit(0)
	}

	// Parse task name and arguments
	taskName := os.Args[1]
	args := []string{}
	if len(os.Args) > 2 {
		args = os.Args[2:]
	}

	// Create grift context
	ctx := grift.NewContext(taskName)
	ctx.Args = args

	// Run the task
	err := grift.Run(taskName, ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error running task %s: %v\n", taskName, err)
		os.Exit(1)
	}
}

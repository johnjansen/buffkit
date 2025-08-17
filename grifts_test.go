package buffkit

import (
	"testing"

	"github.com/markbates/grift/grift"
	"github.com/stretchr/testify/assert"
)

func TestGriftTasksRegistered(t *testing.T) {
	// The init() function in grifts.go should register tasks
	// Let's verify they actually exist by checking the list

	expectedTasks := []string{
		"buffkit:migrate",
		"buffkit:migrate:status",
		"buffkit:migrate:down",
		"buffkit:migrate:create",
		"jobs:worker",
		"jobs:enqueue",
		"jobs:stats",
	}

	// Get all registered tasks
	registeredTasks := grift.List()

	for _, expected := range expectedTasks {
		t.Run(expected, func(t *testing.T) {
			found := false
			for _, registered := range registeredTasks {
				if registered == expected {
					found = true
					break
				}
			}
			assert.True(t, found, "Task %s should be registered", expected)
		})
	}
}

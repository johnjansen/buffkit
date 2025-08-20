package auth

import (
	"context"
	"time"
)

// ExtendedUserStore is a stub interface to satisfy jobs package compilation
// This is NOT part of the BDD feature requirements and should be removed
// once jobs package is properly tested with BDD-first approach
type ExtendedUserStore interface {
	UserStore

	// Minimal stub methods to satisfy compilation
	ByID(ctx context.Context, id string) (*User, error)
	IncrementFailedLoginAttempts(ctx context.Context, email string) error
	ResetFailedLoginAttempts(ctx context.Context, email string) error
	CleanupSessions(ctx context.Context, maxAge, maxInactivity time.Duration) (int, error)
}

// Make MemoryStore implement ExtendedUserStore minimally
func (m *MemoryStore) IncrementFailedLoginAttempts(ctx context.Context, email string) error {
	// Stub - do nothing for now
	return nil
}

func (m *MemoryStore) ResetFailedLoginAttempts(ctx context.Context, email string) error {
	// Stub - do nothing for now
	return nil
}

func (m *MemoryStore) CleanupSessions(ctx context.Context, maxAge, maxInactivity time.Duration) (int, error) {
	// Stub - do nothing for now, return 0 sessions cleaned
	return 0, nil
}

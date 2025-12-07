package repository

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestMemoryUserRepository_Create(t *testing.T) {
	repo := NewMemoryUserRepository()
	ctx := context.Background()

	user := &User{
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "hash123",
		FullName:     "Test User",
		Role:         "user",
	}

	err := repo.Create(ctx, user)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if user.ID == "" {
		t.Error("Create() should set user ID")
	}
	if user.CreatedAt.IsZero() {
		t.Error("Create() should set CreatedAt")
	}
	if user.UpdatedAt.IsZero() {
		t.Error("Create() should set UpdatedAt")
	}
}

func TestMemoryUserRepository_Create_Duplicate(t *testing.T) {
	repo := NewMemoryUserRepository()
	ctx := context.Background()

	user1 := &User{
		Username: "testuser",
		Email:    "test1@example.com",
	}
	user2 := &User{
		Username: "testuser",
		Email:    "test2@example.com",
	}
	user3 := &User{
		Username: "other",
		Email:    "test1@example.com",
	}

	err := repo.Create(ctx, user1)
	if err != nil {
		t.Fatalf("First Create() error = %v", err)
	}

	// Duplicate username
	err = repo.Create(ctx, user2)
	if err != ErrUserAlreadyExists {
		t.Errorf("Create() with duplicate username should return ErrUserAlreadyExists, got %v", err)
	}

	// Duplicate email
	err = repo.Create(ctx, user3)
	if err != ErrUserAlreadyExists {
		t.Errorf("Create() with duplicate email should return ErrUserAlreadyExists, got %v", err)
	}
}

func TestMemoryUserRepository_GetByID(t *testing.T) {
	repo := NewMemoryUserRepository()
	ctx := context.Background()

	user := &User{
		Username: "testuser",
		Email:    "test@example.com",
	}
	_ = repo.Create(ctx, user)

	// Get existing user
	found, err := repo.GetByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if found.Username != user.Username {
		t.Errorf("GetByID() username = %v, want %v", found.Username, user.Username)
	}

	// Get non-existing user
	_, err = repo.GetByID(ctx, "non-existing-id")
	if err != ErrUserNotFound {
		t.Errorf("GetByID() for non-existing user should return ErrUserNotFound, got %v", err)
	}
}

func TestMemoryUserRepository_GetByUsername(t *testing.T) {
	repo := NewMemoryUserRepository()
	ctx := context.Background()

	user := &User{
		Username: "testuser",
		Email:    "test@example.com",
	}
	_ = repo.Create(ctx, user)

	found, err := repo.GetByUsername(ctx, "testuser")
	if err != nil {
		t.Fatalf("GetByUsername() error = %v", err)
	}
	if found.ID != user.ID {
		t.Errorf("GetByUsername() returned wrong user")
	}

	_, err = repo.GetByUsername(ctx, "nonexistent")
	if err != ErrUserNotFound {
		t.Errorf("GetByUsername() for non-existing user should return ErrUserNotFound")
	}
}

func TestMemoryUserRepository_GetByEmail(t *testing.T) {
	repo := NewMemoryUserRepository()
	ctx := context.Background()

	user := &User{
		Username: "testuser",
		Email:    "test@example.com",
	}
	_ = repo.Create(ctx, user)

	found, err := repo.GetByEmail(ctx, "test@example.com")
	if err != nil {
		t.Fatalf("GetByEmail() error = %v", err)
	}
	if found.ID != user.ID {
		t.Errorf("GetByEmail() returned wrong user")
	}
}

func TestMemoryUserRepository_Update(t *testing.T) {
	repo := NewMemoryUserRepository()
	ctx := context.Background()

	user := &User{
		Username: "testuser",
		Email:    "test@example.com",
		FullName: "Original Name",
	}
	_ = repo.Create(ctx, user)

	// Update user
	user.FullName = "Updated Name"
	user.Username = "newusername"
	err := repo.Update(ctx, user)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	// Verify update
	found, _ := repo.GetByID(ctx, user.ID)
	if found.FullName != "Updated Name" {
		t.Errorf("Update() did not update FullName")
	}

	// Can find by new username
	_, err = repo.GetByUsername(ctx, "newusername")
	if err != nil {
		t.Errorf("Should find user by new username")
	}

	// Cannot find by old username
	_, err = repo.GetByUsername(ctx, "testuser")
	if err != ErrUserNotFound {
		t.Errorf("Should not find user by old username")
	}
}

func TestMemoryUserRepository_Delete(t *testing.T) {
	repo := NewMemoryUserRepository()
	ctx := context.Background()

	user := &User{
		Username: "testuser",
		Email:    "test@example.com",
	}
	_ = repo.Create(ctx, user)

	// Delete user
	err := repo.Delete(ctx, user.ID)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Cannot find deleted user
	_, err = repo.GetByID(ctx, user.ID)
	if err != ErrUserNotFound {
		t.Errorf("Deleted user should not be found")
	}

	// Delete non-existing
	err = repo.Delete(ctx, "non-existing")
	if err != ErrUserNotFound {
		t.Errorf("Delete() non-existing should return ErrUserNotFound")
	}
}

func TestMemoryUserRepository_Exists(t *testing.T) {
	repo := NewMemoryUserRepository()
	ctx := context.Background()

	user := &User{
		Username: "testuser",
		Email:    "test@example.com",
	}
	_ = repo.Create(ctx, user)

	// Existing by username
	exists, err := repo.Exists(ctx, "testuser", "other@email.com")
	if err != nil || !exists {
		t.Errorf("Exists() should return true for existing username")
	}

	// Existing by email
	exists, err = repo.Exists(ctx, "other", "test@example.com")
	if err != nil || !exists {
		t.Errorf("Exists() should return true for existing email")
	}

	// Not existing
	exists, err = repo.Exists(ctx, "other", "other@email.com")
	if err != nil || exists {
		t.Errorf("Exists() should return false for non-existing user")
	}
}

func TestMemoryUserRepository_Concurrent(t *testing.T) {
	repo := NewMemoryUserRepository()
	ctx := context.Background()

	var wg sync.WaitGroup
	numGoroutines := 100

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(idx int) {
			defer wg.Done()

			user := &User{
				Username: fmt.Sprintf("user%d", idx),
				Email:    fmt.Sprintf("user%d@example.com", idx),
			}
			_ = repo.Create(ctx, user)

			_, _ = repo.GetByUsername(ctx, user.Username)
		}(i)
	}

	wg.Wait()
}

// Token Blacklist Tests

func TestMemoryTokenBlacklist_Add_Contains(t *testing.T) {
	bl := NewMemoryTokenBlacklist()
	ctx := context.Background()

	token := "test-token-123"

	// Initially not in blacklist
	contains, err := bl.Contains(ctx, token)
	if err != nil || contains {
		t.Errorf("New token should not be in blacklist")
	}

	// Add to blacklist
	err = bl.Add(ctx, token, time.Hour)
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	// Now should be in blacklist
	contains, err = bl.Contains(ctx, token)
	if err != nil || !contains {
		t.Errorf("Added token should be in blacklist")
	}
}

func TestMemoryTokenBlacklist_Expiry(t *testing.T) {
	bl := NewMemoryTokenBlacklist()
	ctx := context.Background()

	token := "expiring-token"

	// Add with very short expiry
	err := bl.Add(ctx, token, 1*time.Millisecond)
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	// Wait for expiry
	time.Sleep(10 * time.Millisecond)

	// Should not be in blacklist (expired)
	contains, err := bl.Contains(ctx, token)
	if err != nil {
		t.Fatalf("Contains() error = %v", err)
	}
	if contains {
		t.Errorf("Expired token should not be in blacklist")
	}
}

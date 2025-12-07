package repository

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
)

// MemoryUserRepository in-memory реализация UserRepository
type MemoryUserRepository struct {
	mu         sync.RWMutex
	users      map[string]*User  // id -> user
	byUsername map[string]string // username -> id
	byEmail    map[string]string // email -> id
}

// NewMemoryUserRepository создаёт новый in-memory репозиторий
func NewMemoryUserRepository() *MemoryUserRepository {
	return &MemoryUserRepository{
		users:      make(map[string]*User),
		byUsername: make(map[string]string),
		byEmail:    make(map[string]string),
	}
}

func (r *MemoryUserRepository) Create(ctx context.Context, user *User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Проверяем уникальность
	if _, exists := r.byUsername[user.Username]; exists {
		return ErrUserAlreadyExists
	}
	if _, exists := r.byEmail[user.Email]; exists {
		return ErrUserAlreadyExists
	}

	// Генерируем ID если не задан
	if user.ID == "" {
		user.ID = uuid.New().String()
	}

	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now

	// Сохраняем копию
	storedUser := *user
	r.users[user.ID] = &storedUser
	r.byUsername[user.Username] = user.ID
	r.byEmail[user.Email] = user.ID

	return nil
}

func (r *MemoryUserRepository) GetByID(ctx context.Context, id string) (*User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	user, exists := r.users[id]
	if !exists {
		return nil, ErrUserNotFound
	}

	// Возвращаем копию
	result := *user
	return &result, nil
}

func (r *MemoryUserRepository) GetByUsername(ctx context.Context, username string) (*User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	id, exists := r.byUsername[username]
	if !exists {
		return nil, ErrUserNotFound
	}

	user := r.users[id]
	result := *user
	return &result, nil
}

func (r *MemoryUserRepository) GetByEmail(ctx context.Context, email string) (*User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	id, exists := r.byEmail[email]
	if !exists {
		return nil, ErrUserNotFound
	}

	user := r.users[id]
	result := *user
	return &result, nil
}

func (r *MemoryUserRepository) Update(ctx context.Context, user *User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, exists := r.users[user.ID]
	if !exists {
		return ErrUserNotFound
	}

	// Обновляем индексы если изменились username/email
	if existing.Username != user.Username {
		delete(r.byUsername, existing.Username)
		r.byUsername[user.Username] = user.ID
	}
	if existing.Email != user.Email {
		delete(r.byEmail, existing.Email)
		r.byEmail[user.Email] = user.ID
	}

	user.UpdatedAt = time.Now()
	storedUser := *user
	r.users[user.ID] = &storedUser

	return nil
}

func (r *MemoryUserRepository) Delete(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	user, exists := r.users[id]
	if !exists {
		return ErrUserNotFound
	}

	delete(r.byUsername, user.Username)
	delete(r.byEmail, user.Email)
	delete(r.users, id)

	return nil
}

func (r *MemoryUserRepository) Exists(ctx context.Context, username, email string) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if _, exists := r.byUsername[username]; exists {
		return true, nil
	}
	if _, exists := r.byEmail[email]; exists {
		return true, nil
	}
	return false, nil
}

// MemoryTokenBlacklist in-memory реализация TokenBlacklist
type MemoryTokenBlacklist struct {
	mu     sync.RWMutex
	tokens map[string]time.Time // token -> expiry
}

// NewMemoryTokenBlacklist создаёт новый blacklist
func NewMemoryTokenBlacklist() *MemoryTokenBlacklist {
	bl := &MemoryTokenBlacklist{
		tokens: make(map[string]time.Time),
	}
	// Запускаем фоновую очистку
	go bl.cleanup()
	return bl
}

func (bl *MemoryTokenBlacklist) Add(ctx context.Context, token string, expiry time.Duration) error {
	bl.mu.Lock()
	defer bl.mu.Unlock()

	bl.tokens[token] = time.Now().Add(expiry)
	return nil
}

func (bl *MemoryTokenBlacklist) Contains(ctx context.Context, token string) (bool, error) {
	bl.mu.RLock()
	defer bl.mu.RUnlock()

	expiry, exists := bl.tokens[token]
	if !exists {
		return false, nil
	}

	// Проверяем не истёк ли токен в blacklist
	if time.Now().After(expiry) {
		return false, nil
	}

	return true, nil
}

func (bl *MemoryTokenBlacklist) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		bl.mu.Lock()
		now := time.Now()
		for token, expiry := range bl.tokens {
			if now.After(expiry) {
				delete(bl.tokens, token)
			}
		}
		bl.mu.Unlock()
	}
}

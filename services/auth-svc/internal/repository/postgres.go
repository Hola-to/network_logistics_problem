package repository

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"logistics/pkg/database"
	"logistics/pkg/logger"
	"logistics/pkg/telemetry"
)

// PostgresUserRepository PostgreSQL реализация UserRepository
type PostgresUserRepository struct {
	db database.DB
}

// NewPostgresUserRepository создаёт новый PostgreSQL репозиторий
func NewPostgresUserRepository(db database.DB) *PostgresUserRepository {
	return &PostgresUserRepository{db: db}
}

func (r *PostgresUserRepository) Create(ctx context.Context, user *User) error {
	ctx, span := telemetry.StartSpan(ctx, "PostgresUserRepository.Create")
	defer span.End()

	query := `
		INSERT INTO users (username, email, password_hash, full_name, role)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`

	err := r.db.QueryRow(ctx, query,
		user.Username,
		user.Email,
		user.PasswordHash,
		user.FullName,
		user.Role,
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		if isUniqueViolation(err) {
			return ErrUserAlreadyExists
		}
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

func (r *PostgresUserRepository) GetByID(ctx context.Context, id string) (*User, error) {
	ctx, span := telemetry.StartSpan(ctx, "PostgresUserRepository.GetByID")
	defer span.End()

	query := `
		SELECT id, username, email, password_hash, full_name, role, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	user := &User{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.FullName,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by id: %w", err)
	}

	return user, nil
}

func (r *PostgresUserRepository) GetByUsername(ctx context.Context, username string) (*User, error) {
	ctx, span := telemetry.StartSpan(ctx, "PostgresUserRepository.GetByUsername")
	defer span.End()

	query := `
		SELECT id, username, email, password_hash, full_name, role, created_at, updated_at
		FROM users
		WHERE username = $1
	`

	user := &User{}
	err := r.db.QueryRow(ctx, query, username).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.FullName,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by username: %w", err)
	}

	return user, nil
}

func (r *PostgresUserRepository) GetByEmail(ctx context.Context, email string) (*User, error) {
	ctx, span := telemetry.StartSpan(ctx, "PostgresUserRepository.GetByEmail")
	defer span.End()

	query := `
		SELECT id, username, email, password_hash, full_name, role, created_at, updated_at
		FROM users
		WHERE email = $1
	`

	user := &User{}
	err := r.db.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.FullName,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	return user, nil
}

func (r *PostgresUserRepository) Update(ctx context.Context, user *User) error {
	ctx, span := telemetry.StartSpan(ctx, "PostgresUserRepository.Update")
	defer span.End()

	query := `
		UPDATE users
		SET username = $2, email = $3, password_hash = $4, full_name = $5, role = $6
		WHERE id = $1
		RETURNING updated_at
	`

	err := r.db.QueryRow(ctx, query,
		user.ID,
		user.Username,
		user.Email,
		user.PasswordHash,
		user.FullName,
		user.Role,
	).Scan(&user.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrUserNotFound
		}
		if isUniqueViolation(err) {
			return ErrUserAlreadyExists
		}
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

func (r *PostgresUserRepository) Delete(ctx context.Context, id string) error {
	ctx, span := telemetry.StartSpan(ctx, "PostgresUserRepository.Delete")
	defer span.End()

	query := `DELETE FROM users WHERE id = $1`

	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrUserNotFound
	}

	return nil
}

func (r *PostgresUserRepository) Exists(ctx context.Context, username, email string) (bool, error) {
	ctx, span := telemetry.StartSpan(ctx, "PostgresUserRepository.Exists")
	defer span.End()

	query := `
		SELECT EXISTS(
			SELECT 1 FROM users WHERE username = $1 OR email = $2
		)
	`

	var exists bool
	err := r.db.QueryRow(ctx, query, username, email).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check user existence: %w", err)
	}

	return exists, nil
}

// Дополнительные методы для PostgresUserRepository

// List возвращает список пользователей с пагинацией
func (r *PostgresUserRepository) List(ctx context.Context, limit, offset int) ([]*User, int64, error) {
	ctx, span := telemetry.StartSpan(ctx, "PostgresUserRepository.List")
	defer span.End()

	// Получаем общее количество
	var total int64
	countQuery := `SELECT COUNT(*) FROM users`
	if err := r.db.QueryRow(ctx, countQuery).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count users: %w", err)
	}

	// Получаем пользователей
	query := `
		SELECT id, username, email, password_hash, full_name, role, created_at, updated_at
		FROM users
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		user := &User{}
		err := rows.Scan(
			&user.ID,
			&user.Username,
			&user.Email,
			&user.PasswordHash,
			&user.FullName,
			&user.Role,
			&user.CreatedAt,
			&user.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows iteration error: %w", err)
	}

	return users, total, nil
}

// UpdatePassword обновляет только пароль пользователя
func (r *PostgresUserRepository) UpdatePassword(ctx context.Context, id, passwordHash string) error {
	ctx, span := telemetry.StartSpan(ctx, "PostgresUserRepository.UpdatePassword")
	defer span.End()

	query := `UPDATE users SET password_hash = $2 WHERE id = $1`

	result, err := r.db.Exec(ctx, query, id, passwordHash)
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrUserNotFound
	}

	return nil
}

// UpdateRole обновляет роль пользователя
func (r *PostgresUserRepository) UpdateRole(ctx context.Context, id, role string) error {
	ctx, span := telemetry.StartSpan(ctx, "PostgresUserRepository.UpdateRole")
	defer span.End()

	query := `UPDATE users SET role = $2 WHERE id = $1`

	result, err := r.db.Exec(ctx, query, id, role)
	if err != nil {
		return fmt.Errorf("failed to update role: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrUserNotFound
	}

	return nil
}

// isUniqueViolation проверяет, является ли ошибка нарушением уникальности
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505" // unique_violation
	}
	return false
}

// PostgresTokenBlacklist PostgreSQL реализация TokenBlacklist
type PostgresTokenBlacklist struct {
	db database.DB
}

// NewPostgresTokenBlacklist создаёт новый PostgreSQL blacklist
func NewPostgresTokenBlacklist(db database.DB) *PostgresTokenBlacklist {
	bl := &PostgresTokenBlacklist{db: db}
	// Запускаем фоновую очистку
	go bl.startCleanup()
	return bl
}

func (bl *PostgresTokenBlacklist) Add(ctx context.Context, token string, expiry time.Duration) error {
	ctx, span := telemetry.StartSpan(ctx, "PostgresTokenBlacklist.Add")
	defer span.End()

	tokenHash := hashToken(token)
	expiresAt := time.Now().Add(expiry)

	query := `
		INSERT INTO token_blacklist (token_hash, expires_at)
		VALUES ($1, $2)
		ON CONFLICT (token_hash) DO UPDATE SET expires_at = $2
	`

	_, err := bl.db.Exec(ctx, query, tokenHash, expiresAt)
	if err != nil {
		return fmt.Errorf("failed to add token to blacklist: %w", err)
	}

	return nil
}

func (bl *PostgresTokenBlacklist) Contains(ctx context.Context, token string) (bool, error) {
	ctx, span := telemetry.StartSpan(ctx, "PostgresTokenBlacklist.Contains")
	defer span.End()

	tokenHash := hashToken(token)

	query := `
		SELECT EXISTS(
			SELECT 1 FROM token_blacklist
			WHERE token_hash = $1 AND expires_at > NOW()
		)
	`

	var exists bool
	err := bl.db.QueryRow(ctx, query, tokenHash).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check token in blacklist: %w", err)
	}

	return exists, nil
}

// Remove удаляет токен из blacklist (опционально)
func (bl *PostgresTokenBlacklist) Remove(ctx context.Context, token string) error {
	ctx, span := telemetry.StartSpan(ctx, "PostgresTokenBlacklist.Remove")
	defer span.End()

	tokenHash := hashToken(token)

	query := `DELETE FROM token_blacklist WHERE token_hash = $1`

	_, err := bl.db.Exec(ctx, query, tokenHash)
	if err != nil {
		return fmt.Errorf("failed to remove token from blacklist: %w", err)
	}

	return nil
}

// Cleanup удаляет устаревшие токены
func (bl *PostgresTokenBlacklist) Cleanup(ctx context.Context) (int64, error) {
	ctx, span := telemetry.StartSpan(ctx, "PostgresTokenBlacklist.Cleanup")
	defer span.End()

	query := `DELETE FROM token_blacklist WHERE expires_at < NOW()`

	result, err := bl.db.Exec(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup blacklist: %w", err)
	}

	return result.RowsAffected(), nil
}

// startCleanup запускает периодическую очистку
func (bl *PostgresTokenBlacklist) startCleanup() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		deleted, err := bl.Cleanup(ctx)
		cancel()

		if err != nil {
			logger.Log.Debug("Cleanup Error", "message", err.Error())
			continue
		}

		if deleted > 0 {
			logger.Log.Debug("Deleted expired tokens", "count", deleted)
		}
	}
}

// hashToken хеширует токен для хранения
func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

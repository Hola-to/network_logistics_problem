package service

import (
	"context"
	"testing"
	"time"

	authv1 "logistics/gen/go/logistics/auth/v1"
	"logistics/pkg/passhash"
	"logistics/services/auth-svc/internal/repository"
	"logistics/services/auth-svc/internal/token"
)

// Mock repositories for testing
type mockUserRepository struct {
	users       map[string]*repository.User
	byUsername  map[string]string
	returnError error
}

func newMockUserRepository() *mockUserRepository {
	return &mockUserRepository{
		users:      make(map[string]*repository.User),
		byUsername: make(map[string]string),
	}
}

func (m *mockUserRepository) Create(ctx context.Context, user *repository.User) error {
	if m.returnError != nil {
		return m.returnError
	}
	user.ID = "test-id-123"
	user.CreatedAt = time.Now()
	m.users[user.ID] = user
	m.byUsername[user.Username] = user.ID
	return nil
}

func (m *mockUserRepository) GetByID(ctx context.Context, id string) (*repository.User, error) {
	if m.returnError != nil {
		return nil, m.returnError
	}
	if user, ok := m.users[id]; ok {
		return user, nil
	}
	return nil, repository.ErrUserNotFound
}

func (m *mockUserRepository) GetByUsername(ctx context.Context, username string) (*repository.User, error) {
	if m.returnError != nil {
		return nil, m.returnError
	}
	if id, ok := m.byUsername[username]; ok {
		return m.users[id], nil
	}
	return nil, repository.ErrUserNotFound
}

func (m *mockUserRepository) GetByEmail(ctx context.Context, email string) (*repository.User, error) {
	return nil, repository.ErrUserNotFound
}

func (m *mockUserRepository) Update(ctx context.Context, user *repository.User) error {
	return nil
}

func (m *mockUserRepository) Delete(ctx context.Context, id string) error {
	return nil
}

func (m *mockUserRepository) Exists(ctx context.Context, username, email string) (bool, error) {
	_, ok := m.byUsername[username]
	return ok, nil
}

type mockBlacklist struct {
	tokens map[string]bool
}

func newMockBlacklist() *mockBlacklist {
	return &mockBlacklist{tokens: make(map[string]bool)}
}

func (m *mockBlacklist) Add(ctx context.Context, token string, expiry time.Duration) error {
	m.tokens[token] = true
	return nil
}

func (m *mockBlacklist) Contains(ctx context.Context, token string) (bool, error) {
	return m.tokens[token], nil
}

func createTestService(t *testing.T) (*AuthService, *mockUserRepository, *mockBlacklist) {
	userRepo := newMockUserRepository()
	blacklist := newMockBlacklist()
	tokenMgr := token.NewManager(&token.Config{
		SecretKey:          "test-secret-key-for-testing-only",
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 24 * time.Hour,
		Issuer:             "test",
	})

	svc := NewAuthService(userRepo, blacklist, tokenMgr)
	return svc, userRepo, blacklist
}

func TestAuthService_Register(t *testing.T) {
	svc, userRepo, _ := createTestService(t)
	ctx := context.Background()

	tests := []struct {
		name        string
		request     *authv1.RegisterRequest
		setup       func()
		wantSuccess bool
		wantErr     bool
	}{
		{
			name: "successful registration",
			request: &authv1.RegisterRequest{
				Username: "newuser",
				Password: "password123",
				Email:    "new@example.com",
				FullName: "New User",
			},
			wantSuccess: true,
			wantErr:     false,
		},
		{
			name: "short username",
			request: &authv1.RegisterRequest{
				Username: "ab",
				Password: "password123",
				Email:    "test@example.com",
			},
			wantSuccess: false,
			wantErr:     false,
		},
		{
			name: "short password",
			request: &authv1.RegisterRequest{
				Username: "testuser",
				Password: "short",
				Email:    "test@example.com",
			},
			wantSuccess: false,
			wantErr:     false,
		},
		{
			name: "invalid email",
			request: &authv1.RegisterRequest{
				Username: "testuser",
				Password: "password123",
				Email:    "invalid",
			},
			wantSuccess: false,
			wantErr:     false,
		},
		{
			name: "empty username",
			request: &authv1.RegisterRequest{
				Username: "",
				Password: "password123",
				Email:    "test@example.com",
			},
			wantSuccess: false,
			wantErr:     false,
		},
		{
			name: "duplicate username",
			request: &authv1.RegisterRequest{
				Username: "existing",
				Password: "password123",
				Email:    "new@example.com",
			},
			setup: func() {
				userRepo.byUsername["existing"] = "some-id"
			},
			wantSuccess: false,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}

			resp, err := svc.Register(ctx, tt.request)

			if (err != nil) != tt.wantErr {
				t.Errorf("Register() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if resp.Success != tt.wantSuccess {
				t.Errorf("Register() success = %v, want %v, message: %s",
					resp.Success, tt.wantSuccess, resp.ErrorMessage)
			}
		})
	}
}

func TestAuthService_Login(t *testing.T) {
	svc, userRepo, _ := createTestService(t)
	ctx := context.Background()

	// Create a test user
	passwordHash, _ := passhash.HashPassword("correctpassword")
	testUser := &repository.User{
		ID:           "user-123",
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: passwordHash,
		Role:         "user",
	}
	userRepo.users[testUser.ID] = testUser
	userRepo.byUsername[testUser.Username] = testUser.ID

	tests := []struct {
		name        string
		request     *authv1.LoginRequest
		wantSuccess bool
	}{
		{
			name: "successful login",
			request: &authv1.LoginRequest{
				Username: "testuser",
				Password: "correctpassword",
			},
			wantSuccess: true,
		},
		{
			name: "wrong password",
			request: &authv1.LoginRequest{
				Username: "testuser",
				Password: "wrongpassword",
			},
			wantSuccess: false,
		},
		{
			name: "non-existent user",
			request: &authv1.LoginRequest{
				Username: "nonexistent",
				Password: "password",
			},
			wantSuccess: false,
		},
		{
			name: "empty credentials",
			request: &authv1.LoginRequest{
				Username: "",
				Password: "",
			},
			wantSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := svc.Login(ctx, tt.request)
			if err != nil {
				t.Fatalf("Login() error = %v", err)
			}

			if resp.Success != tt.wantSuccess {
				t.Errorf("Login() success = %v, want %v, message: %s",
					resp.Success, tt.wantSuccess, resp.ErrorMessage)
			}

			if tt.wantSuccess {
				if resp.AccessToken == "" {
					t.Error("Login() should return access token")
				}
				if resp.RefreshToken == "" {
					t.Error("Login() should return refresh token")
				}
				if resp.User == nil {
					t.Error("Login() should return user info")
				}
			}
		})
	}
}

func TestAuthService_ValidateToken(t *testing.T) {
	svc, userRepo, blacklist := createTestService(t)
	ctx := context.Background()

	// Create test user and login
	passwordHash, _ := passhash.HashPassword("password")
	testUser := &repository.User{
		ID:           "user-123",
		Username:     "testuser",
		PasswordHash: passwordHash,
		Role:         "user",
	}
	userRepo.users[testUser.ID] = testUser
	userRepo.byUsername[testUser.Username] = testUser.ID

	loginResp, _ := svc.Login(ctx, &authv1.LoginRequest{
		Username: "testuser",
		Password: "password",
	})

	tests := []struct {
		name      string
		token     string
		setup     func()
		wantValid bool
	}{
		{
			name:      "valid token",
			token:     loginResp.AccessToken,
			wantValid: true,
		},
		{
			name:      "empty token",
			token:     "",
			wantValid: false,
		},
		{
			name:      "invalid token",
			token:     "invalid.token.here",
			wantValid: false,
		},
		{
			name:  "blacklisted token",
			token: loginResp.AccessToken,
			setup: func() {
				blacklist.tokens[loginResp.AccessToken] = true
			},
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}

			resp, err := svc.ValidateToken(ctx, &authv1.ValidateTokenRequest{
				Token: tt.token,
			})
			if err != nil {
				t.Fatalf("ValidateToken() error = %v", err)
			}

			if resp.Valid != tt.wantValid {
				t.Errorf("ValidateToken() valid = %v, want %v", resp.Valid, tt.wantValid)
			}
		})
	}
}

func TestAuthService_RefreshToken(t *testing.T) {
	svc, userRepo, blacklist := createTestService(t)
	ctx := context.Background()

	// Create test user and login
	passwordHash, _ := passhash.HashPassword("password")
	testUser := &repository.User{
		ID:           "user-123",
		Username:     "testuser",
		PasswordHash: passwordHash,
		Role:         "user",
	}
	userRepo.users[testUser.ID] = testUser
	userRepo.byUsername[testUser.Username] = testUser.ID

	loginResp, _ := svc.Login(ctx, &authv1.LoginRequest{
		Username: "testuser",
		Password: "password",
	})

	tests := []struct {
		name        string
		token       string
		setup       func()
		wantSuccess bool
	}{
		{
			name:        "valid refresh token",
			token:       loginResp.RefreshToken,
			wantSuccess: true,
		},
		{
			name:        "empty token",
			token:       "",
			wantSuccess: false,
		},
		{
			name:  "blacklisted token",
			token: loginResp.RefreshToken,
			setup: func() {
				blacklist.tokens[loginResp.RefreshToken] = true
			},
			wantSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset blacklist for each test
			blacklist.tokens = make(map[string]bool)
			if tt.setup != nil {
				tt.setup()
			}

			resp, err := svc.RefreshToken(ctx, &authv1.RefreshTokenRequest{
				RefreshToken: tt.token,
			})
			if err != nil {
				t.Fatalf("RefreshToken() error = %v", err)
			}

			if resp.Success != tt.wantSuccess {
				t.Errorf("RefreshToken() success = %v, want %v, message: %s",
					resp.Success, tt.wantSuccess, resp.ErrorMessage)
			}

			if tt.wantSuccess {
				if resp.AccessToken == "" {
					t.Error("RefreshToken() should return new access token")
				}
			}
		})
	}
}

func TestAuthService_Logout(t *testing.T) {
	svc, _, blacklist := createTestService(t)
	ctx := context.Background()

	tests := []struct {
		name           string
		token          string
		wantSuccess    bool
		checkBlacklist bool
	}{
		{
			name:           "logout with token",
			token:          "some-token",
			wantSuccess:    true,
			checkBlacklist: true,
		},
		{
			name:           "logout without token",
			token:          "",
			wantSuccess:    true,
			checkBlacklist: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blacklist.tokens = make(map[string]bool)

			resp, err := svc.Logout(ctx, &authv1.LogoutRequest{
				Token: tt.token,
			})
			if err != nil {
				t.Fatalf("Logout() error = %v", err)
			}

			if resp.Success != tt.wantSuccess {
				t.Errorf("Logout() success = %v, want %v", resp.Success, tt.wantSuccess)
			}

			if tt.checkBlacklist {
				if !blacklist.tokens[tt.token] {
					t.Error("Token should be added to blacklist")
				}
			}
		})
	}
}

func TestValidateRegisterRequest(t *testing.T) {
	tests := []struct {
		name    string
		request *authv1.RegisterRequest
		wantErr bool
	}{
		{
			name: "valid request",
			request: &authv1.RegisterRequest{
				Username: "testuser",
				Password: "password123",
				Email:    "test@example.com",
			},
			wantErr: false,
		},
		{
			name: "empty username",
			request: &authv1.RegisterRequest{
				Username: "",
				Password: "password123",
				Email:    "test@example.com",
			},
			wantErr: true,
		},
		{
			name: "short username",
			request: &authv1.RegisterRequest{
				Username: "ab",
				Password: "password123",
				Email:    "test@example.com",
			},
			wantErr: true,
		},
		{
			name: "empty password",
			request: &authv1.RegisterRequest{
				Username: "testuser",
				Password: "",
				Email:    "test@example.com",
			},
			wantErr: true,
		},
		{
			name: "short password",
			request: &authv1.RegisterRequest{
				Username: "testuser",
				Password: "short",
				Email:    "test@example.com",
			},
			wantErr: true,
		},
		{
			name: "empty email",
			request: &authv1.RegisterRequest{
				Username: "testuser",
				Password: "password123",
				Email:    "",
			},
			wantErr: true,
		},
		{
			name: "invalid email - no @",
			request: &authv1.RegisterRequest{
				Username: "testuser",
				Password: "password123",
				Email:    "invalid",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRegisterRequest(tt.request)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateRegisterRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

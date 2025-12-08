package v1_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authv1 "logistics/gen/go/logistics/auth/v1"
	"logistics/tests/integration/testutil"
)

func TestAuthService_Register(t *testing.T) {
	client := SetupAuthClient(t)
	ctx, cancel := testutil.Context(t)
	defer cancel()

	username := "testuser_" + testutil.RandomString(8)
	email := username + "@test.com"

	resp, err := client.Register(ctx, &authv1.RegisterRequest{
		Username: username,
		Email:    email,
		Password: "SecurePassword123!",
		FullName: "Test User",
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.Success)
	assert.NotEmpty(t, resp.UserId)
}

func TestAuthService_RegisterDuplicate(t *testing.T) {
	client := SetupAuthClient(t)
	ctx, cancel := testutil.Context(t)
	defer cancel()

	username := "dupuser_" + testutil.RandomString(8)
	email := username + "@test.com"

	// First registration should succeed
	_, err := client.Register(ctx, &authv1.RegisterRequest{
		Username: username,
		Email:    email,
		Password: "Password123!",
		FullName: "Test User",
	})
	require.NoError(t, err)

	// Second registration with same username should fail
	resp, err := client.Register(ctx, &authv1.RegisterRequest{
		Username: username,
		Email:    "different@test.com",
		Password: "Password123!",
		FullName: "Test User 2",
	})

	// May return error or unsuccessful response
	if err == nil {
		assert.False(t, resp.Success)
	}
}

func TestAuthService_Login(t *testing.T) {
	client := SetupAuthClient(t)
	ctx, cancel := testutil.Context(t)
	defer cancel()

	username := "logintest_" + testutil.RandomString(8)
	password := "TestPassword123!"

	// Register first
	_, err := client.Register(ctx, &authv1.RegisterRequest{
		Username: username,
		Email:    username + "@test.com",
		Password: password,
		FullName: "Login Test User",
	})
	require.NoError(t, err)

	// Then login
	resp, err := client.Login(ctx, &authv1.LoginRequest{
		Username: username,
		Password: password,
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.Success)
	assert.NotEmpty(t, resp.AccessToken)
	assert.NotEmpty(t, resp.RefreshToken)
	assert.Greater(t, resp.ExpiresIn, int64(0))
	assert.NotNil(t, resp.User)
	assert.Equal(t, username, resp.User.Username)
}

func TestAuthService_LoginInvalidCredentials(t *testing.T) {
	client := SetupAuthClient(t)
	ctx, cancel := testutil.Context(t)
	defer cancel()

	resp, err := client.Login(ctx, &authv1.LoginRequest{
		Username: "nonexistent_user",
		Password: "wrongpassword",
	})

	// Should return unsuccessful response
	if err == nil {
		require.NotNil(t, resp)
		assert.False(t, resp.Success)
		assert.NotEmpty(t, resp.ErrorMessage)
	}
}

func TestAuthService_ValidateToken(t *testing.T) {
	client := SetupAuthClient(t)
	ctx, cancel := testutil.Context(t)
	defer cancel()

	username := "tokentest_" + testutil.RandomString(8)
	password := "TestPassword123!"

	// Register and login
	_, err := client.Register(ctx, &authv1.RegisterRequest{
		Username: username,
		Email:    username + "@test.com",
		Password: password,
		FullName: "Token Test User",
	})
	require.NoError(t, err)

	loginResp, err := client.Login(ctx, &authv1.LoginRequest{
		Username: username,
		Password: password,
	})
	require.NoError(t, err)
	require.True(t, loginResp.Success)

	// Validate token
	validateResp, err := client.ValidateToken(ctx, &authv1.ValidateTokenRequest{
		Token: loginResp.AccessToken,
	})

	require.NoError(t, err)
	require.NotNil(t, validateResp)
	assert.True(t, validateResp.Valid)
	assert.NotEmpty(t, validateResp.UserId)
}

func TestAuthService_RefreshToken(t *testing.T) {
	client := SetupAuthClient(t)
	ctx, cancel := testutil.Context(t)
	defer cancel()

	username := "refreshtest_" + testutil.RandomString(8)
	password := "TestPassword123!"

	// Register and login
	_, err := client.Register(ctx, &authv1.RegisterRequest{
		Username: username,
		Email:    username + "@test.com",
		Password: password,
		FullName: "Refresh Test User",
	})
	require.NoError(t, err)

	loginResp, err := client.Login(ctx, &authv1.LoginRequest{
		Username: username,
		Password: password,
	})
	require.NoError(t, err)

	// Refresh token
	refreshResp, err := client.RefreshToken(ctx, &authv1.RefreshTokenRequest{
		RefreshToken: loginResp.RefreshToken,
	})

	require.NoError(t, err)
	require.NotNil(t, refreshResp)
	assert.True(t, refreshResp.Success)
	assert.NotEmpty(t, refreshResp.AccessToken)
	assert.NotEmpty(t, refreshResp.RefreshToken)
}

func TestAuthService_Logout(t *testing.T) {
	client := SetupAuthClient(t)
	ctx, cancel := testutil.Context(t)
	defer cancel()

	username := "logouttest_" + testutil.RandomString(8)
	password := "TestPassword123!"

	// Register and login
	_, err := client.Register(ctx, &authv1.RegisterRequest{
		Username: username,
		Email:    username + "@test.com",
		Password: password,
		FullName: "Logout Test User",
	})
	require.NoError(t, err)

	loginResp, err := client.Login(ctx, &authv1.LoginRequest{
		Username: username,
		Password: password,
	})
	require.NoError(t, err)

	// Logout
	logoutResp, err := client.Logout(ctx, &authv1.LogoutRequest{
		Token: loginResp.AccessToken,
	})

	require.NoError(t, err)
	require.NotNil(t, logoutResp)
	assert.True(t, logoutResp.Success)

	// Token should be invalid after logout
	validateResp, _ := client.ValidateToken(ctx, &authv1.ValidateTokenRequest{
		Token: loginResp.AccessToken,
	})
	if validateResp != nil {
		assert.False(t, validateResp.Valid)
	}
}

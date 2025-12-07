package benchmark

import (
	"testing"

	"logistics/pkg/passhash"
)

func BenchmarkHashPassword(b *testing.B) {
	password := "testPassword123!"

	for i := 0; i < b.N; i++ {
		passhash.HashPassword(password)
	}
}

func BenchmarkHashPassword_Parallel(b *testing.B) {
	password := "testPassword123!"

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			passhash.HashPassword(password)
		}
	})
}

func BenchmarkVerifyPassword(b *testing.B) {
	password := "testPassword123!"
	hash, _ := passhash.HashPassword(password)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		passhash.VerifyPassword(password, hash)
	}
}

func BenchmarkVerifyPassword_Parallel(b *testing.B) {
	password := "testPassword123!"
	hash, _ := passhash.HashPassword(password)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			passhash.VerifyPassword(password, hash)
		}
	})
}

func BenchmarkHashPasswordWithParams(b *testing.B) {
	password := "testPassword123!"

	params := []struct {
		name   string
		params *passhash.Argon2Params
	}{
		{
			name: "low",
			params: &passhash.Argon2Params{
				Memory:      32 * 1024,
				Iterations:  1,
				Parallelism: 1,
				SaltLength:  16,
				KeyLength:   32,
			},
		},
		{
			name:   "default",
			params: passhash.DefaultArgon2Params(),
		},
		{
			name: "high",
			params: &passhash.Argon2Params{
				Memory:      128 * 1024,
				Iterations:  4,
				Parallelism: 4,
				SaltLength:  16,
				KeyLength:   32,
			},
		},
	}

	for _, p := range params {
		b.Run(p.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				passhash.HashPasswordWithParams(password, p.params)
			}
		})
	}
}

func BenchmarkGenerateRandomString(b *testing.B) {
	lengths := []int{8, 16, 32, 64, 128}

	for _, length := range lengths {
		b.Run(string(rune('0'+length)), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				passhash.GenerateRandomString(length)
			}
		})
	}
}

func BenchmarkJWT_Generate(b *testing.B) {
	manager := passhash.NewJWTManager(nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager.GenerateAccessToken("user-123", "testuser", "user")
	}
}

func BenchmarkJWT_Validate(b *testing.B) {
	manager := passhash.NewJWTManager(nil)
	token, _ := manager.GenerateAccessToken("user-123", "testuser", "user")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager.ValidateToken(token)
	}
}

func BenchmarkJWT_GenerateValidate(b *testing.B) {
	manager := passhash.NewJWTManager(nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		token, _ := manager.GenerateAccessToken("user-123", "testuser", "user")
		manager.ValidateToken(token)
	}
}

func BenchmarkJWT_Parallel(b *testing.B) {
	manager := passhash.NewJWTManager(nil)

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			token, _ := manager.GenerateAccessToken("user-123", "testuser", "user")
			manager.ValidateToken(token)
			i++
		}
	})
}

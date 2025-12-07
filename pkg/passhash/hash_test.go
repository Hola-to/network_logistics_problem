package passhash

import (
	"strings"
	"testing"
)

func TestHashPassword(t *testing.T) {
	password := "securePassword123!"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	if hash == "" {
		t.Error("expected non-empty hash")
	}

	// Hash should start with $argon2id$
	if !strings.HasPrefix(hash, "$argon2id$") {
		t.Errorf("expected hash to start with $argon2id$, got %s", hash[:20])
	}

	// Hash should have 6 parts
	parts := strings.Split(hash, "$")
	if len(parts) != 6 {
		t.Errorf("expected 6 parts, got %d", len(parts))
	}
}

func TestHashPassword_DifferentSalts(t *testing.T) {
	password := "testPassword"

	hash1, _ := HashPassword(password)
	hash2, _ := HashPassword(password)

	// Same password should produce different hashes (different salts)
	if hash1 == hash2 {
		t.Error("expected different hashes for same password")
	}
}

func TestVerifyPassword(t *testing.T) {
	password := "correctPassword"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("failed to hash: %v", err)
	}

	// Correct password
	valid, err := VerifyPassword(password, hash)
	if err != nil {
		t.Fatalf("failed to verify: %v", err)
	}
	if !valid {
		t.Error("expected valid password to verify")
	}

	// Wrong password
	valid, err = VerifyPassword("wrongPassword", hash)
	if err != nil {
		t.Fatalf("failed to verify: %v", err)
	}
	if valid {
		t.Error("expected wrong password to not verify")
	}
}

func TestVerifyPassword_InvalidHash(t *testing.T) {
	tests := []struct {
		name string
		hash string
	}{
		{"empty", ""},
		{"invalid format", "not-a-valid-hash"},
		{"wrong parts", "$argon2id$v=19$m=65536"},
		{"wrong algorithm", "$bcrypt$v=19$m=65536,t=3,p=2$salt$hash"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := VerifyPassword("password", tt.hash)
			if err == nil {
				t.Error("expected error for invalid hash")
			}
		})
	}
}

func TestHashPasswordWithParams(t *testing.T) {
	password := "testPassword"
	params := &Argon2Params{
		Memory:      32 * 1024,
		Iterations:  2,
		Parallelism: 1,
		SaltLength:  8,
		KeyLength:   16,
	}

	hash, err := HashPasswordWithParams(password, params)
	if err != nil {
		t.Fatalf("failed to hash: %v", err)
	}

	valid, err := VerifyPassword(password, hash)
	if err != nil {
		t.Fatalf("failed to verify: %v", err)
	}
	if !valid {
		t.Error("expected password to verify with custom params")
	}
}

func TestDefaultArgon2Params(t *testing.T) {
	params := DefaultArgon2Params()

	if params.Memory != 64*1024 {
		t.Errorf("expected memory 64MB, got %d", params.Memory)
	}
	if params.Iterations != 3 {
		t.Errorf("expected 3 iterations, got %d", params.Iterations)
	}
	if params.Parallelism != 2 {
		t.Errorf("expected parallelism 2, got %d", params.Parallelism)
	}
	if params.SaltLength != 16 {
		t.Errorf("expected salt length 16, got %d", params.SaltLength)
	}
	if params.KeyLength != 32 {
		t.Errorf("expected key length 32, got %d", params.KeyLength)
	}
}

func TestGenerateRandomString(t *testing.T) {
	lengths := []int{8, 16, 32, 64}

	for _, length := range lengths {
		s, err := GenerateRandomString(length)
		if err != nil {
			t.Fatalf("failed to generate: %v", err)
		}
		if len(s) != length {
			t.Errorf("expected length %d, got %d", length, len(s))
		}
	}
}

func TestGenerateRandomString_Unique(t *testing.T) {
	s1, _ := GenerateRandomString(32)
	s2, _ := GenerateRandomString(32)

	if s1 == s2 {
		t.Error("expected unique random strings")
	}
}

func BenchmarkHashPassword(b *testing.B) {
	password := "benchmarkPassword"

	for i := 0; i < b.N; i++ {
		HashPassword(password)
	}
}

func BenchmarkVerifyPassword(b *testing.B) {
	password := "benchmarkPassword"
	hash, _ := HashPassword(password)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		VerifyPassword(password, hash)
	}
}

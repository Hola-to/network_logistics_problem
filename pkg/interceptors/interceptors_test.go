package interceptors

import (
	"context"
	"errors"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"logistics/pkg/logger"
)

func init() {
	logger.Init("error")
}

// Mock handler for testing
func mockHandler(_ context.Context, _ any) (any, error) {
	return "response", nil
}

func mockErrorHandler(_ context.Context, _ any) (any, error) {
	return nil, status.Error(codes.Internal, "internal error")
}

func mockPanicHandler(_ context.Context, _ any) (any, error) {
	panic("test panic")
}

func TestRecoveryInterceptor(t *testing.T) {
	interceptor := RecoveryInterceptor()

	t.Run("normal execution", func(t *testing.T) {
		resp, err := interceptor(
			context.Background(),
			"request",
			&grpc.UnaryServerInfo{FullMethod: "/test"},
			mockHandler,
		)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if resp != "response" {
			t.Errorf("unexpected response: %v", resp)
		}
	})

	t.Run("panic recovery", func(t *testing.T) {
		_, err := interceptor(
			context.Background(),
			"request",
			&grpc.UnaryServerInfo{FullMethod: "/test"},
			mockPanicHandler,
		)

		if err == nil {
			t.Error("expected error after panic")
		}

		st, ok := status.FromError(err)
		if !ok {
			t.Error("expected gRPC status error")
		}
		if st.Code() != codes.Internal {
			t.Errorf("expected Internal code, got %v", st.Code())
		}
	})
}

func TestLoggingInterceptor(t *testing.T) {
	interceptor := LoggingInterceptor()

	t.Run("successful request", func(t *testing.T) {
		resp, err := interceptor(
			context.Background(),
			"request",
			&grpc.UnaryServerInfo{FullMethod: "/test.Service/Method"},
			mockHandler,
		)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if resp != "response" {
			t.Errorf("unexpected response: %v", resp)
		}
	})

	t.Run("failed request", func(t *testing.T) {
		_, err := interceptor(
			context.Background(),
			"request",
			&grpc.UnaryServerInfo{FullMethod: "/test.Service/Method"},
			mockErrorHandler,
		)

		if err == nil {
			t.Error("expected error")
		}
	})
}

// Mock validator
type mockValidatable struct {
	shouldFail bool
}

func (m *mockValidatable) Validate() error {
	if m.shouldFail {
		return errors.New("validation failed")
	}
	return nil
}

func TestValidationInterceptor(t *testing.T) {
	interceptor := ValidationInterceptor()

	t.Run("valid request", func(t *testing.T) {
		req := &mockValidatable{shouldFail: false}
		_, err := interceptor(
			context.Background(),
			req,
			&grpc.UnaryServerInfo{FullMethod: "/test"},
			mockHandler,
		)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("invalid request", func(t *testing.T) {
		req := &mockValidatable{shouldFail: true}
		_, err := interceptor(
			context.Background(),
			req,
			&grpc.UnaryServerInfo{FullMethod: "/test"},
			mockHandler,
		)

		if err == nil {
			t.Error("expected error")
		}

		st, _ := status.FromError(err)
		if st.Code() != codes.InvalidArgument {
			t.Errorf("expected InvalidArgument, got %v", st.Code())
		}
	})

	t.Run("non-validatable request", func(t *testing.T) {
		_, err := interceptor(
			context.Background(),
			"string request",
			&grpc.UnaryServerInfo{FullMethod: "/test"},
			mockHandler,
		)

		if err != nil {
			t.Errorf("unexpected error for non-validatable: %v", err)
		}
	})
}

func TestChainUnaryInterceptors(t *testing.T) {
	var order []string

	interceptor1 := func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		order = append(order, "1-before")
		resp, err := handler(ctx, req)
		order = append(order, "1-after")
		return resp, err
	}

	interceptor2 := func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		order = append(order, "2-before")
		resp, err := handler(ctx, req)
		order = append(order, "2-after")
		return resp, err
	}

	chain := chainUnaryInterceptors(interceptor1, interceptor2)

	handler := func(_ context.Context, _ any) (any, error) {
		order = append(order, "handler")
		return "response", nil
	}

	_, _ = chain(context.Background(), "req", &grpc.UnaryServerInfo{}, handler)

	expected := []string{"1-before", "2-before", "handler", "2-after", "1-after"}
	if len(order) != len(expected) {
		t.Errorf("order length = %d, want %d", len(order), len(expected))
	}

	for i, v := range expected {
		if order[i] != v {
			t.Errorf("order[%d] = %s, want %s", i, order[i], v)
		}
	}
}

func TestMethodToAction(t *testing.T) {
	tests := []struct {
		method   string
		expected string
	}{
		{"/service/CreateUser", "CREATE"},
		{"/service/GetUser", "READ"},
		{"/service/UpdateUser", "UPDATE"},
		{"/service/DeleteUser", "DELETE"},
		{"/service/Login", "LOGIN"},
		{"/service/Logout", "LOGOUT"},
		{"/service/Solve", "SOLVE"},
		{"/service/Analyze", "ANALYZE"},
		{"/service/Unknown", "READ"},
	}

	for _, tt := range tests {
		action := methodToAction(tt.method)
		if string(action) != tt.expected {
			t.Errorf("methodToAction(%s) = %s, want %s", tt.method, action, tt.expected)
		}
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		s      string
		substr string
		want   bool
	}{
		{"CreateUser", "Create", true},
		{"GetUser", "User", true},
		{"Get", "Get", true},
		{"Get", "Set", false},
		{"", "a", false},
		{"abc", "", true},
	}

	for _, tt := range tests {
		got := contains(tt.s, tt.substr)
		if got != tt.want {
			t.Errorf("contains(%q, %q) = %v, want %v", tt.s, tt.substr, got, tt.want)
		}
	}
}

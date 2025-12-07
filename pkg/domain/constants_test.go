package domain

import (
	"math"
	"testing"
)

func TestIsVirtualNode(t *testing.T) {
	tests := []struct {
		nodeID   int64
		expected bool
	}{
		{SuperSourceID, true},
		{SuperSinkID, true},
		{-100, true},
		{0, false},
		{1, false},
		{100, false},
	}

	for _, tt := range tests {
		if got := IsVirtualNode(tt.nodeID); got != tt.expected {
			t.Errorf("IsVirtualNode(%d) = %v, want %v", tt.nodeID, got, tt.expected)
		}
	}
}

func TestFloatEquals(t *testing.T) {
	tests := []struct {
		a, b     float64
		expected bool
	}{
		{1.0, 1.0, true},
		{1.0, 1.0 + Epsilon/2, true},
		{1.0, 1.0 + Epsilon*2, false},
		{0, 0, true},
		{0, Epsilon / 2, true},
		{-1.0, -1.0, true},
	}

	for _, tt := range tests {
		if got := FloatEquals(tt.a, tt.b); got != tt.expected {
			t.Errorf("FloatEquals(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.expected)
		}
	}
}

func TestFloatLess(t *testing.T) {
	tests := []struct {
		a, b     float64
		expected bool
	}{
		{1.0, 2.0, true},
		{2.0, 1.0, false},
		{1.0, 1.0, false},
		{1.0, 1.0 + Epsilon/2, false}, // within epsilon
		{1.0, 1.0 + Epsilon*2, true},
	}

	for _, tt := range tests {
		if got := FloatLess(tt.a, tt.b); got != tt.expected {
			t.Errorf("FloatLess(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.expected)
		}
	}
}

func TestFloatGreater(t *testing.T) {
	tests := []struct {
		a, b     float64
		expected bool
	}{
		{2.0, 1.0, true},
		{1.0, 2.0, false},
		{1.0, 1.0, false},
		{1.0 + Epsilon/2, 1.0, false}, // within epsilon
		{1.0 + Epsilon*2, 1.0, true},
	}

	for _, tt := range tests {
		if got := FloatGreater(tt.a, tt.b); got != tt.expected {
			t.Errorf("FloatGreater(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.expected)
		}
	}
}

func TestIsZero(t *testing.T) {
	tests := []struct {
		v        float64
		expected bool
	}{
		{0, true},
		{Epsilon / 2, true},
		{-Epsilon / 2, true},
		{Epsilon * 2, false},
		{1.0, false},
		{-1.0, false},
	}

	for _, tt := range tests {
		if got := IsZero(tt.v); got != tt.expected {
			t.Errorf("IsZero(%v) = %v, want %v", tt.v, got, tt.expected)
		}
	}
}

func TestIsPositive(t *testing.T) {
	tests := []struct {
		v        float64
		expected bool
	}{
		{1.0, true},
		{Epsilon * 2, true},
		{Epsilon / 2, false},
		{0, false},
		{-1.0, false},
	}

	for _, tt := range tests {
		if got := IsPositive(tt.v); got != tt.expected {
			t.Errorf("IsPositive(%v) = %v, want %v", tt.v, got, tt.expected)
		}
	}
}

func TestMin(t *testing.T) {
	tests := []struct {
		a, b, expected float64
	}{
		{1.0, 2.0, 1.0},
		{2.0, 1.0, 1.0},
		{1.0, 1.0, 1.0},
		{-1.0, 1.0, -1.0},
		{math.Inf(1), 1.0, 1.0},
	}

	for _, tt := range tests {
		if got := Min(tt.a, tt.b); got != tt.expected {
			t.Errorf("Min(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.expected)
		}
	}
}

func TestMax(t *testing.T) {
	tests := []struct {
		a, b, expected float64
	}{
		{1.0, 2.0, 2.0},
		{2.0, 1.0, 2.0},
		{1.0, 1.0, 1.0},
		{-1.0, 1.0, 1.0},
		{math.Inf(-1), 1.0, 1.0},
	}

	for _, tt := range tests {
		if got := Max(tt.a, tt.b); got != tt.expected {
			t.Errorf("Max(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.expected)
		}
	}
}

func TestConstants(t *testing.T) {
	if Epsilon <= 0 {
		t.Error("Epsilon should be positive")
	}
	if Infinity != math.MaxFloat64 {
		t.Error("Infinity should equal MaxFloat64")
	}
	if NegativeInfinity != -math.MaxFloat64 {
		t.Error("NegativeInfinity should equal -MaxFloat64")
	}
	if SuperSourceID >= 0 {
		t.Error("SuperSourceID should be negative")
	}
	if SuperSinkID >= 0 {
		t.Error("SuperSinkID should be negative")
	}
}

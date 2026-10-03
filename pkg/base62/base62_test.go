package base62

import (
	"math"
	"testing"
)

func TestEncodeDecodeRoundtrip(t *testing.T) {
	testCases := []int64{
		0,
		1,
		61,
		62,
		63,
		999,
		1000000,
		123456789,
		9876543210,
		math.MaxInt32,
		1 << 40,
		math.MaxInt64 - 1,
	}

	for _, original := range testCases {
		encoded, err := Encode(original)
		if err != nil {
			t.Fatalf("unexpected error encoding %d: %v", original, err)
		}

		if !IsValid(encoded) {
			t.Errorf("encoded string %q for %d failed IsValid", encoded, original)
		}

		decoded, err := Decode(encoded)
		if err != nil {
			t.Fatalf("unexpected error decoding %q (original %d): %v", encoded, original, err)
		}

		if decoded != original {
			t.Errorf("roundtrip mismatch: expected %d, got %d (encoded: %s)", original, decoded, encoded)
		}
	}
}

func TestEncodeNegative(t *testing.T) {
	_, err := Encode(-1)
	if err == nil {
		t.Fatal("expected error when encoding negative number, got nil")
	}
}

func TestDecodeErrors(t *testing.T) {
	tests := []struct {
		input       string
		expectError bool
	}{
		{"", true},
		{"abc!", true},
		{"hello-world", true},
		{" ", true},
		{"valid123", false},
		{"0", false},
		{"Z", false},
	}

	for _, tc := range tests {
		_, err := Decode(tc.input)
		if tc.expectError && err == nil {
			t.Errorf("expected error decoding %q, got nil", tc.input)
		}
		if !tc.expectError && err != nil {
			t.Errorf("unexpected error decoding %q: %v", tc.input, err)
		}
	}
}

func TestIsValid(t *testing.T) {
	if IsValid("") {
		t.Error("expected empty string to be invalid")
	}
	if !IsValid("abcXYZ012") {
		t.Error("expected valid base62 string to return true")
	}
	if IsValid("abc@123") {
		t.Error("expected string with '@' to return false")
	}
}

func BenchmarkEncode(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = Encode(10000000 + int64(i))
	}
}

func BenchmarkDecode(b *testing.B) {
	s, _ := Encode(987654321)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Decode(s)
	}
}

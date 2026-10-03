package base62

import (
	"errors"
	"strings"
)

const (
	// Alphabet is the standard 62-character set [0-9a-zA-Z] used for encoding.
	Alphabet = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	Base     = int64(len(Alphabet))
)

var (
	// ErrInvalidBase62String is returned when an input string contains characters outside the alphabet.
	ErrInvalidBase62String = errors.New("invalid base62 character detected")
	// ErrEmptyString is returned when an empty string is provided to decode.
	ErrEmptyString = errors.New("cannot decode empty base62 string")
	// ErrNegativeNumber is returned when attempting to encode a negative integer.
	ErrNegativeNumber = errors.New("cannot encode negative number")
)

// lookupTable maps byte values to their index in Alphabet for O(1) character decoding.
var lookupTable [256]int8

func init() {
	for i := range lookupTable {
		lookupTable[i] = -1
	}
	for i := 0; i < len(Alphabet); i++ {
		lookupTable[Alphabet[i]] = int8(i)
	}
}

// Encode converts a non-negative 64-bit integer into a Base62 string.
func Encode(num int64) (string, error) {
	if num < 0 {
		return "", ErrNegativeNumber
	}
	if num == 0 {
		return string(Alphabet[0]), nil
	}

	var builder strings.Builder
	// In Base62, int64 max (9.22e18) fits in at most 11 characters.
	builder.Grow(11)

	// Temporary buffer for reversed digits
	var digits [11]byte
	idx := 0

	n := num
	for n > 0 {
		rem := n % Base
		digits[idx] = Alphabet[rem]
		idx++
		n /= Base
	}

	// Reverse the digits into the string builder
	for i := idx - 1; i >= 0; i-- {
		builder.WriteByte(digits[i])
	}

	return builder.String(), nil
}

// Decode parses a Base62 encoded string back to an int64 integer.
func Decode(s string) (int64, error) {
	if len(s) == 0 {
		return 0, ErrEmptyString
	}

	var result int64
	for i := 0; i < len(s); i++ {
		b := s[i]
		val := lookupTable[b]
		if val < 0 {
			return 0, ErrInvalidBase62String
		}

		// Overflow check before multiplication
		if result > (1<<63-1-int64(val))/Base {
			return 0, errors.New("base62 integer overflow")
		}

		result = result*Base + int64(val)
	}

	return result, nil
}

// IsValid validates whether the provided string consists solely of Base62 characters.
func IsValid(s string) bool {
	if len(s) == 0 {
		return false
	}
	for i := 0; i < len(s); i++ {
		if lookupTable[s[i]] < 0 {
			return false
		}
	}
	return true
}

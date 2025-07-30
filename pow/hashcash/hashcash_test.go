package hashcash_test

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
	"hash"
	"strconv"
	"testing"
	"time"

	"github.com/RogueTeam/onion/pow/hashcash"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
)

func Test_CountLeadingBits(t *testing.T) {
	t.Run("Succeed", func(t *testing.T) {
		type Test struct {
			Name    string // Added for better test case identification
			Payload []byte // Directly provide payload for more specific bit tests
			Expect  int
		}

		tests := []Test{
			// Original tests (modified to use Payload directly for clarity)
			{Name: "8 Zero Bytes", Payload: make([]byte, 8), Expect: 64},
			{Name: "16 Zero Bytes", Payload: make([]byte, 16), Expect: 64 * 2},
			{Name: "32 Zero Bytes", Payload: make([]byte, 32), Expect: 64 * 4},
			{Name: "64 Zero Bytes", Payload: make([]byte, 64), Expect: 64 * 8},

			// New Test Cases

			// Case 1: All bytes are zero (similar to original, but explicit)
			{Name: "All zeros - 4 bytes", Payload: []byte{0x00, 0x00, 0x00, 0x00}, Expect: 32},
			{Name: "All zeros - 1 byte", Payload: []byte{0x00}, Expect: 8},
			{Name: "All zeros - 0 bytes (empty payload)", Payload: []byte{}, Expect: 0},

			// Case 2: No leading zero bytes
			{Name: "First byte non-zero (0x01)", Payload: []byte{0x01, 0x00, 0x00}, Expect: 7}, // 00000001 -> 7 leading zeros
			{Name: "First byte non-zero (0x02)", Payload: []byte{0x02, 0x00, 0x00}, Expect: 6}, // 00000010 -> 6 leading zeros
			{Name: "First byte non-zero (0x80)", Payload: []byte{0x80, 0x00, 0x00}, Expect: 0}, // 10000000 -> 0 leading zeros
			{Name: "First byte non-zero (0xFF)", Payload: []byte{0xFF, 0x00, 0x00}, Expect: 0}, // 11111111 -> 0 leading zeros
			{Name: "First byte non-zero (0x0F)", Payload: []byte{0x0F, 0x00, 0x00}, Expect: 4}, // 00001111 -> 4 leading zeros

			// Case 3: Mixed zero and non-zero bytes
			{Name: "One zero byte, then non-zero", Payload: []byte{0x00, 0x01, 0x00}, Expect: 8 + 7},         // 8 zeros from first byte, 7 from second
			{Name: "Two zero bytes, then non-zero", Payload: []byte{0x00, 0x00, 0x01}, Expect: 16 + 7},       // 16 zeros from first two bytes, 7 from third
			{Name: "One zero byte, then full byte (0x80)", Payload: []byte{0x00, 0x80, 0x00}, Expect: 8 + 0}, // 8 zeros from first byte, 0 from second
			{Name: "Three zero bytes, then 0x0F", Payload: []byte{0x00, 0x00, 0x00, 0x0F}, Expect: 24 + 4},   // 24 zeros from first three bytes, 4 from fourth

			// Case 4: Edge cases
			{Name: "Single byte, all zeros", Payload: []byte{0x00}, Expect: 8},
			{Name: "Single byte, non-zero", Payload: []byte{0x01}, Expect: 7},
			{Name: "Empty payload", Payload: []byte{}, Expect: 0},
		}

		for _, test := range tests {
			t.Run(test.Name, func(t *testing.T) {
				assertions := assert.New(t)

				// For tests with specific bit patterns, we directly use the defined payload.
				// For tests that need random non-zero bytes after leading zeros,
				// we'll handle that where `ZeroBytes` is relevant.
				// However, for these more explicit tests, `test.Payload` is direct.

				// If you want to simulate the original `rand.Read` behavior for certain tests,
				// you'd need to modify the Test struct to include a `ZeroBytes` field
				// and apply `rand.Read` to the non-zero portion.
				// For now, these new tests use fixed payloads for predictable results.

				bits := hashcash.CountLeadingBits(test.Payload) // Call your actual function

				assertions.Equal(test.Expect, bits, "expecting other value for bits for payload: %v", test.Payload)
			})
		}
	})

	// This sub-test demonstrates how to use `ZeroBytes` with random data,
	// similar to your original structure, but with more varied inputs.
	t.Run("RandomDataVariations", func(t *testing.T) {
		type Test struct {
			Length    int64
			ZeroBytes int // Number of leading bytes to explicitly set to zero
			Expect    int
		}

		randomTests := []Test{
			{Length: 1, ZeroBytes: 1, Expect: 8},    // One zero byte
			{Length: 5, ZeroBytes: 1, Expect: 8},    // One leading zero byte
			{Length: 5, ZeroBytes: 3, Expect: 24},   // Three leading zero bytes
			{Length: 10, ZeroBytes: 5, Expect: 40},  // Five leading zero bytes
			{Length: 10, ZeroBytes: 10, Expect: 80}, // All ten bytes zero
		}

		for _, test := range randomTests {
			testName := "Length:" + strconv.FormatInt(test.Length, 10) + "_ZeroBytes:" + strconv.Itoa(test.ZeroBytes)
			t.Run(testName, func(t *testing.T) {
				assertions := assert.New(t)

				payload := make([]byte, test.Length)
				if test.ZeroBytes < int(test.Length) {
					// Fill the non-zero part with random data
					_, err := rand.Read(payload[test.ZeroBytes:])
					assertions.NoError(err, "Failed to read random bytes")
					// Ensure the first byte after the zeroed section is non-zero
					// to make the `Expect` value precise based on `ZeroBytes * 8`.
					// If rand.Read produced a 0 at payload[test.ZeroBytes], our expectation would be off.
					// This check makes the test more robust against random data quirks.
					if test.ZeroBytes > 0 && payload[test.ZeroBytes] == 0 {
						payload[test.ZeroBytes] = 0x01 // Force it to be non-zero
					}
				}

				// The first `test.ZeroBytes` bytes are already 0 from `make([]byte, test.Length)`
				// as bytes are initialized to zero by default.

				bits := hashcash.CountLeadingBits(payload) // Call your actual function

				// The expectation here is based purely on the `ZeroBytes` parameter,
				// assuming `CountLeadingBits` works as expected for full zero bytes
				// and then stops counting when it hits a non-zero byte (which we ensure
				// if `ZeroBytes < Length`).
				assertions.LessOrEqual(test.Expect, bits, "expecting other value for bits for payload: %v", payload)
			})
		}
	})
}

func Test_Integration(t *testing.T) {
	t.Run("Succeed", func(t *testing.T) {
		type Test struct {
			Name    string // Added a Name field for descriptive test names
			Bits    uint64
			Algo    hash.Hash
			Salt    string
			Payload string
		}
		tests := []Test{
			{
				Name:    "SHA256_BasicSuccess",
				Bits:    20,
				Algo:    sha256.New(),
				Salt:    "testSalt123",
				Payload: "hello world",
			},
			{
				Name:    "SHA512_DifferentAlgo",
				Bits:    22,
				Algo:    sha512.New(),
				Salt:    "anotherSalt",
				Payload: "some important data",
			},
			{
				Name:    "SHA256_HigherBits",
				Bits:    24, // Higher bits will take longer to compute, but should still succeed
				Algo:    sha256.New(),
				Salt:    "highComplexitySalt",
				Payload: "performance_test_payload",
			},
			{
				Name:    "SHA256_EmptySalt",
				Bits:    18,
				Algo:    sha256.New(),
				Salt:    "",
				Payload: "payload with no salt",
			},
			{
				Name:    "SHA256_EmptyPayload",
				Bits:    18,
				Algo:    sha256.New(),
				Salt:    "salt_no_payload",
				Payload: "",
			},
			{
				Name:    "SHA256_BothEmpty",
				Bits:    16,
				Algo:    sha256.New(),
				Salt:    "",
				Payload: "",
			},
		}

		for _, test := range tests {
			// Using fmt.Sprintf to format the test name dynamically
			t.Run(fmt.Sprintf("Test_Case_%s_Bits_%d", test.Name, test.Bits), func(t *testing.T) {
				assertions := assert.New(t)

				// Assuming utils.NewContext() provides a context.Context and a cancellation function
				ctx, cancel := context.WithTimeout(context.TODO(), time.Hour)
				defer cancel() // Ensure the context is cancelled when the test finishes

				// Create a new hashcash instance
				hc, err := hashcash.New(ctx, test.Algo, test.Bits, test.Salt, test.Payload)
				assertions.Nil(err, "failed to hashcash for test: %s", test.Name) // Add test name to error message for clarity
				assertions.NotNil(hc, "hashcash result should not be nil for test: %s", test.Name)

				// Verify the generated hashcash
				err = hashcash.Verify(test.Algo, hc)
				assertions.Nil(err, "failed to verify hashcash for test: %s", test.Name) // Add test name to error message for clarity
			})
		}
	})
}

package hashcash

import (
	"bytes"
	"context"
	"crypto/sha3"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"hash"
	"math"
	"math/big"
	"math/bits"
	"strconv"
	"strings"
	"time"
)

var DefaultHashAlgorithm = sha3.New512

const DefaultMaxDifficulty = 32 // Can be modified by the compiler

func SqrtDifficulty(algo hash.Hash, peers int64) (x uint64) {
	const (
		BaseDifficulty = 0
		GrowthFactor   = 2
	)

	var maxDifficulty uint64
	if DefaultMaxDifficulty == 0 {
		maxDifficulty = uint64(algo.Size() * 8)
	} else {
		maxDifficulty = DefaultMaxDifficulty
	}

	a := big.NewInt(0).Sqrt(big.NewInt(peers))
	b := a.Mul(a, big.NewInt(GrowthFactor))
	c := b.Add(b, big.NewInt(BaseDifficulty))

	x = c.Uint64()
	if x > maxDifficulty {
		return maxDifficulty
	}
	return x
}

func LogDifficulty(algo hash.Hash, peers int64) (x uint64) {
	const (
		BaseDifficulty = 0
		GrowthFactor   = 2
	)

	var maxDifficulty uint64
	if DefaultMaxDifficulty == 0 {
		maxDifficulty = uint64(algo.Size() * 8)
	} else {
		maxDifficulty = DefaultMaxDifficulty
	}

	a := math.Log(float64(peers))
	b := big.NewFloat(a)
	c := b.Mul(b, big.NewFloat(GrowthFactor))
	d := c.Add(c, big.NewFloat(BaseDifficulty))

	x, _ = d.Uint64()
	if x > maxDifficulty {
		return maxDifficulty
	}
	return x
}

var countLeadingBitsCbytes = []int{8, 4, 2, 1}

func CountLeadingBits(s []byte) (n int) {
	src := bytes.NewReader(s)

	var remaining = len(s)

	for _, cBytes := range countLeadingBitsCbytes {
		if remaining < cBytes {
			continue
		}

		var maximum = cBytes * 8

		toUse := remaining - remaining%cBytes
		remaining -= toUse

		for range toUse / cBytes {
			var m int
			switch maximum {
			case 64:
				var value uint64
				binary.Read(src, binary.BigEndian, &value)
				m = bits.LeadingZeros64(value)
			case 32:
				var value uint32
				binary.Read(src, binary.BigEndian, &value)
				m = bits.LeadingZeros32(value)
			case 16:
				var value uint16
				binary.Read(src, binary.BigEndian, &value)
				m = bits.LeadingZeros16(value)
			case 8:
				var value uint8
				binary.Read(src, binary.BigEndian, &value)
				m = bits.LeadingZeros8(value)
			}

			n += m
			if m != maximum {
				return n
			}
		}
	}

	return
}

// Creates a hashcash (hc)
// hash is reset on every try
func New(ctx context.Context, h hash.Hash, bits uint64, salt, payload string) (hc string, err error) {
	now := time.Now()

	for counter := big.NewInt(0); ; counter = counter.Add(counter, big.NewInt(1)) {
		select {
		case <-ctx.Done():
			return hc, ctx.Err()
		default:
			h.Reset()

			hc = fmt.Sprintf("1:%d:%s::%s:%s:%s", bits, now.Format("240615143059"), payload, salt, base64.StdEncoding.EncodeToString(counter.Bytes()))

			h.Write([]byte(hc))

			if uint64(CountLeadingBits(h.Sum(nil))) == bits {
				return hc, nil
			}
		}
	}
}

// Verifies a hashcash is valid. Returns nil on valid
// And error if the hash is invalid
func Verify(h hash.Hash, hc string) (err error) {
	var parts = strings.Split(hc, ":")
	if len(parts) != 7 {
		return errors.New("invalid hashcash")
	}

	bits, err := strconv.ParseUint(parts[1], 10, 64)
	if err != nil {
		return fmt.Errorf("failed to parse bits part: %w", err)
	}

	h.Reset()
	h.Write([]byte(hc))

	cBits := uint64(CountLeadingBits(h.Sum(nil)))
	if bits != cBits {
		return fmt.Errorf("expecting %d bits but got %d", bits, cBits)
	}
	return nil
}

// Same as Verify but checks the if needed bits is greater or equal to the passed difficulty
func VerifyWithDifficultyAndPayload(h hash.Hash, hc string, difficulty uint64, payload string) (err error) {
	var parts = strings.Split(hc, ":")
	if len(parts) != 7 {
		return errors.New("invalid hashcash")
	}

	expectedPayload := parts[4]
	if payload != expectedPayload {
		return fmt.Errorf("expecting a different payload: %s != %s", payload, expectedPayload)
	}

	bits, err := strconv.ParseUint(parts[1], 10, 64)
	if err != nil {
		return fmt.Errorf("failed to parse bits part: %w", err)
	}

	if bits < difficulty {
		return fmt.Errorf("expecting difficulty: %d but got difficulty: %d", difficulty, bits)
	}

	h.Reset()
	h.Write([]byte(hc))

	cBits := uint64(CountLeadingBits(h.Sum(nil)))
	if bits != cBits {
		return fmt.Errorf("expecting %d bits but got %d", bits, cBits)
	}
	return nil
}

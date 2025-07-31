package compressedtunnel_test

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/RogueTeam/onion/net/compressedtunnel"
	"github.com/stretchr/testify/assert"
)

func Test_Tunnel(t *testing.T) {
	t.Run("Succeed", func(t *testing.T) {
		type Test struct {
			Length int
		}
		tests := []Test{
			{Length: 10},
			{Length: 10240},
			{Length: 1024},
			{Length: 1025},
			{Length: 2048},
			{Length: 1},
			{Length: 124},
			{Length: 8},
		}
		for _, test := range tests {
			t.Run(fmt.Sprintf("Length %d", test.Length), func(t *testing.T) {
				assertions := assert.New(t)

				payload := make([]byte, test.Length)
				raw := bytes.NewReader(payload)
				compressed := bytes.NewBuffer(nil)
				compressed.Grow(test.Length)

				const chunkSize = 1024
				for {
					n, err := compressedtunnel.PipeFromRaw(compressed, raw, chunkSize)
					if err != nil {
						if errors.Is(err, io.EOF) {
							break
						}
						assertions.Nil(err, "failed to pipe from raw")
					}
					t.Log("Compressed", n)
				}

				result := bytes.NewBuffer(nil)
				result.Grow(test.Length)

				compressedReader := bytes.NewReader(compressed.Bytes())
				for result.Len() < len(payload) {
					n, err := compressedtunnel.PipeFromCompressed(result, compressedReader)
					assertions.Nil(err, "failed to obtain result")
					t.Log("Raw", n)

				}

				assertions.True(bytes.Equal(result.Bytes(), payload), "received other stuff")
			})
		}
	})
}

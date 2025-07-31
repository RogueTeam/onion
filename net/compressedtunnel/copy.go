package compressedtunnel

import (
	"fmt"
	"io"
)

func PipeFromRaw(compressedDst io.Writer, rawSrc io.Reader, size int) (n int, err error) {
	var chunk = make([]byte, size)

	n, err = rawSrc.Read(chunk)
	if err != nil {
		return 0, fmt.Errorf("failed to read from source: %w", err)
	}

	err = Send(compressedDst, chunk[:n])
	if err != nil {
		return 0, fmt.Errorf("failed to write to destination: %w", err)
	}
	return n, nil
}

func PipeFromCompressed(rawDst io.Writer, compressedSrc io.Reader) (n int, err error) {
	var msg Msg

	err = msg.Recv(compressedSrc)
	if err != nil {
		return 0, fmt.Errorf("failed to receive from source: %w", err)
	}

	n, err = rawDst.Write(msg.Data)
	if err != nil {
		return 0, fmt.Errorf("failed to write to destination: %w", err)
	}
	return n, nil
}

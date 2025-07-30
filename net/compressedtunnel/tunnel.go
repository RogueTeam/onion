package compressedtunnel

import (
	"fmt"
	"io"
)

type Compressed struct {
	rw           io.ReadWriteCloser
	pendingSlice []byte
}

func New(rw io.ReadWriteCloser) (c *Compressed) {
	return &Compressed{rw: rw}
}

func (c *Compressed) Close() (err error) {
	return c.rw.Close()
}

func (c *Compressed) Write(b []byte) (n int, err error) {
	err = Send(c.rw, b)
	if err != nil {
		return 0, fmt.Errorf("failed to send msgs: %w", err)
	}
	return len(b), nil
}

func (c *Compressed) Read(b []byte) (n int, err error) {
	var delta int
	var msg Msg
	for {
		if len(c.pendingSlice) == 0 {
			err = msg.Recv(c.rw)
			if err != nil {
				return n, fmt.Errorf("failed to recv msg: %w", err)
			}
			c.pendingSlice = msg.Data
		}

		copied := copy(b[delta:], c.pendingSlice)
		if copied == 0 {
			return n, nil
		}
		n += copied
		c.pendingSlice = c.pendingSlice[copied:]
		delta += copied
	}
}

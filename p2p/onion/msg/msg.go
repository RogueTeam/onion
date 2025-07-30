package msg

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"

	"github.com/RogueTeam/onion/utils"
	"github.com/klauspost/compress/gzip"
)

var buffersPool = utils.NewPool[bytes.Buffer]()

type Compression uint8

const (
	CompressionNode Compression = iota
	CompressionGzip
)

func (c Compression) String() (s string) {
	switch c {
	case CompressionNode:
		return "none"
	case CompressionGzip:
		return "gzip"
	default:
		return "<unknown>"
	}
}

type Msg struct {
	Compression Compression
	Length      uint64
	Data        []byte
}

func (m *Msg) String() (s string) {
	return fmt.Sprintf("%s:%d", m.Compression.String(), m.Length)
}

func (m *Msg) Recv(r io.Reader) (err error) {
	err = binary.Read(r, binary.BigEndian, &m.Compression)
	if err != nil {
		return fmt.Errorf("failed to read compression level: %w", err)
	}

	err = binary.Read(r, binary.BigEndian, &m.Length)
	if err != nil {
		return fmt.Errorf("failed to read length: %w", err)
	}

	rawData := make([]byte, m.Length)
	n, err := r.Read(rawData)
	if err != nil {
		return fmt.Errorf("failed to read raw data: %w", err)
	}

	if uint64(n) != m.Length {
		return fmt.Errorf("wrong length: expecting = %d but got %d", m.Length, n)
	}
	switch m.Compression {
	case CompressionNode:
		m.Data = rawData
	case CompressionGzip:
		compressR, err := gzip.NewReader(bytes.NewBuffer(rawData))
		if err != nil {
			return fmt.Errorf("failed to prepare gzip reader: %w", err)
		}

		m.Data, err = io.ReadAll(compressR)
		if err != nil {
			return fmt.Errorf("failed to read compressed data: %w", err)
		}
	}
	return nil
}

func Send(w io.Writer, data []byte) (err error) {
	buf := buffersPool.Get()
	defer buffersPool.Put(buf)
	buf.Reset()

	// Compresss
	compressW, err := gzip.NewWriterLevel(buf, gzip.BestCompression)
	if err != nil {
		return fmt.Errorf("failed to prepare gzip writer: %w", err)
	}
	_, err = compressW.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write data to gzip: %w", err)
	}

	err = compressW.Flush()
	if err != nil {
		return fmt.Errorf("failed to flush gzip: %w", err)
	}

	err = compressW.Close()
	if err != nil {
		return fmt.Errorf("failed to close gzip: %w", err)
	}

	// Send msg
	var msg Msg
	compressed := buf.Bytes()
	if len(compressed) < len(data) {
		msg.Compression = CompressionGzip
		msg.Length = uint64(buf.Len())
		msg.Data = buf.Bytes()
	} else {
		msg.Compression = CompressionNode
		msg.Length = uint64(len(data))
		msg.Data = data
	}

	bw := bufio.NewWriter(w)
	err = binary.Write(bw, binary.BigEndian, msg.Compression)
	if err != nil {
		return fmt.Errorf("failed to write compression: %w", err)
	}

	err = binary.Write(bw, binary.BigEndian, msg.Length)
	if err != nil {
		return fmt.Errorf("failed to write length: %w", err)
	}

	log.Println("Sending", msg.String())
	_, err = bw.Write(msg.Data)
	if err != nil {
		return fmt.Errorf("failed to write data: %w", err)
	}

	err = bw.Flush()
	if err != nil {
		return fmt.Errorf("failed to flush writer: %w", err)
	}
	return nil
}

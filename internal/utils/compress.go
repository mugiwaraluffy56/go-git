package utils

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
)

// Compress compresses data using zlib
func Compress(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	if _, err := w.Write(data); err != nil {
		return nil, fmt.Errorf("failed to compress: %w", err)
	}
	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("failed to close compressor: %w", err)
	}
	return buf.Bytes(), nil
}

// Decompress decompresses zlib-compressed data
func Decompress(data []byte) ([]byte, error) {
	r, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create decompressor: %w", err)
	}
	defer r.Close()

	result, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to decompress: %w", err)
	}
	return result, nil
}

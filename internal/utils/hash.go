package utils

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
)

// HashObject computes the SHA-1 hash of an object in Git format
// Format: "<type> <size>\0<content>"
func HashObject(objType string, data []byte) string {
	header := fmt.Sprintf("%s %d\x00", objType, len(data))
	store := append([]byte(header), data...)
	hash := sha1.Sum(store)
	return hex.EncodeToString(hash[:])
}

// HashBytes computes SHA-1 hash of raw bytes
func HashBytes(data []byte) string {
	hash := sha1.Sum(data)
	return hex.EncodeToString(hash[:])
}

// HashBytesRaw returns the raw 20-byte SHA-1 hash
func HashBytesRaw(data []byte) [20]byte {
	return sha1.Sum(data)
}

// HexToBytes converts a hex string to bytes
func HexToBytes(hexStr string) ([]byte, error) {
	return hex.DecodeString(hexStr)
}

// BytesToHex converts bytes to hex string
func BytesToHex(data []byte) string {
	return hex.EncodeToString(data)
}

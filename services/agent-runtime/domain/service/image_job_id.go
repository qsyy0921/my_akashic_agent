package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

func NewImageJobID(requestID string) string {
	requestID = strings.TrimSpace(requestID)
	if requestID != "" {
		sum := sha256.Sum256([]byte(requestID))
		return "img_" + hex.EncodeToString(sum[:8])
	}

	var data [16]byte
	if _, err := rand.Read(data[:]); err != nil {
		sum := sha256.Sum256([]byte(NewNonce()))
		return "img_" + hex.EncodeToString(sum[:8])
	}
	return "img_" + hex.EncodeToString(data[:])
}


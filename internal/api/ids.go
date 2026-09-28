package api

import (
	"crypto/rand"
	"encoding/hex"
)

// newFileID generates a random, URL-safe, collision-resistant identifier.
// A dependency-free crypto/rand hex string is enough entropy for this
// project's scale and avoids pulling in a UUID library.
func newFileID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

package common //nolint: revive //fine

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

const HashHeader = "HashSHA256"

func Hash(data []byte, key []byte) (string, error) {
	var bufH bytes.Buffer
	enc := base64.NewEncoder(base64.StdEncoding, &bufH)
	hash := hmac.New(sha256.New, key)
	hash.Write(data)
	if _, err := enc.Write(hash.Sum(nil)); err != nil {
		return "", fmt.Errorf("writing to encoder: %w", err)
	}
	if err := enc.Close(); err != nil {
		return "", fmt.Errorf("closing encoder: %w", err)
	}
	return bufH.String(), nil
}

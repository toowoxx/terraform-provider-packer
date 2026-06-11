package crypto_util

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
)

// FileSHA256 hashes only the file's content, suitable for comparing
// against externally published checksums.
func FileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func FilesSHA256(paths ...string) (string, error) {
	h := sha256.New()

	for _, path := range paths {
		h.Write([]byte(path))

		f, err := os.Open(path)
		if err != nil {
			_ = f.Close()
			return "", err
		}

		_, err = io.Copy(h, f)
		if err != nil {
			_ = f.Close()
			return "", err
		}
		_ = f.Close()
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

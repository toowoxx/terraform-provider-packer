package crypto_util

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
)

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

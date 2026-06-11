package provider

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"terraform-provider-packer/crypto_util"
)

const downloadCacheSubdir = "terraform-provider-packer/downloaded-binaries"

var zipMagic = []byte{'P', 'K', 0x03, 0x04}

func normalizeChecksum(checksum string) (string, error) {
	c := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(checksum)), "sha256:")
	if len(c) != 64 {
		return "", fmt.Errorf("expected a 64-character hexadecimal SHA-256 checksum, got %d characters", len(c))
	}
	if _, err := hex.DecodeString(c); err != nil {
		return "", fmt.Errorf("checksum is not valid hexadecimal: %v", err)
	}
	return c, nil
}

func downloadedBinaryName() string {
	if runtime.GOOS == "windows" {
		return "packer.exe"
	}
	return "packer"
}

// downloadCacheKey derives a stable directory name from URL and checksum so
// that a cached binary is only reused for the exact same source and
// verification requirements it was originally downloaded with.
func downloadCacheKey(rawURL string, checksum string) string {
	sum := sha256.Sum256([]byte(rawURL + "\n" + checksum))
	return hex.EncodeToString(sum[:])
}

var downloadCacheBaseDir = func() string {
	base, err := os.UserCacheDir()
	if err != nil {
		base = os.TempDir()
	}
	return filepath.Join(base, filepath.FromSlash(downloadCacheSubdir))
}

// ensureDownloadedPackerBinary returns the local path of the binary served by
// rawURL, downloading and caching it if necessary. The artifact may be a raw
// executable or a zip archive containing one. checksum, when non-empty, is a
// SHA-256 hash that the downloaded artifact itself must match.
func ensureDownloadedPackerBinary(ctx context.Context, rawURL string, checksum string) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("invalid URL %q: %v", rawURL, err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("unsupported URL scheme %q: only http and https are supported", parsed.Scheme)
	}

	if checksum != "" {
		if checksum, err = normalizeChecksum(checksum); err != nil {
			return "", err
		}
	}

	targetDir := filepath.Join(downloadCacheBaseDir(), downloadCacheKey(rawURL, checksum))
	target := filepath.Join(targetDir, downloadedBinaryName())
	if _, statErr := os.Stat(target); statErr == nil {
		return target, nil
	}

	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return "", fmt.Errorf("could not create cache directory %q: %v", targetDir, err)
	}

	artifact, err := downloadToTempFile(ctx, rawURL, targetDir)
	if err != nil {
		return "", err
	}
	defer func() { _ = os.Remove(artifact) }()

	if checksum != "" {
		actual, hashErr := crypto_util.FileSHA256(artifact)
		if hashErr != nil {
			return "", fmt.Errorf("could not hash downloaded file: %v", hashErr)
		}
		if actual != checksum {
			return "", fmt.Errorf(
				"checksum mismatch for %s: expected sha256:%s, got sha256:%s",
				rawURL, checksum, actual,
			)
		}
	}

	binary := artifact
	if isZip, zipErr := fileHasZipMagic(artifact); zipErr != nil {
		return "", zipErr
	} else if isZip {
		binary, err = extractPackerFromZip(artifact, targetDir)
		if err != nil {
			return "", err
		}
		defer func() { _ = os.Remove(binary) }()
	}

	if err := os.Chmod(binary, 0o755); err != nil {
		return "", fmt.Errorf("could not make downloaded binary executable: %v", err)
	}
	if err := os.Rename(binary, target); err != nil {
		// A concurrent run may have populated the cache entry first; that
		// copy passed the same verification, so use it.
		if _, statErr := os.Stat(target); statErr == nil {
			return target, nil
		}
		return "", fmt.Errorf("could not move downloaded binary into cache: %v", err)
	}
	return target, nil
}

func downloadToTempFile(ctx context.Context, rawURL string, dir string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", fmt.Errorf("could not create request for %s: %v", rawURL, err)
	}
	req.Header.Set("User-Agent", "terraform-provider-packer")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("could not download %s: %v", rawURL, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("could not download %s: unexpected status %s", rawURL, resp.Status)
	}

	tmp, err := os.CreateTemp(dir, "download-*")
	if err != nil {
		return "", fmt.Errorf("could not create temporary file in %q: %v", dir, err)
	}
	if _, err := io.Copy(tmp, resp.Body); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
		return "", fmt.Errorf("could not write download to %q: %v", tmp.Name(), err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmp.Name())
		return "", fmt.Errorf("could not finish writing download to %q: %v", tmp.Name(), err)
	}
	return tmp.Name(), nil
}

func fileHasZipMagic(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, fmt.Errorf("could not open downloaded file: %v", err)
	}
	defer func() { _ = f.Close() }()

	header := make([]byte, len(zipMagic))
	if _, err := io.ReadFull(f, header); err != nil {
		// Too short to be a zip archive; treat as a raw binary.
		return false, nil
	}
	return bytes.Equal(header, zipMagic), nil
}

// extractPackerFromZip extracts the Packer executable from the archive into
// dir and returns its temporary path. The archive must contain a file named
// packer (or packer.exe), or exactly one file.
func extractPackerFromZip(archivePath string, dir string) (string, error) {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return "", fmt.Errorf("could not open downloaded zip archive: %v", err)
	}
	defer func() { _ = r.Close() }()

	var files []*zip.File
	var named []*zip.File
	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}
		files = append(files, f)
		base := strings.ToLower(filepath.Base(filepath.FromSlash(f.Name)))
		if base == "packer" || base == "packer.exe" {
			named = append(named, f)
		}
	}

	var chosen *zip.File
	switch {
	case len(named) == 1:
		chosen = named[0]
	case len(named) > 1:
		return "", fmt.Errorf("downloaded zip archive contains multiple files named packer")
	case len(files) == 1:
		chosen = files[0]
	default:
		return "", fmt.Errorf(
			"could not identify a binary in the downloaded zip archive: " +
				"expected a file named packer (or packer.exe) or an archive containing exactly one file",
		)
	}

	src, err := chosen.Open()
	if err != nil {
		return "", fmt.Errorf("could not read %q from downloaded zip archive: %v", chosen.Name, err)
	}
	defer func() { _ = src.Close() }()

	tmp, err := os.CreateTemp(dir, "extract-*")
	if err != nil {
		return "", fmt.Errorf("could not create temporary file in %q: %v", dir, err)
	}
	if _, err := io.Copy(tmp, src); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
		return "", fmt.Errorf("could not extract %q from downloaded zip archive: %v", chosen.Name, err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmp.Name())
		return "", fmt.Errorf("could not finish extracting %q: %v", chosen.Name, err)
	}
	return tmp.Name(), nil
}

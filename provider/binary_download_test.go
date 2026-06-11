package provider

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
)

func useTempCacheDir(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	previous := downloadCacheBaseDir
	downloadCacheBaseDir = func() string { return dir }
	t.Cleanup(func() { downloadCacheBaseDir = previous })
}

func serveArtifact(t *testing.T, content []byte) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		_, _ = w.Write(content)
	}))
	t.Cleanup(server.Close)
	return server, &requests
}

func zipArchive(t *testing.T, files map[string][]byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for name, content := range files {
		f, err := w.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.Write(content); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func sha256Hex(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

func TestNormalizeChecksum(t *testing.T) {
	valid := strings.Repeat("ab", 32)
	for _, input := range []string{valid, "sha256:" + valid, "  SHA256:" + strings.ToUpper(valid) + "  "} {
		got, err := normalizeChecksum(input)
		if err != nil {
			t.Errorf("normalizeChecksum(%q) returned error: %v", input, err)
		} else if got != valid {
			t.Errorf("normalizeChecksum(%q) = %q, want %q", input, got, valid)
		}
	}
	for _, input := range []string{"", "abc", strings.Repeat("g", 64), "md5:" + valid} {
		if _, err := normalizeChecksum(input); err == nil {
			t.Errorf("normalizeChecksum(%q) should have failed", input)
		}
	}
}

func TestDownloadRawBinaryAndCache(t *testing.T) {
	useTempCacheDir(t)
	content := []byte("#!/bin/sh\necho fake packer\n")
	server, requests := serveArtifact(t, content)

	path, err := ensureDownloadedPackerBinary(context.Background(), server.URL+"/packer", "")
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, content) {
		t.Error("downloaded binary content does not match served content")
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm()&0o111 == 0 {
			t.Error("downloaded binary is not executable")
		}
	}

	cached, err := ensureDownloadedPackerBinary(context.Background(), server.URL+"/packer", "")
	if err != nil {
		t.Fatal(err)
	}
	if cached != path {
		t.Errorf("second call returned %q, want cached %q", cached, path)
	}
	if requests.Load() != 1 {
		t.Errorf("expected exactly 1 download request, got %d", requests.Load())
	}
}

func TestDownloadChecksumVerification(t *testing.T) {
	useTempCacheDir(t)
	content := []byte("fake packer binary")
	server, _ := serveArtifact(t, content)

	if _, err := ensureDownloadedPackerBinary(context.Background(), server.URL, strings.Repeat("00", 32)); err == nil {
		t.Error("expected checksum mismatch error")
	} else if !strings.Contains(err.Error(), "checksum mismatch") {
		t.Errorf("unexpected error: %v", err)
	}

	if _, err := ensureDownloadedPackerBinary(context.Background(), server.URL, "sha256:"+sha256Hex(content)); err != nil {
		t.Errorf("download with correct checksum failed: %v", err)
	}
}

func TestDownloadZipWithNamedBinary(t *testing.T) {
	useTempCacheDir(t)
	binary := []byte("fake packer binary from zip")
	archive := zipArchive(t, map[string][]byte{
		"some-dir/packer": binary,
		"LICENSE.txt":     []byte("license text"),
	})
	server, _ := serveArtifact(t, archive)

	path, err := ensureDownloadedPackerBinary(context.Background(), server.URL, sha256Hex(archive))
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, binary) {
		t.Error("extracted binary content does not match archived content")
	}
}

func TestDownloadZipWithSingleFile(t *testing.T) {
	useTempCacheDir(t)
	binary := []byte("fork binary with custom name")
	archive := zipArchive(t, map[string][]byte{"custom-packer-fork": binary})
	server, _ := serveArtifact(t, archive)

	path, err := ensureDownloadedPackerBinary(context.Background(), server.URL, "")
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, binary) {
		t.Error("extracted binary content does not match archived content")
	}
}

func TestDownloadZipWithoutIdentifiableBinary(t *testing.T) {
	useTempCacheDir(t)
	archive := zipArchive(t, map[string][]byte{
		"first":  []byte("a"),
		"second": []byte("b"),
	})
	server, _ := serveArtifact(t, archive)

	if _, err := ensureDownloadedPackerBinary(context.Background(), server.URL, ""); err == nil {
		t.Error("expected error for archive without identifiable binary")
	}
}

func TestDownloadRejectsUnsupportedScheme(t *testing.T) {
	useTempCacheDir(t)
	for _, rawURL := range []string{"file:///usr/bin/packer", "ftp://example.com/packer"} {
		if _, err := ensureDownloadedPackerBinary(context.Background(), rawURL, ""); err == nil {
			t.Errorf("expected error for URL %q", rawURL)
		}
	}
}

func TestDownloadFailsOnHTTPError(t *testing.T) {
	useTempCacheDir(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	t.Cleanup(server.Close)

	if _, err := ensureDownloadedPackerBinary(context.Background(), server.URL, ""); err == nil {
		t.Error("expected error for HTTP 404 response")
	}
}

func TestChecksumChangeBustsCache(t *testing.T) {
	useTempCacheDir(t)
	content := []byte("fake packer binary")
	server, requests := serveArtifact(t, content)
	checksum := sha256Hex(content)

	if _, err := ensureDownloadedPackerBinary(context.Background(), server.URL, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := ensureDownloadedPackerBinary(context.Background(), server.URL, checksum); err != nil {
		t.Fatal(err)
	}
	if requests.Load() != 2 {
		t.Errorf("expected a fresh download after adding a checksum, got %d requests", requests.Load())
	}
}

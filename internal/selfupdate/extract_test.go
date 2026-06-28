package selfupdate

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func tarGz(t *testing.T, name string, data []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: int64(len(data))}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func zipArchive(t *testing.T, name string, data []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create(name)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestExtractBinaryTarGz(t *testing.T) {
	want := []byte("fake-harness-binary")
	archive := tarGz(t, "harness", want)
	got, err := extractBinary(archive, "harness_linux_amd64.tar.gz", "harness")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("extracted %q, want %q", got, want)
	}
}

func TestExtractBinaryZip(t *testing.T) {
	want := []byte("fake-harness.exe")
	archive := zipArchive(t, "harness.exe", want)
	got, err := extractBinary(archive, "harness_windows_amd64.zip", "harness.exe")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("extracted %q, want %q", got, want)
	}
}

func TestExtractBinaryMissing(t *testing.T) {
	archive := tarGz(t, "README.md", []byte("nope"))
	if _, err := extractBinary(archive, "harness_linux_amd64.tar.gz", "harness"); err == nil {
		t.Fatal("expected error when the binary is absent")
	}
}

func TestVerifyChecksum(t *testing.T) {
	data := []byte("archive-bytes")
	sum := sha256.Sum256(data)
	hexsum := hex.EncodeToString(sum[:])
	checksums := []byte("deadbeef  other_asset.tar.gz\n" + hexsum + "  harness_linux_amd64.tar.gz\n")

	if err := verifyChecksum("harness_linux_amd64.tar.gz", data, checksums); err != nil {
		t.Fatalf("valid checksum rejected: %v", err)
	}
	if err := verifyChecksum("harness_linux_amd64.tar.gz", []byte("tampered"), checksums); err == nil {
		t.Fatal("expected mismatch error for tampered data")
	}
	if err := verifyChecksum("missing.tar.gz", data, checksums); err == nil {
		t.Fatal("expected error when asset is not listed")
	}
}

package selfupdate

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"path"
	"strings"
)

// maxBinarySize bounds how much we read out of an archive, guarding against a
// decompression bomb. harness binaries are a few MB; 200 MB is ample headroom.
const maxBinarySize = 200 << 20

// verifyChecksum confirms that data's SHA-256 matches the entry for assetName in
// a GoReleaser checksums.txt ("<hex>  <name>" lines).
func verifyChecksum(assetName string, data, checksums []byte) error {
	sum := sha256.Sum256(data)
	got := hex.EncodeToString(sum[:])
	for _, line := range strings.Split(string(checksums), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[1] == assetName {
			if !strings.EqualFold(fields[0], got) {
				return fmt.Errorf("checksum mismatch for %s", assetName)
			}
			return nil
		}
	}
	return fmt.Errorf("no checksum listed for %s", assetName)
}

// extractBinary returns the bytes of binaryName from a release archive. The
// archive is a .tar.gz, or a .zip when assetName ends in .zip (Windows).
func extractBinary(archive []byte, assetName, binaryName string) ([]byte, error) {
	if strings.HasSuffix(assetName, ".zip") {
		return extractFromZip(archive, binaryName)
	}
	return extractFromTarGz(archive, binaryName)
}

func extractFromTarGz(archive []byte, binaryName string) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return nil, fmt.Errorf("open gzip: %w", err)
	}
	defer func() { _ = gz.Close() }()

	reader := tar.NewReader(gz)
	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read tar: %w", err)
		}
		if path.Base(header.Name) == binaryName {
			return io.ReadAll(io.LimitReader(reader, maxBinarySize))
		}
	}
	return nil, fmt.Errorf("%s not found in archive", binaryName)
}

func extractFromZip(archive []byte, binaryName string) ([]byte, error) {
	reader, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		return nil, fmt.Errorf("open zip: %w", err)
	}
	for _, file := range reader.File {
		if path.Base(file.Name) == binaryName {
			rc, err := file.Open()
			if err != nil {
				return nil, err
			}
			defer func() { _ = rc.Close() }()
			return io.ReadAll(io.LimitReader(rc, maxBinarySize))
		}
	}
	return nil, fmt.Errorf("%s not found in archive", binaryName)
}

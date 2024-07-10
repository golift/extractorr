package xtractr

import (
	"archive/zip"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

/* How to extract a ZIP file. */

// ExtractZIP extracts a zip file.. to a destination. Simple enough.
func ExtractZIP(xFile *XFile) (int64, []string, error) {
	zipReader, err := zip.OpenReader(xFile.FilePath)
	if err != nil {
		return 0, nil, fmt.Errorf("zip.OpenReader: %w", err)
	}
	defer zipReader.Close()

	files := []string{}
	size := int64(0)

	for _, zipFile := range zipReader.Reader.File {
		fSize, wfile, err := xFile.unzip(zipFile)
		if err != nil {
			return size, files, fmt.Errorf("%s: %w", xFile.FilePath, err)
		}

		files = append(files, filepath.Join(xFile.OutputDir, zipFile.Name)) //nolint: gosec
		size += fSize
		xFile.Debugf("Wrote archived file: %s (%d bytes), total: %d files and %d bytes", wfile, fSize, len(files), size)
	}

	return size, files, nil
}

func (x *XFile) unzip(zipFile *zip.File) (int64, string, error) {
	wfile := x.clean(zipFile.Name)
	if !strings.HasPrefix(wfile, x.OutputDir) {
		// The file being written is trying to write outside of our base path. Malicious archive?
		return 0, wfile, fmt.Errorf("%s: %w: %s (from: %s)", zipFile.FileInfo().Name(), ErrInvalidPath, wfile, zipFile.Name)
	}

	if zipFile.FileInfo().IsDir() {
		x.Debugf("Writing archived directory: %s", wfile)

		if err := os.MkdirAll(wfile, x.DirMode); err != nil {
			return 0, wfile, fmt.Errorf("making zipFile dir: %w", err)
		}

		return 0, wfile, nil
	}

	x.Debugf("Writing archived file: %s (packed: %d, unpacked: %d)", wfile,
		zipFile.CompressedSize64, zipFile.UncompressedSize64)

	zFile, err := zipFile.Open()
	if err != nil {
		return 0, wfile, fmt.Errorf("zipFile.Open: %w", err)
	}
	defer zFile.Close()

	s, err := writeFile(wfile, zFile, x.FileMode, x.DirMode)
	if err != nil {
		return s, wfile, fmt.Errorf("%s: %w: %s (from: %s)", zipFile.FileInfo().Name(), err, wfile, zipFile.Name)
	}

	return s, wfile, nil
}

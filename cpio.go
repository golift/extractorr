package xtractr

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/cavaliergopher/cpio"
)

// ExtractCPIOGzip extracts a gzip-compressed cpio archive (cpgz).
func ExtractCPIOGzip(xFile *XFile) (int64, []string, error) {
	compressedFile, err := os.Open(xFile.FilePath)
	if err != nil {
		return 0, nil, fmt.Errorf("os.Open: %w", err)
	}
	defer compressedFile.Close()

	zipStream, err := gzip.NewReader(compressedFile)
	if err != nil {
		return 0, nil, fmt.Errorf("gzip.NewReader: %w", err)
	}
	defer zipStream.Close()

	return xFile.uncpio(zipStream)
}

// ExtractCPIO extracts a .cpio file.
func ExtractCPIO(xFile *XFile) (int64, []string, error) {
	fileReader, err := os.Open(xFile.FilePath)
	if err != nil {
		return 0, nil, fmt.Errorf("os.Open: %w", err)
	}
	defer fileReader.Close()

	return xFile.uncpio(fileReader)
}

func (x *XFile) uncpio(reader io.Reader) (int64, []string, error) {
	zipReader := cpio.NewReader(reader)
	files := []string{}
	size := int64(0)

	for {
		zipFile, err := zipReader.Next()
		if err == io.EOF {
			return size, files, nil
		} else if err != nil {
			return 0, nil, fmt.Errorf("cpio Next() failed: %w", err)
		}

		fSize, err := x.uncpioFile(zipFile, zipReader)
		if err != nil {
			return size, files, fmt.Errorf("%s: %w", x.FilePath, err)
		}

		files = append(files, filepath.Join(x.OutputDir, zipFile.Name)) //nolint: gosec
		size += fSize
	}
}

func (x *XFile) uncpioFile(cpioFile *cpio.Header, cpioReader *cpio.Reader) (int64, error) {
	wfile := x.clean(cpioFile.Name)
	if !strings.HasPrefix(wfile, x.OutputDir) {
		// The file being written is trying to write outside of the base path. Malicious archive?
		return 0, fmt.Errorf("%s: %w: %s (from: %s)", cpioFile.FileInfo().Name(), ErrInvalidPath, wfile, cpioFile.Name)
	}

	if cpioFile.Mode.IsDir() || cpioFile.FileInfo().IsDir() {
		if err := os.MkdirAll(wfile, x.DirMode); err != nil {
			return 0, fmt.Errorf("making cpio dir: %w", err)
		}

		return 0, nil
	}

	// This turns hard links into symlinks.
	if cpioFile.Linkname != "" {
		err := os.Symlink(cpioFile.Linkname, wfile)
		if err != nil {
			return 0, fmt.Errorf("%s symlink: %w: %s (from: %s)", cpioFile.FileInfo().Name(), err, wfile, cpioFile.Name)
		}

		return 0, nil
	}

	// This should turn non-regular files into empty files.
	// ie. sockets, block, character and fifo devices.
	s, err := writeFile(wfile, cpioReader, x.FileMode, x.DirMode)
	if err != nil {
		return s, fmt.Errorf("%s: %w: %s (from: %s)", cpioFile.FileInfo().Name(), err, wfile, cpioFile.Name)
	}

	return s, nil
}

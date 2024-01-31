package xtractr

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/peterebden/ar"
)

// ExtractAr extracts a raw ar archive. Used by debian (.deb) packages.
func ExtractAr(xFile *XFile) (int64, []string, error) {
	arFile, err := os.Open(xFile.FilePath)
	if err != nil {
		return 0, nil, fmt.Errorf("os.Open: %w", err)
	}
	defer arFile.Close()

	return xFile.unAr(arFile)
}

func (x *XFile) unAr(reader io.Reader) (int64, []string, error) {
	arReader := ar.NewReader(reader)
	files := []string{}
	size := int64(0)

	for {
		header, err := arReader.Next()

		switch {
		case errors.Is(err, io.EOF):
			return size, files, nil
		case err != nil:
			return size, files, fmt.Errorf("%s: arReader.Next: %w", x.FilePath, err)
		case header == nil:
			return size, files, fmt.Errorf("%w: %s", ErrInvalidHead, x.FilePath)
		}

		wfile := x.clean(header.Name)
		if !strings.HasPrefix(wfile, x.OutputDir) {
			// The file being written is trying to write outside of our base path. Malicious archive?
			return size, files, fmt.Errorf("%s: %w: %s (from: %s)", x.FilePath, ErrInvalidPath, wfile, header.Name)
		}

		// ar format does not store directory paths. Flat list of files.
		fSize, err := writeFile(wfile, arReader, os.FileMode(header.Mode), x.DirMode)
		if err != nil {
			return size, files, err
		}

		files = append(files, wfile)
		size += fSize
	}
}

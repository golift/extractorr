package xtractr_test

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"golift.io/xtractr"
)

//nolint:gochecknoglobals
var filesInTestArchive = []string{
	"doc.go",
	"files.go",
	"queue.go",
	"rar.go",
	"start.go",
	"zip.go",
}

const (
	testFile     = "test_data/archive.rar"
	testDataSize = int64(20770)
	testPath     = "right_here"
)

type testLogger struct{ t *testing.T }

func (l *testLogger) Debugf(msg string, format ...interface{}) {
	l.t.Helper()

	msg = "[DEBUG] " + msg
	//	l.t.Logf(msg, format...)
	log.Printf(msg, format...)
}

func (l *testLogger) Printf(msg string, format ...interface{}) {
	l.t.Helper()

	msg = "[INFO] " + msg
	//	l.t.Logf(msg, format...)
	log.Printf(msg, format...)
}

func TestSingleFolder(t *testing.T) {
	t.Parallel()

	queue := xtractr.NewQueue(&xtractr.Config{Logger: &testLogger{t: t}})
	xFile := &xtractr.Xtract{
		Name:       "SomeItem",
		SearchPath: testSetupTestDir(t),
		TempFolder: false,
		DeleteOrig: false,
		Password:   "some_password",
		CBChannel:  make(chan *xtractr.Response),
	}

	depth, err := queue.Extract(xFile)
	assert.Equal(t, 0, depth, "there should be 1 item queued now")
	assert.NoError(t, err, "why is there an error?!")

	for resp := range xFile.CBChannel {
		assert.NoError(t, resp.Error, "the test archives should extract without any error")
		assert.Equal(t, 4, len(resp.Archives), "four directories have archives in them")

		if resp.Done {
			assert.Equal(t, len(filesInTestArchive)*4, len(resp.NewFiles), "wrong count of files were extracted")
			assert.Equal(t, testDataSize*4, resp.Size, "wrong amount of data was written")

			break
		}
	}

	// test written files here?
	// each directory should have its own files.
	os.RemoveAll(testPath)
}

// testSetupTestDir creates a temp directory with 4 copies of a rar archive in it.
func testSetupTestDir(t *testing.T) string {
	t.Helper()

	_ = os.MkdirAll(testPath, 0o755)

	name, err := ioutil.TempDir(testPath, "xtractr_test_*_data")
	if err != nil {
		t.Fatalf("could not make temporary directory: %v", err)
	}

	testFileData, err := ioutil.ReadFile(testFile)
	if err != nil {
		t.Fatalf("could not read test data file: %v", err)
	}

	for _, sub := range []string{"subDir1", "subDir2", "subDir3"} {
		err = os.MkdirAll(filepath.Join(name, "subDirectory", sub), 0o755)
		if err != nil {
			t.Fatalf("could not make temporary directory: %v", err)
		}

		fileName := filepath.Join(name, "subDirectory", sub, sub+"_archive.rar")

		err := makeFile(testFileData, fileName)
		if err != nil {
			t.Fatalf("creating test archive: %v", err)
		}
	}

	err = makeFile(testFileData, filepath.Join(name, "subDirectory", "primary_arechive.rar"))
	if err != nil {
		t.Fatalf("creating test archive: %v", err)
	}

	return name
}

//nolint:wrapcheck
func makeFile(data []byte, fileName string) error {
	openFile, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer openFile.Close()

	_, err = openFile.Write(data)

	return err
}
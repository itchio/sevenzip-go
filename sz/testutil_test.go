package sz

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
)

const (
	// URL pattern for downloading native libraries from broth.itch.zone
	// %s is replaced with the platform (e.g., "linux-amd64", "darwin-arm64-head")
	brothLibraryURL = "https://broth.itch.zone/libc7zip/%s/LATEST/archive.zip"

	// useHeadBuild controls whether to use head builds (-head suffix) or stable builds
	useHeadBuild = true
)

var (
	librariesOnce   sync.Once
	librariesError  error
	librariesLoaded bool
)

// ensureNativeLibraries downloads and extracts the native libraries if they don't exist.
// Uses sync.Once to ensure this only happens once per test run.
func ensureNativeLibraries() error {
	librariesOnce.Do(func() {
		librariesError = downloadNativeLibraries()
		if librariesError == nil {
			librariesLoaded = true
		}
	})
	return librariesError
}

// downloadNativeLibraries fetches the native libraries from broth.itch.zone
func downloadNativeLibraries() error {
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}
	execDir := filepath.Dir(execPath)

	// Determine library names based on platform
	var libc7zipName, szName string
	switch runtime.GOOS {
	case "linux":
		libc7zipName = "libc7zip.so"
		szName = "7z.so"
	case "darwin":
		libc7zipName = "libc7zip.dylib"
		szName = "7z.so"
	case "windows":
		libc7zipName = "c7zip.dll"
		szName = "7z.dll"
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	libc7zipPath := filepath.Join(execDir, libc7zipName)
	szPath := filepath.Join(execDir, szName)

	// Check if libraries already exist
	if fileExists(libc7zipPath) && fileExists(szPath) {
		return nil
	}

	// Map GOARCH to broth arch names
	arch := runtime.GOARCH
	if arch == "arm64" {
		// broth uses arm64
	} else if arch == "amd64" {
		// broth uses amd64
	} else if arch == "386" {
		// broth uses 386
	}

	platform := fmt.Sprintf("%s-%s", runtime.GOOS, arch)
	if useHeadBuild {
		platform += "-head"
	}
	url := fmt.Sprintf(brothLibraryURL, platform)

	fmt.Printf("Downloading native libraries from %s...\n", url)

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download libraries: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download libraries: HTTP %d", resp.StatusCode)
	}

	// Read the entire response into memory
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	// Extract the ZIP
	zipReader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return fmt.Errorf("failed to open zip: %w", err)
	}

	for _, f := range zipReader.File {
		name := filepath.Base(f.Name)
		// Only extract the libraries we need
		if name != libc7zipName && name != szName {
			continue
		}

		destPath := filepath.Join(execDir, name)
		fmt.Printf("Extracting %s to %s...\n", name, destPath)

		rc, err := f.Open()
		if err != nil {
			return fmt.Errorf("failed to open %s in zip: %w", f.Name, err)
		}

		outFile, err := os.Create(destPath)
		if err != nil {
			rc.Close()
			return fmt.Errorf("failed to create %s: %w", destPath, err)
		}

		_, err = io.Copy(outFile, rc)
		rc.Close()
		outFile.Close()
		if err != nil {
			return fmt.Errorf("failed to extract %s: %w", f.Name, err)
		}

		// Make libraries executable on Unix
		if runtime.GOOS != "windows" {
			os.Chmod(destPath, 0755)
		}
	}

	return nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// skipIfNoLibraries skips the test if native libraries are not available
func skipIfNoLibraries(t *testing.T) {
	t.Helper()
	if !librariesLoaded {
		t.Skip("Native libraries not available")
	}
}

// bytesReaderAtCloser wraps a bytes.Reader to implement ReaderAtCloser
// It suppresses io.EOF when data was successfully read, as required by
// the sz library's inReadGo function which treats any error as failure.
type bytesReaderAtCloser struct {
	data []byte
}

func (b *bytesReaderAtCloser) ReadAt(p []byte, off int64) (n int, err error) {
	if off < 0 {
		return 0, fmt.Errorf("negative offset")
	}
	// Allow reads at exactly the end with zero length (7-zip does this as a check)
	if off > int64(len(b.data)) {
		return 0, io.EOF
	}
	if off == int64(len(b.data)) {
		// At exactly the end - can only read 0 bytes
		return 0, nil
	}
	n = copy(p, b.data[off:])
	// Don't return EOF if we read any data - the sz library treats any error as failure
	return n, nil
}

func (b *bytesReaderAtCloser) Close() error {
	return nil
}

// newBytesReaderAtCloser creates a ReaderAtCloser from a byte slice
func newBytesReaderAtCloser(data []byte) ReaderAtCloser {
	return &bytesReaderAtCloser{data: data}
}

// createTestZip creates a ZIP archive in memory with the given files
func createTestZip(files map[string][]byte) ([]byte, error) {
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)

	for name, content := range files {
		f, err := w.Create(name)
		if err != nil {
			return nil, err
		}
		_, err = f.Write(content)
		if err != nil {
			return nil, err
		}
	}

	err := w.Close()
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// createTestZipWithDirs creates a ZIP archive with files and explicit directory entries
func createTestZipWithDirs(files map[string][]byte, dirs []string) ([]byte, error) {
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)

	// Create directory entries first
	for _, dir := range dirs {
		// Directory entries must end with /
		dirName := dir
		if dirName[len(dirName)-1] != '/' {
			dirName += "/"
		}
		_, err := w.Create(dirName)
		if err != nil {
			return nil, err
		}
	}

	// Create file entries
	for name, content := range files {
		f, err := w.Create(name)
		if err != nil {
			return nil, err
		}
		_, err = f.Write(content)
		if err != nil {
			return nil, err
		}
	}

	err := w.Close()
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// testExtractCallback implements ExtractCallbackFuncs for testing
type testExtractCallback struct {
	// Extracted files: path -> content
	files map[string]*bytes.Buffer
	// Progress updates: (completed, total) pairs
	progress []progressUpdate
	// Errors encountered
	errors []error
	mu     sync.Mutex
}

type progressUpdate struct {
	completed int64
	total     int64
}

func newTestExtractCallback() *testExtractCallback {
	return &testExtractCallback{
		files:    make(map[string]*bytes.Buffer),
		progress: make([]progressUpdate, 0),
	}
}

func (t *testExtractCallback) SetProgress(completed int64, total int64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.progress = append(t.progress, progressUpdate{completed, total})
}

func (t *testExtractCallback) GetStream(item *Item) (*OutStream, error) {
	path, ok := item.GetStringProperty(PidPath)
	if !ok {
		return nil, fmt.Errorf("could not get item path")
	}

	isDir, _ := item.GetBoolProperty(PidIsDir)
	if isDir {
		// Skip directories
		return nil, nil
	}

	t.mu.Lock()
	buf := new(bytes.Buffer)
	t.files[path] = buf
	t.mu.Unlock()

	return NewOutStream(&nopWriteCloser{buf})
}

// nopWriteCloser wraps a writer with a no-op Close
type nopWriteCloser struct {
	io.Writer
}

func (n *nopWriteCloser) Close() error {
	return nil
}

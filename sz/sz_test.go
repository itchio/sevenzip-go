package sz

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestMain(m *testing.M) {
	if err := ensureNativeLibraries(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: %v (integration tests will skip)\n", err)
	}
	os.Exit(m.Run())
}

func TestNewLib(t *testing.T) {
	skipIfNoLibraries(t)

	lib, err := NewLib()
	if err != nil {
		t.Fatalf("NewLib failed: %v", err)
	}
	defer lib.Free()

	version := lib.GetVersion()
	if version == "" {
		t.Error("GetVersion returned empty string")
	}
	t.Logf("7-zip version: %s", version)
}

func TestNewInStream(t *testing.T) {
	skipIfNoLibraries(t)

	data := []byte("test data for stream")
	reader := newBytesReaderAtCloser(data)

	stream, err := NewInStream(reader, "bin", int64(len(data)))
	if err != nil {
		t.Fatalf("NewInStream failed: %v", err)
	}
	defer stream.Free()

	// Test Seek operations
	pos, err := stream.Seek(5, io.SeekStart)
	if err != nil {
		t.Fatalf("Seek(5, SeekStart) failed: %v", err)
	}
	if pos != 5 {
		t.Errorf("Seek(5, SeekStart) = %d, want 5", pos)
	}

	pos, err = stream.Seek(3, io.SeekCurrent)
	if err != nil {
		t.Fatalf("Seek(3, SeekCurrent) failed: %v", err)
	}
	if pos != 8 {
		t.Errorf("Seek(3, SeekCurrent) = %d, want 8", pos)
	}

	pos, err = stream.Seek(-5, io.SeekEnd)
	if err != nil {
		t.Fatalf("Seek(-5, SeekEnd) failed: %v", err)
	}
	expected := int64(len(data)) - 5
	if pos != expected {
		t.Errorf("Seek(-5, SeekEnd) = %d, want %d", pos, expected)
	}
}

func TestOpenArchive_SimpleZip(t *testing.T) {
	skipIfNoLibraries(t)

	// Create a simple ZIP with one file
	files := map[string][]byte{
		"hello.txt": []byte("Hello, World!"),
	}
	zipData, err := createTestZip(files)
	if err != nil {
		t.Fatalf("createTestZip failed: %v", err)
	}

	lib, err := NewLib()
	if err != nil {
		t.Fatalf("NewLib failed: %v", err)
	}
	defer lib.Free()

	reader := newBytesReaderAtCloser(zipData)
	stream, err := NewInStream(reader, "zip", int64(len(zipData)))
	if err != nil {
		t.Fatalf("NewInStream failed: %v", err)
	}
	defer stream.Free()

	archive, err := lib.OpenArchive(stream, false)
	if err != nil {
		t.Fatalf("OpenArchive failed: %v", err)
	}
	defer archive.Free()
	defer archive.Close()

	format := archive.GetArchiveFormat()
	if format != "zip" {
		t.Errorf("GetArchiveFormat() = %q, want %q", format, "zip")
	}

	count, err := archive.GetItemCount()
	if err != nil {
		t.Fatalf("GetItemCount failed: %v", err)
	}
	if count != 1 {
		t.Errorf("GetItemCount() = %d, want 1", count)
	}
}

func TestOpenArchive_BySignature(t *testing.T) {
	skipIfNoLibraries(t)

	// Create a ZIP but give it the wrong extension
	files := map[string][]byte{
		"test.txt": []byte("test content"),
	}
	zipData, err := createTestZip(files)
	if err != nil {
		t.Fatalf("createTestZip failed: %v", err)
	}

	lib, err := NewLib()
	if err != nil {
		t.Fatalf("NewLib failed: %v", err)
	}
	defer lib.Free()

	// Use wrong extension
	reader := newBytesReaderAtCloser(zipData)
	stream, err := NewInStream(reader, "bin", int64(len(zipData)))
	if err != nil {
		t.Fatalf("NewInStream failed: %v", err)
	}
	defer stream.Free()

	// Open by signature (should detect ZIP format)
	archive, err := lib.OpenArchive(stream, true)
	if err != nil {
		t.Fatalf("OpenArchive by signature failed: %v", err)
	}
	defer archive.Free()
	defer archive.Close()

	format := archive.GetArchiveFormat()
	if format != "zip" {
		t.Errorf("GetArchiveFormat() = %q, want %q (detected by signature)", format, "zip")
	}
}

func TestOpenArchive_InvalidArchive(t *testing.T) {
	skipIfNoLibraries(t)

	// Create invalid archive data
	invalidData := []byte("this is not a valid archive")

	lib, err := NewLib()
	if err != nil {
		t.Fatalf("NewLib failed: %v", err)
	}
	defer lib.Free()

	reader := newBytesReaderAtCloser(invalidData)
	stream, err := NewInStream(reader, "zip", int64(len(invalidData)))
	if err != nil {
		t.Fatalf("NewInStream failed: %v", err)
	}
	defer stream.Free()

	_, err = lib.OpenArchive(stream, false)
	if err == nil {
		t.Error("OpenArchive should fail on invalid data")
	}
}

func TestGetItem_Properties(t *testing.T) {
	skipIfNoLibraries(t)

	// Create a ZIP with a file
	content := []byte("Hello, World!")
	files := map[string][]byte{
		"subdir/hello.txt": content,
	}
	zipData, err := createTestZip(files)
	if err != nil {
		t.Fatalf("createTestZip failed: %v", err)
	}

	lib, err := NewLib()
	if err != nil {
		t.Fatalf("NewLib failed: %v", err)
	}
	defer lib.Free()

	reader := newBytesReaderAtCloser(zipData)
	stream, err := NewInStream(reader, "zip", int64(len(zipData)))
	if err != nil {
		t.Fatalf("NewInStream failed: %v", err)
	}
	defer stream.Free()

	archive, err := lib.OpenArchive(stream, false)
	if err != nil {
		t.Fatalf("OpenArchive failed: %v", err)
	}
	defer archive.Free()
	defer archive.Close()

	item := archive.GetItem(0)
	if item == nil {
		t.Fatal("GetItem(0) returned nil")
	}
	defer item.Free()

	// Test PidPath
	path, ok := item.GetStringProperty(PidPath)
	if !ok {
		t.Error("GetStringProperty(PidPath) failed")
	}
	// Normalize path separators for cross-platform comparison
	if filepath.ToSlash(path) != "subdir/hello.txt" {
		t.Errorf("PidPath = %q, want %q", path, "subdir/hello.txt")
	}

	// Test PidSize
	size, ok := item.GetUInt64Property(PidSize)
	if !ok {
		t.Error("GetUInt64Property(PidSize) failed")
	}
	if size != uint64(len(content)) {
		t.Errorf("PidSize = %d, want %d", size, len(content))
	}

	// Test PidIsDir
	isDir, ok := item.GetBoolProperty(PidIsDir)
	if !ok {
		t.Error("GetBoolProperty(PidIsDir) failed")
	}
	if isDir {
		t.Error("PidIsDir = true, want false")
	}
}

func TestExtract_SingleItem(t *testing.T) {
	skipIfNoLibraries(t)

	content := []byte("Hello, extraction test!")
	files := map[string][]byte{
		"test.txt": content,
	}
	zipData, err := createTestZip(files)
	if err != nil {
		t.Fatalf("createTestZip failed: %v", err)
	}

	lib, err := NewLib()
	if err != nil {
		t.Fatalf("NewLib failed: %v", err)
	}
	defer lib.Free()

	reader := newBytesReaderAtCloser(zipData)
	stream, err := NewInStream(reader, "zip", int64(len(zipData)))
	if err != nil {
		t.Fatalf("NewInStream failed: %v", err)
	}
	defer stream.Free()

	archive, err := lib.OpenArchive(stream, false)
	if err != nil {
		t.Fatalf("OpenArchive failed: %v", err)
	}
	defer archive.Free()
	defer archive.Close()

	item := archive.GetItem(0)
	if item == nil {
		t.Fatal("GetItem(0) returned nil")
	}
	defer item.Free()

	// Extract to a buffer
	buf := new(bytes.Buffer)
	outStream, err := NewOutStream(&nopWriteCloser{buf})
	if err != nil {
		t.Fatalf("NewOutStream failed: %v", err)
	}
	defer outStream.Free()
	defer outStream.Close()

	err = archive.Extract(item, outStream)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if !bytes.Equal(buf.Bytes(), content) {
		t.Errorf("Extracted content = %q, want %q", buf.String(), string(content))
	}
}

func TestExtractSeveral_MultipleFiles(t *testing.T) {
	skipIfNoLibraries(t)

	files := map[string][]byte{
		"file1.txt": []byte("content of file 1"),
		"file2.txt": []byte("content of file 2"),
		"file3.txt": []byte("content of file 3"),
	}
	zipData, err := createTestZip(files)
	if err != nil {
		t.Fatalf("createTestZip failed: %v", err)
	}

	lib, err := NewLib()
	if err != nil {
		t.Fatalf("NewLib failed: %v", err)
	}
	defer lib.Free()

	reader := newBytesReaderAtCloser(zipData)
	stream, err := NewInStream(reader, "zip", int64(len(zipData)))
	if err != nil {
		t.Fatalf("NewInStream failed: %v", err)
	}
	defer stream.Free()

	archive, err := lib.OpenArchive(stream, false)
	if err != nil {
		t.Fatalf("OpenArchive failed: %v", err)
	}
	defer archive.Free()
	defer archive.Close()

	count, err := archive.GetItemCount()
	if err != nil {
		t.Fatalf("GetItemCount failed: %v", err)
	}

	// Create extraction callback
	callback := newTestExtractCallback()
	ec, err := NewExtractCallback(callback)
	if err != nil {
		t.Fatalf("NewExtractCallback failed: %v", err)
	}
	defer ec.Free()

	// Extract all items
	indices := make([]int64, count)
	for i := int64(0); i < count; i++ {
		indices[i] = i
	}

	err = archive.ExtractSeveral(indices, ec)
	if err != nil {
		t.Fatalf("ExtractSeveral failed: %v", err)
	}

	// Verify extracted content
	if len(callback.files) != len(files) {
		t.Errorf("Extracted %d files, want %d", len(callback.files), len(files))
	}

	for name, expectedContent := range files {
		buf, ok := callback.files[name]
		if !ok {
			t.Errorf("File %q not extracted", name)
			continue
		}
		if !bytes.Equal(buf.Bytes(), expectedContent) {
			t.Errorf("File %q: content = %q, want %q", name, buf.String(), string(expectedContent))
		}
	}

	// Check for errors
	if errs := ec.Errors(); len(errs) > 0 {
		for _, e := range errs {
			t.Errorf("Extraction error: %v", e)
		}
	}
}

func TestExtractSeveral_WithDirectories(t *testing.T) {
	skipIfNoLibraries(t)

	files := map[string][]byte{
		"dir/file.txt": []byte("file in directory"),
	}
	dirs := []string{"dir"}

	zipData, err := createTestZipWithDirs(files, dirs)
	if err != nil {
		t.Fatalf("createTestZipWithDirs failed: %v", err)
	}

	lib, err := NewLib()
	if err != nil {
		t.Fatalf("NewLib failed: %v", err)
	}
	defer lib.Free()

	reader := newBytesReaderAtCloser(zipData)
	stream, err := NewInStream(reader, "zip", int64(len(zipData)))
	if err != nil {
		t.Fatalf("NewInStream failed: %v", err)
	}
	defer stream.Free()

	archive, err := lib.OpenArchive(stream, false)
	if err != nil {
		t.Fatalf("OpenArchive failed: %v", err)
	}
	defer archive.Free()
	defer archive.Close()

	count, err := archive.GetItemCount()
	if err != nil {
		t.Fatalf("GetItemCount failed: %v", err)
	}

	// Should have both directory and file entries
	if count < 2 {
		t.Errorf("GetItemCount() = %d, want at least 2 (dir + file)", count)
	}

	callback := newTestExtractCallback()
	ec, err := NewExtractCallback(callback)
	if err != nil {
		t.Fatalf("NewExtractCallback failed: %v", err)
	}
	defer ec.Free()

	indices := make([]int64, count)
	for i := int64(0); i < count; i++ {
		indices[i] = i
	}

	err = archive.ExtractSeveral(indices, ec)
	if err != nil {
		t.Fatalf("ExtractSeveral failed: %v", err)
	}

	// Should have extracted the file (directories are skipped by our callback)
	if len(callback.files) != 1 {
		t.Errorf("Extracted %d files, want 1", len(callback.files))
	}

	if errs := ec.Errors(); len(errs) > 0 {
		for _, e := range errs {
			t.Errorf("Extraction error: %v", e)
		}
	}
}

func TestExtractCallback_ProgressReported(t *testing.T) {
	skipIfNoLibraries(t)

	// Create a larger file to ensure progress is reported
	largeContent := bytes.Repeat([]byte("x"), 10000)
	files := map[string][]byte{
		"large.txt": largeContent,
	}
	zipData, err := createTestZip(files)
	if err != nil {
		t.Fatalf("createTestZip failed: %v", err)
	}

	lib, err := NewLib()
	if err != nil {
		t.Fatalf("NewLib failed: %v", err)
	}
	defer lib.Free()

	reader := newBytesReaderAtCloser(zipData)
	stream, err := NewInStream(reader, "zip", int64(len(zipData)))
	if err != nil {
		t.Fatalf("NewInStream failed: %v", err)
	}
	defer stream.Free()

	archive, err := lib.OpenArchive(stream, false)
	if err != nil {
		t.Fatalf("OpenArchive failed: %v", err)
	}
	defer archive.Free()
	defer archive.Close()

	callback := newTestExtractCallback()
	ec, err := NewExtractCallback(callback)
	if err != nil {
		t.Fatalf("NewExtractCallback failed: %v", err)
	}
	defer ec.Free()

	err = archive.ExtractSeveral([]int64{0}, ec)
	if err != nil {
		t.Fatalf("ExtractSeveral failed: %v", err)
	}

	// Progress callbacks should have been called
	// Note: The exact number depends on 7-zip's internal behavior
	t.Logf("Progress updates received: %d", len(callback.progress))
	for i, p := range callback.progress {
		t.Logf("  Progress[%d]: %d / %d", i, p.completed, p.total)
	}
}

func TestReadStats(t *testing.T) {
	// This test doesn't need native libraries
	stats := &ReadStats{}

	stats.RecordRead(0, 100)
	stats.RecordRead(100, 50)
	stats.RecordRead(150, 200)

	if len(stats.Reads) != 3 {
		t.Errorf("len(Reads) = %d, want 3", len(stats.Reads))
	}

	expected := []ReadOp{
		{Offset: 0, Size: 100},
		{Offset: 100, Size: 50},
		{Offset: 150, Size: 200},
	}

	for i, exp := range expected {
		if stats.Reads[i].Offset != exp.Offset || stats.Reads[i].Size != exp.Size {
			t.Errorf("Reads[%d] = {%d, %d}, want {%d, %d}",
				i, stats.Reads[i].Offset, stats.Reads[i].Size, exp.Offset, exp.Size)
		}
	}
}

func TestInStream_WithStats(t *testing.T) {
	skipIfNoLibraries(t)

	content := []byte("Hello, stats test!")
	files := map[string][]byte{
		"test.txt": content,
	}
	zipData, err := createTestZip(files)
	if err != nil {
		t.Fatalf("createTestZip failed: %v", err)
	}

	lib, err := NewLib()
	if err != nil {
		t.Fatalf("NewLib failed: %v", err)
	}
	defer lib.Free()

	reader := newBytesReaderAtCloser(zipData)
	stream, err := NewInStream(reader, "zip", int64(len(zipData)))
	if err != nil {
		t.Fatalf("NewInStream failed: %v", err)
	}
	defer stream.Free()

	// Enable stats tracking
	stream.Stats = &ReadStats{}

	archive, err := lib.OpenArchive(stream, false)
	if err != nil {
		t.Fatalf("OpenArchive failed: %v", err)
	}
	defer archive.Free()
	defer archive.Close()

	callback := newTestExtractCallback()
	ec, err := NewExtractCallback(callback)
	if err != nil {
		t.Fatalf("NewExtractCallback failed: %v", err)
	}
	defer ec.Free()

	err = archive.ExtractSeveral([]int64{0}, ec)
	if err != nil {
		t.Fatalf("ExtractSeveral failed: %v", err)
	}

	// Verify that reads were recorded
	if len(stream.Stats.Reads) == 0 {
		t.Error("No reads were recorded in Stats")
	}

	t.Logf("Recorded %d read operations", len(stream.Stats.Reads))
	for i, op := range stream.Stats.Reads {
		t.Logf("  Read[%d]: offset=%d, size=%d", i, op.Offset, op.Size)
	}
}

func TestInStream_ChunkSize(t *testing.T) {
	skipIfNoLibraries(t)

	content := bytes.Repeat([]byte("x"), 10000)
	files := map[string][]byte{
		"large.txt": content,
	}
	zipData, err := createTestZip(files)
	if err != nil {
		t.Fatalf("createTestZip failed: %v", err)
	}

	lib, err := NewLib()
	if err != nil {
		t.Fatalf("NewLib failed: %v", err)
	}
	defer lib.Free()

	reader := newBytesReaderAtCloser(zipData)
	stream, err := NewInStream(reader, "zip", int64(len(zipData)))
	if err != nil {
		t.Fatalf("NewInStream failed: %v", err)
	}
	defer stream.Free()

	// Set a small chunk size to force multiple reads
	stream.ChunkSize = 512
	stream.Stats = &ReadStats{}

	archive, err := lib.OpenArchive(stream, false)
	if err != nil {
		t.Fatalf("OpenArchive failed: %v", err)
	}
	defer archive.Free()
	defer archive.Close()

	callback := newTestExtractCallback()
	ec, err := NewExtractCallback(callback)
	if err != nil {
		t.Fatalf("NewExtractCallback failed: %v", err)
	}
	defer ec.Free()

	err = archive.ExtractSeveral([]int64{0}, ec)
	if err != nil {
		t.Fatalf("ExtractSeveral failed: %v", err)
	}

	// Verify that reads respect chunk size
	for i, op := range stream.Stats.Reads {
		if op.Size > stream.ChunkSize {
			t.Errorf("Read[%d] size %d exceeds ChunkSize %d", i, op.Size, stream.ChunkSize)
		}
	}

	t.Logf("Recorded %d read operations with ChunkSize=%d", len(stream.Stats.Reads), stream.ChunkSize)
}

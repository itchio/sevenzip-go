package sz

/*
#include <stdlib.h> // for C.free
#include "glue.h"

// forward declaration for gateway functions
int inReadGo_cgo(int64_t id, void *data, int64_t size, int64_t *processed_size);
int inSeekGo_cgo(int64_t id, int64_t offset, int32_t whence, int64_t *new_position);

int outWriteGo_cgo(int64_t id, const void *data, int64_t size, int64_t *processed_size);
void outCloseGo_cgo(int64_t id);
*/
import "C"
import (
	"fmt"
	"io"
	"os"
	"reflect"
	"unsafe"

	"github.com/go-errors/errors"
)

type ReaderAtCloser interface {
	io.ReaderAt
	io.Closer
}

type InStream struct {
	reader ReaderAtCloser
	size   int64
	id     int64
	offset int64
	strm   *C.in_stream
	err    error

	Stats *ReadStats

	ChunkSize int64
}

type OutStream struct {
	writer io.WriteCloser
	id     int64
	strm   *C.out_stream
	closed bool
	err    error

	ChunkSize int64
}

type Lib struct {
	lib *C.lib
}

func NewLib() (*Lib, error) {
	ret := C.libc7zip_initialize()
	if ret != 0 {
		return nil, fmt.Errorf("could not initialize libc7zip")
	}

	lib := C.libc7zip_lib_new()
	if lib == nil {
		return nil, fmt.Errorf("could not create new lib")
	}

	l := &Lib{
		lib: lib,
	}
	return l, nil
}

func (l *Lib) Free() {
	C.libc7zip_lib_free(l.lib)
}

func NewInStream(reader ReaderAtCloser, ext string, size int64) (*InStream, error) {
	strm := C.libc7zip_in_stream_new()
	if strm == nil {
		return nil, fmt.Errorf("could not create new InStream")
	}

	in := &InStream{
		reader: reader,
		size:   size,
		offset: 0,
		strm:   strm,
	}
	reserveInStreamId(in)

	def := C.libc7zip_in_stream_get_def(strm)
	def.size = C.int64_t(in.size)
	def.ext = C.CString(ext)
	def.id = C.int64_t(in.id)
	def.read_cb = (C.read_cb_t)(unsafe.Pointer(C.inReadGo_cgo))
	def.seek_cb = (C.seek_cb_t)(unsafe.Pointer(C.inSeekGo_cgo))

	C.libc7zip_in_stream_commit_def(strm)

	return in, nil
}

func (in *InStream) Free() {
	if in.id > 0 {
		freeInStreamId(in.id)
		in.id = 0
	}

	if in.strm != nil {
		C.libc7zip_in_stream_free(in.strm)
		in.strm = nil
	}
}

func (in *InStream) Error() error {
	return in.err
}

func NewOutStream(writer io.WriteCloser) (*OutStream, error) {
	strm := C.libc7zip_out_stream_new()
	if strm == nil {
		return nil, fmt.Errorf("could not create new OutStream")
	}

	out := &OutStream{
		writer: writer,
		strm:   strm,
	}
	reserveOutStreamId(out)

	def := C.libc7zip_out_stream_get_def(strm)
	def.id = C.int64_t(out.id)
	def.write_cb = (C.write_cb_t)(unsafe.Pointer(C.outWriteGo_cgo))
	def.close_cb = (C.close_cb_t)(unsafe.Pointer(C.outCloseGo_cgo))

	return out, nil
}

func (out *OutStream) Close() error {
	if out.id > 0 {
		freeOutStreamId(out.id)
		out.id = 0
		return out.writer.Close()
	}

	// already closed
	return nil
}

func (out *OutStream) Free() {
	if out.strm != nil {
		C.libc7zip_out_stream_free(out.strm)
		out.strm = nil
	}
}

func (out *OutStream) Error() error {
	return out.err
}

type Archive struct {
	arch *C.archive
	in   *InStream
}

func (lib *Lib) OpenArchive(in *InStream) (*Archive, error) {
	arch := C.libc7zip_archive_open(lib.lib, in.strm)
	if arch == nil {
		// TODO: relay actual 7-zip errors
		return nil, fmt.Errorf("could not open archive")
	}

	a := &Archive{
		arch: arch,
		in:   in,
	}
	return a, nil
}

func (a *Archive) GetItemCount() (int64, error) {
	res := int64(C.libc7zip_archive_get_item_count(a.arch))
	if res < 0 {
		return 0, errors.Wrap(a.Error(), 0)
	}
	return res, nil
}

func (a *Archive) Error() error {
	// TODO: relay actual 7-zip errors
	return a.in.err
}

type Item struct {
	item *C.item
}

func (a *Archive) GetItem(index int64) *Item {
	item := C.libc7zip_archive_get_item(a.arch, C.int64_t(index))
	if item == nil {
		return nil
	}

	return &Item{
		item: item,
	}
}

type PropertyIndex int32

var (
	// Packed Size
	PidPackSize PropertyIndex = C.kpidPackSize
	// Attributes
	PidAttrib PropertyIndex = C.kpidAttrib
	// Created
	PidCTime PropertyIndex = C.kpidCTime
	// Accessed
	PidATime PropertyIndex = C.kpidATime
	// Modified
	PidMTime PropertyIndex = C.kpidMTime
	// Solid
	PidSolid PropertyIndex = C.kpidSolid
	// Encrypted
	PidEncrypted PropertyIndex = C.kpidEncrypted
	// User
	PidUser PropertyIndex = C.kpidUser
	// Group
	PidGroup PropertyIndex = C.kpidGroup
	// Comment
	PidComment PropertyIndex = C.kpidComment
	// Physical Size
	PidPhySize PropertyIndex = C.kpidPhySize
	// Headers Size
	PidHeadersSize PropertyIndex = C.kpidHeadersSize
	// Checksum
	PidChecksum PropertyIndex = C.kpidChecksum
	// Characteristics
	PidCharacts PropertyIndex = C.kpidCharacts
	// Creator Application
	PidCreatorApp PropertyIndex = C.kpidCreatorApp
	// Total Size
	PidTotalSize PropertyIndex = C.kpidTotalSize
	// Free Space
	PidFreeSpace PropertyIndex = C.kpidFreeSpace
	// Cluster Size
	PidClusterSize PropertyIndex = C.kpidClusterSize
	// Label
	PidVolumeName PropertyIndex = C.kpidVolumeName
	// FullPath
	PidPath PropertyIndex = C.kpidPath
	// IsDir
	PidIsDir PropertyIndex = C.kpidIsDir
	// Uncompressed Size
	PidSize PropertyIndex = C.kpidSize
)

func (i *Item) GetStringProperty(id PropertyIndex) string {
	cstr := C.libc7zip_item_get_string_property(i.item, C.int32_t(id))
	if cstr == nil {
		return ""
	}

	defer C.free(unsafe.Pointer(cstr))
	return C.GoString(cstr)
}

func (i *Item) GetUInt64Property(id PropertyIndex) uint64 {
	return uint64(C.libc7zip_item_get_uint64_property(i.item, C.int32_t(id)))
}

func (i *Item) GetBoolProperty(id PropertyIndex) bool {
	return C.libc7zip_item_get_bool_property(i.item, C.int32_t(id)) != 0
}

func (i *Item) Free() {
	C.libc7zip_item_free(i.item)
}

var ErrExtractionGeneric = errors.New("unknown extraction error")

func (a *Archive) Extract(i *Item, out *OutStream) error {
	success := C.libc7zip_archive_extract_item(a.arch, i.item, out.strm)
	if success == 0 {
		err := a.Error()
		if err != nil {
			return errors.Wrap(err, 0)
		}

		err = out.Error()
		if err != nil {
			return errors.Wrap(err, 0)
		}

		return errors.Wrap(ErrExtractionGeneric, 0)
	}

	return nil
}

//export inSeekGo
func inSeekGo(id int64, offset int64, whence int32, newPosition unsafe.Pointer) int {
	is, ok := inStreams[id]
	if !ok {
		fmt.Fprintf(os.Stderr, "sz: no such InStream: %d", id)
		return 1
	}

	switch whence {
	case io.SeekStart:
		is.offset = offset
	case io.SeekCurrent:
		is.offset += offset
	case io.SeekEnd:
		is.offset = is.size + offset
	}

	newPosPtr := (*int64)(newPosition)
	*newPosPtr = is.offset

	return 0
}

//export inReadGo
func inReadGo(id int64, data unsafe.Pointer, size int64, processedSize unsafe.Pointer) int {
	is, ok := inStreams[id]
	if !ok {
		fmt.Fprintf(os.Stderr, "sz: no such InStream: %d", id)
		return 1
	}

	if is.ChunkSize > 0 && size > is.ChunkSize {
		size = is.ChunkSize
	}

	if is.offset+size > is.size {
		size = is.size - is.offset
	}

	if is.Stats != nil {
		is.Stats.RecordRead(is.offset, size)
	}

	h := reflect.SliceHeader{
		Data: uintptr(data),
		Cap:  int(size),
		Len:  int(size),
	}
	buf := *(*[]byte)(unsafe.Pointer(&h))

	readBytes, err := is.reader.ReadAt(buf, is.offset)
	if err != nil {
		fmt.Fprintf(os.Stderr, "sz: readAt error: %s", err.Error())
		return 1
	}

	is.offset += int64(readBytes)

	processedSizePtr := (*int64)(processedSize)
	*processedSizePtr = int64(readBytes)

	return 0
}

//export outWriteGo
func outWriteGo(id int64, data unsafe.Pointer, size int64, processedSize unsafe.Pointer) int {
	out, ok := outStreams[id]
	if !ok {
		// should never happen
		fmt.Fprintf(os.Stderr, "sz: no such OutStream: %d", id)
		return 1
	}

	if out.ChunkSize > 0 && size > out.ChunkSize {
		size = out.ChunkSize
	}

	h := reflect.SliceHeader{
		Data: uintptr(data),
		Cap:  int(size),
		Len:  int(size),
	}
	buf := *(*[]byte)(unsafe.Pointer(&h))

	writtenBytes, err := out.writer.Write(buf)
	if err != nil {
		out.err = err
		return 1
	}

	processedSizePtr := (*int64)(processedSize)
	*processedSizePtr = int64(writtenBytes)

	return 0
}

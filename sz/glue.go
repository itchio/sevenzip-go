package sz

/*
#include "glue.h"

// forward declaration for gateway functions
int inReadGo_cgo(int64_t id, void *data, int64_t size, int64_t *processed_size);
int inSeekGo_cgo(int64_t id, int64_t offset, int32_t whence, int64_t *new_position);

int outWriteGo_cgo(int64_t id, const void *data, int64_t size, int64_t *processed_size);
int outSeekGo_cgo(int64_t id, int64_t offset, int32_t whence, int64_t *new_position);
*/
import "C"
import (
	"fmt"
	"io"
	"log"
	"reflect"
	"unsafe"
)

// const kTestMaxOpSize int64 = 512 * 1024

type ReaderAtCloser interface {
	io.ReaderAt
	io.Closer
}

// TODO: handle Close() correctly
type InStream struct {
	reader ReaderAtCloser
	size   int64
	id     int64
	offset int64
	strm   *C.in_stream
}

var inStreams = make(map[int64]*InStream)
var inStreamSeed int64 = 666

type WriterAtCloser interface {
	io.WriterAt
	io.Closer
}

type OutStream struct {
	writer WriterAtCloser
	size   int64
	id     int64
	offset int64
	strm   *C.out_stream
}

var outStreams = make(map[int64]*OutStream)
var outStreamSeed int64 = 777

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

	is := &InStream{
		reader: reader,
		size:   size,
		id:     inStreamSeed,
		offset: 0,
		strm:   strm,
	}
	inStreams[is.id] = is
	inStreamSeed += 1

	def := C.libc7zip_in_stream_get_def(strm)
	def.size = C.int64_t(is.size)
	def.ext = C.CString(ext)
	def.id = C.int64_t(is.id)
	def.read_cb = (C.read_cb_t)(unsafe.Pointer(C.inReadGo_cgo))
	def.seek_cb = (C.seek_cb_t)(unsafe.Pointer(C.inSeekGo_cgo))

	C.libc7zip_in_stream_commit_def(strm)

	return is, nil
}

func (is *InStream) Free() {
	C.libc7zip_in_stream_free(is.strm)
}

func NewOutStream(writer WriterAtCloser) (*OutStream, error) {
	strm := C.libc7zip_out_stream_new()
	if strm == nil {
		return nil, fmt.Errorf("could not create new OutStream")
	}

	os := &OutStream{
		writer: writer,
		id:     outStreamSeed,
		offset: 0,
		size:   0, // TODO: what about this?
		strm:   strm,
	}
	outStreams[os.id] = os
	outStreamSeed += 1

	def := C.libc7zip_out_stream_get_def(strm)
	def.id = C.int64_t(os.id)
	def.write_cb = (C.write_cb_t)(unsafe.Pointer(C.outWriteGo_cgo))
	def.seek_cb = (C.seek_cb_t)(unsafe.Pointer(C.outSeekGo_cgo))

	return os, nil
}

func (os *OutStream) Free() {
	C.libc7zip_out_stream_free(os.strm)
}

type Archive struct {
	arch *C.archive
}

func (lib *Lib) OpenArchive(is *InStream) (*Archive, error) {
	arch := C.libc7zip_archive_open(lib.lib, is.strm)
	if arch == nil {
		return nil, fmt.Errorf("could not open archive")
	}

	a := &Archive{
		arch: arch,
	}
	return a, nil
}

func (a *Archive) GetItemCount() int64 {
	return int64(C.libc7zip_archive_get_item_count(a.arch))
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

func (i *Item) Free() {
	C.libc7zip_archive_item_free(i.item)
}

func (a *Archive) Extract(i *Item, os *OutStream) error {
	success := C.libc7zip_archive_extract(a.arch, i.item, os.strm)
	if success == 0 {
		return fmt.Errorf(`extraction was not successful`)
	}

	return nil
}

//export inSeekGo
func inSeekGo(id int64, offset int64, whence int32, newPosition unsafe.Pointer) int {
	is, ok := inStreams[id]
	if !ok {
		log.Printf("no such InStream: %d", id)
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
		log.Printf("no such InStream: %d", id)
		return 1
	}

	// FIXME: just testing things
	// if size > kTestMaxOpSize {
	// 	size = kTestMaxOpSize
	// }

	log.Printf("[%d] inRead %d bytes at %d", id, size, is.offset)

	h := reflect.SliceHeader{
		Data: uintptr(data),
		Cap:  int(size),
		Len:  int(size),
	}
	buf := *(*[]byte)(unsafe.Pointer(&h))

	readBytes, err := is.reader.ReadAt(buf, is.offset)
	if err != nil {
		return 1
	}

	is.offset += int64(readBytes)

	processedSizePtr := (*int64)(processedSize)
	*processedSizePtr = int64(readBytes)

	return 0
}

//export outSeekGo
func outSeekGo(id int64, offset int64, whence int32, newPosition unsafe.Pointer) int {
	os, ok := outStreams[id]
	if !ok {
		log.Printf("no such OutStream: %d", id)
		return 1
	}

	switch whence {
	case io.SeekStart:
		os.offset = offset
	case io.SeekCurrent:
		os.offset += offset
	case io.SeekEnd:
		os.offset = os.size + offset
	}

	newPosPtr := (*int64)(newPosition)
	*newPosPtr = os.offset

	return 0
}

//export outWriteGo
func outWriteGo(id int64, data unsafe.Pointer, size int64, processedSize unsafe.Pointer) int {
	os, ok := outStreams[id]
	if !ok {
		log.Printf("no such OutStream: %d", id)
		return 1
	}

	// FIXME: just testing things
	// if size > kTestMaxOpSize {
	// 	size = kTestMaxOpSize
	// }

	log.Printf("[%d] outWrite %d bytes at %d", id, size, os.offset)

	h := reflect.SliceHeader{
		Data: uintptr(data),
		Cap:  int(size),
		Len:  int(size),
	}
	buf := *(*[]byte)(unsafe.Pointer(&h))

	writtenBytes, err := os.writer.WriteAt(buf, os.offset)
	if err != nil {
		return 1
	}

	os.offset += int64(writtenBytes)

	processedSizePtr := (*int64)(processedSize)
	*processedSizePtr = int64(writtenBytes)

	return 0
}

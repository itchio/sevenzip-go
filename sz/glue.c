
#define GLUE_IMPLEMENT 1
#include "glue.h"

#include "_cgo_export.h"

#include <stdio.h>

#ifdef __WIN32
#include <windows.h>
#endif

#define LOADSYM(sym) { \
  sym ## _ = (void*) GetProcAddress(dynlib, #sym); \
  if (! sym ## _) { \
    fprintf(stderr, "Could not load symbol %s\n", #sym); \
    fflush(stderr); \
    return 1; \
  } \
}

HMODULE dynlib;

int libc7zip_initialize() {
  dynlib = LoadLibrary("libc7zip.dll");
  if (!dynlib) {
    fprintf(stderr, "Could not load libc7zip.dll\n");
    fflush(stderr);
    return 1;
  }

  LOADSYM(lib_new)
  LOADSYM(lib_free)

  LOADSYM(in_stream_new)
  LOADSYM(in_stream_get_def)
  LOADSYM(in_stream_free)

  LOADSYM(out_stream_new)
  LOADSYM(out_stream_get_def)
  LOADSYM(out_stream_free)

  LOADSYM(archive_open)
  LOADSYM(archive_get_item_count)

  return 0;
}

lib *libc7zip_lib_new() {
  return lib_new_();
}

void libc7zip_lib_free(lib *l) {
  return lib_free_(l);
}

//-----------------

in_stream *libc7zip_in_stream_new() {
  return in_stream_new_();
}

in_stream_def *libc7zip_in_stream_get_def(in_stream *is) {
  return in_stream_get_def_(is);
}

void libc7zip_in_stream_free(in_stream *is) {
  return in_stream_free_(is);
}

//-----------------

out_stream *libc7zip_out_stream_new() {
  return out_stream_new_();
}

out_stream_def *libc7zip_out_stream_get_def(out_stream *os) {
  return out_stream_get_def_(os);
}

void libc7zip_out_stream_free(out_stream *os) {
  return out_stream_free_(os);
}

//-----------------

archive *libc7zip_archive_open(lib *l, in_stream *is) {
  return archive_open_(l, is);
}

int64_t libc7zip_archive_get_item_count(archive *a) {
  return archive_get_item_count_(a);
}

// Gateway functions

int inSeekGo_cgo(int64_t id, int64_t offset, int32_t whence, int64_t *new_position) {
  return inSeekGo(id, offset, whence, new_position);
}

int inReadGo_cgo(int64_t id, void *data, int64_t size, int64_t *processed_size) {
  return inReadGo(id, data, size, processed_size);
}

int outSeekGo_cgo(int64_t id, int64_t offset, int32_t whence, int64_t *new_position) {
  return outSeekGo(id, offset, whence, new_position);
}

int outWriteGo_cgo(int64_t id, const void *data, int64_t size, int64_t *processed_size) {
  return outWriteGo(id, (void*) data, size, processed_size);
}


#ifndef LIBC7ZIP_H
#define LIBC7ZIP_H

#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif // __cplusplus

typedef int (*read_cb_t)(int64_t id, void *data, int64_t size, int64_t *processed_size);
typedef int (*seek_cb_t)(int64_t id, int64_t offset, int32_t whence, int64_t *new_position);

typedef struct in_stream_def {
  int64_t id;
	seek_cb_t seek_cb;
	read_cb_t read_cb;
  char *ext;
  int64_t size;
} in_stream_def;

struct lib;
typedef struct lib lib;

lib *lib_new();

struct in_stream;
typedef struct in_stream in_stream;
in_stream *in_stream_new();
in_stream_def *in_stream_get_def(in_stream *s);

struct archive;
typedef struct archive archive;
archive *archive_open(lib *l, in_stream *s);
int64_t archive_get_item_count(archive *a);

#ifdef __cplusplus
} // extern "C"
#endif // __cplusplus

#endif // LIBC7ZIP_H
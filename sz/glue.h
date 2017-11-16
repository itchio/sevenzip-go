
#include "libc7zip.h"

#ifdef GLUE_IMPLEMENT
#define GLUE 
#else
#define GLUE extern
#endif

#define DECLARE(x) GLUE x##_t x##_;

typedef lib *(*lib_new_t)();
DECLARE(lib_new)

typedef in_stream *(*in_stream_new_t)();
DECLARE(in_stream_new)

typedef in_stream_def *(*in_stream_get_def_t)();
DECLARE(in_stream_get_def)

typedef archive *(*archive_open_t)(lib *l, in_stream *s);
DECLARE(archive_open)

int libc7zip_initialize();
lib *libc7zip_lib_new();
in_stream *libc7zip_in_stream_new();
in_stream_def *libc7zip_in_stream_get_def(in_stream *s);
archive *libc7zip_archive_open(lib *l, in_stream *s);

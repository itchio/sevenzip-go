
#include "libc7zip.h"

#ifdef GLUE_IMPLEMENT
#define GLUE 
#else
#define GLUE extern
#endif

#define DECLARE(x) GLUE x##_t x##_;

int libc7zip_initialize();

// lib_new
typedef lib *(*lib_new_t)();
DECLARE(lib_new)
lib *libc7zip_lib_new();

// lib_free
typedef void (*lib_free_t)(lib *l);
DECLARE(lib_free)
void libc7zip_lib_free(lib *l);

// in_stream_new
typedef in_stream *(*in_stream_new_t)();
DECLARE(in_stream_new)
in_stream *libc7zip_in_stream_new();

// in_stream_get_def
typedef in_stream_def *(*in_stream_get_def_t)(in_stream *is);
DECLARE(in_stream_get_def)
in_stream_def *libc7zip_in_stream_get_def(in_stream *is);

// in_stream_commit_def
typedef void (*in_stream_commit_def_t)(in_stream *is);
DECLARE(in_stream_commit_def)
void libc7zip_in_stream_commit_def(in_stream *is);

// in_stream_free
typedef void (*in_stream_free_t)(in_stream *is);
DECLARE(in_stream_free)
void libc7zip_in_stream_free(in_stream *is);

// out_stream_new
typedef out_stream *(*out_stream_new_t)();
DECLARE(out_stream_new)
out_stream *libc7zip_out_stream_new();

// out_stream_get_def
typedef out_stream_def *(*out_stream_get_def_t)();
DECLARE(out_stream_get_def)
out_stream_def *libc7zip_out_stream_get_def(out_stream *s);

// out_stream_free
typedef void (*out_stream_free_t)(out_stream *os);
DECLARE(out_stream_free)
void libc7zip_out_stream_free(out_stream *os);

// archive_open
typedef archive *(*archive_open_t)(lib *l, in_stream *s);
DECLARE(archive_open)
archive *libc7zip_archive_open(lib *l, in_stream *s);

// archive_get_item_count
typedef int64_t (*archive_get_item_count_t)(archive *a);
DECLARE(archive_get_item_count)
int64_t libc7zip_archive_get_item_count(archive *a);

// archive_get_item
typedef item *(*archive_get_item_t)(archive *a, int64_t index);
DECLARE(archive_get_item)
item *libc7zip_archive_get_item(archive *a, int64_t index);

// item_get_string_property
typedef char *(*item_get_string_property_t)(item *i, int32_t property_index);
DECLARE(item_get_string_property)
char *libc7zip_item_get_string_property(item *i, int32_t property_index);

// item_get_uint64_property
typedef uint64_t (*item_get_uint64_property_t)(item *i, int32_t property_index);
DECLARE(item_get_uint64_property)
uint64_t libc7zip_item_get_uint64_property(item *i, int32_t property_index);

// item_get_bool_property
typedef int32_t (*item_get_bool_property_t)(item *i, int32_t property_index);
DECLARE(item_get_bool_property)
int32_t libc7zip_item_get_bool_property(item *i, int32_t property_index);

// item_free
typedef void (*item_free_t)(item *i);
DECLARE(item_free)
void libc7zip_item_free(item *i);

// archive_extract_item
typedef int (*archive_extract_item_t)(archive *a, item *i, out_stream *os);
DECLARE(archive_extract_item)
int libc7zip_archive_extract_item(archive *a, item *i, out_stream *os);

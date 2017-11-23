# sevenzip-go

Bindings to use 7-zip as a library from golang.

### Structure

sevenzip-go needs two dynamic libraries to operate, and it expects them to be
in the executable's folder.

For example

  * On Windows, you'll need `foobar.exe`, `c7zip.dll`, and `7z.dll` in the same directory
  * On Linux, you'll need `foobar`, `libc7zip.so`, and `7z.so` in the same directory
  * On macOS, you'll need `foobar`, `libc7zip.dylib`, and `7z.so` in the same directory

Note: the 7-zip library is called `7z.so` on macOS, that's not a typo.

If it can't find it, it'll print messages to stderr (and return an error).

#### Rationale

sevenzip-go was made primarily to serve as a decompression engine for <https://github.com/itchio/butler>

most of butler's functionality does not require 7-zip, and:

  * we want folks to be able to build butler easily, without having to build C/C++ projects manually
  * we want folks to be able to run their custom butler builds easily, without having to worry about missing
  dynamic libraries
  * we want to use `7z.dll` from the official 7-zip builds (it is a notorious pain to build, as it requires MSVC 2010)

While the whole setup sounds crazy (especially considering the whole Go->cgo->C->C++->COM/C++ pipeline),
it fits all those goals.

### Caveats

Pay attention to the dynamic library requirement above:

> Neither sevenzip-go nor lib7zip look for DLLs in the `PATH` or `LD_LIBRARY_PATH` or `DYLD_LIBRARY_PATH`,
> they only look **in the executable's directory**. This is on purpose, so we don't accidentally load
> an older version of 7-zip.

The library allocates memory via C functions, so you should make sure to call `.Free()` on the
various objects you get from sevenzip-go.

Error handling is best-effort, but there's many moving pieces involved here. Some items of an archive
may fail to extract, the errors can be retrieved with `extractCallback.Errors()` (which returns a slice of
errors).

### Example

The `./cmd/go7z` package

### Links

  * <https://github.com/itchio/libc7zip> - a C wrapper for lib7zip, based on structs and function pointers
  * <https://github.com/itchio/lib7zip> - a C++ wrapper for the 7-zip COM API, based on abstract base classes
  * <http://7-zip.org/> - the official 7-zip distribution (Windows)
  * <http://p7zip.sourceforge.net/> - a 7-zip port for Linux/macOS/etc.

### License

sevenzip-go is released under the MIT license, see the `LICENSE` file.

Other required components are distributed under the MPL 2.0, the LGPL 2.1, and
other terms - see their own `LICENSE` or `COPYING` files.

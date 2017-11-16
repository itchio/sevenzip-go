# go-libc7zip

A cgo binding for libc7zip

  * <https://github.com/fasterthanlime/libc7zip> is a C wrapper for...
  * <https://github.com/stonewell/lib7zip> which is a C++ wrapper for either of
    * <http://7-zip.org/> - the official 7-zip distribution (Windows)
    * <http://p7zip.sourceforge.net/> - a 7-zip port for Linux/macOS/etc.

### Usage

See `./cmd/go7z` for a sample.

### How it works

You probably don't want to know.

#### No seriously, how does it work?

Ok so:

  * the `sz` package contains a `glue.c` file
  * it loads the libc7zip library dynamically (with `LoadLibrary` or `dlopen`)
  * ...then it sets function pointers from the loaded library
  * it also contains a bunch of cgo callbacks, and wrappers of libc7zip functions
  * then it lets you interact with all that via Go types

libc7zip is linked statically against lib7zip, which does it
own loading of `7z.dll` (or `lib7z.so`, or `lib7z.dylib`), the point is:
you'll need both `7z.dll` and `libc7zip.dll` in your %PATH% on Windows.

#### Why it works

I have no idea.

#### No I mean, why is it designed like that?

The use case is for `butler` to be able to extract archives on-the-fly using
7-zip's decompression engine (for all its codecs), consuming remote files
(over HTTPS, via itchfs).

But butler is built in various configurations by a bunch of folks, and I don't
want it to stop working if it's missing a dynamic library or two. I also don't
want folks to have to compile 7-zip/p7zip and lib7zip when they just want the
bulk of butler's functionality.

Ergo: double layer of dynamic libraries, everybody's happy.

### License

go-libc7zip is cautiously released under the MIT license, see the `LICENSE` file

package main

import (
	"errors"
	"log"
	"os"
	"path/filepath"

	humanize "github.com/dustin/go-humanize"
	"github.com/fasterthanlime/go-libc7zip/sz"
)

func main() {
	lib, err := sz.NewLib()
	must(err)
	log.Printf("Initialized library...")

	args := os.Args[1:]

	if len(args) < 1 {
		log.Printf("Usage: go7z ARCHIVE")
		os.Exit(1)
	}

	inPath := args[0]
	ext := filepath.Ext(inPath)
	if ext != "" {
		ext = ext[1:]
	}
	log.Printf("ext = %s", ext)

	f, err := os.Open(inPath)
	must(err)

	stats, err := f.Stat()
	must(err)

	is, err := sz.NewInStream(f, ext, stats.Size())
	must(err)
	log.Printf("Created input stream (%s, %d bytes)...", inPath, stats.Size())

	a, err := lib.OpenArchive(is)
	must(err)
	log.Printf("Opened archive...")
	defer lib.Free()

	log.Printf("Archive has %d items", a.GetItemCount())

	for i := int64(0); i < a.GetItemCount(); i++ {
		func() {
			item := a.GetItem(i)
			if item == nil {
				must(errors.New("null item :("))
			}
			defer item.Free()

			outPath := filepath.ToSlash(item.GetStringProperty(sz.PidPath))
			absoluteOutPath := filepath.Join("out", outPath)

			log.Printf("out      = '%s'", outPath)
			for i := 0; i < len(outPath); i++ {
				log.Printf("out[%d] = %0x ", i, outPath[i])
			}

			if item.GetBoolProperty(sz.PidIsDir) {
				err := os.MkdirAll(absoluteOutPath, 0755)
				must(err)
				return
			}

			dirPath := filepath.Dir(absoluteOutPath)
			must(os.MkdirAll(dirPath, 0755))

			log.Printf("Extracting %s (%s compressed, %s uncompressed)...",
				outPath,
				humanize.IBytes(item.GetUInt64Property(sz.PidPackSize)),
				humanize.IBytes(item.GetUInt64Property(sz.PidSize)),
			)

			of, err := os.Create(absoluteOutPath)
			must(err)
			defer of.Close()

			os, err := sz.NewOutStream(of)
			must(err)
			defer os.Free()

			err = a.Extract(item, os)
			must(err)
		}()
	}
}

func must(err error) {
	if err != nil {
		log.Panic(err)
	}
}

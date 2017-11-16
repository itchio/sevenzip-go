package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"

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
			of, err := os.Create(filepath.Join("out", fmt.Sprintf("item%d.dat", i)))
			must(err)

			os, err := sz.NewOutStream(of)
			must(err)
			log.Printf("Created output stream...")
			defer os.Free()

			item := a.GetItem(i)
			if item == nil {
				must(errors.New("null item :("))
			}
			defer item.Free()

			log.Printf("Extracting item %d...", i)
			err = a.Extract(item, os)
			must(err)
			log.Printf("Done!")
		}()
	}
}

func must(err error) {
	if err != nil {
		log.Panic(err)
	}
}

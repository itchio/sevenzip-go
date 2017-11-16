package main

import (
	"errors"
	"log"
	"os"

	"github.com/fasterthanlime/go-libc7zip/sz"
)

func main() {
	lib, err := sz.NewLib()
	must(err)
	log.Printf("Initialized library...")

	f, err := os.Open("main.7z")
	must(err)

	stats, err := f.Stat()
	must(err)

	is, err := sz.NewInStream(f, ".7z", stats.Size())
	must(err)
	log.Printf("Created input stream...")

	a, err := lib.OpenArchive(is)
	must(err)
	log.Printf("Opened archive...")

	log.Printf("Archive has %d items", a.GetItemCount())

	log.Printf("trying to extract last file...")

	of, err := os.Create("out.dat")
	must(err)

	os, err := sz.NewOutStream(of)
	must(err)
	log.Printf("Created output stream...")

	item := a.GetItem(a.GetItemCount() - 1)
	if item == nil {
		must(errors.New("null item :("))
	}

	log.Printf("Extracting...")
	err = a.Extract(item, os)
	must(err)
	log.Printf("Done!")
}

func must(err error) {
	if err != nil {
		log.Panic(err)
	}
}

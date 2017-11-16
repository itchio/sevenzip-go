package main

import (
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
}

func must(err error) {
	if err != nil {
		log.Panic(err)
	}
}

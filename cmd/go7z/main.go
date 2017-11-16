package main

import (
	"log"

	"github.com/fasterthanlime/go-libc7zip/sz"
)

func main() {
	err := sz.Initialize()
	if err != nil {
		log.Panic(err)
	}

	log.Printf("Alles gut!")
}

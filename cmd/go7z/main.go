package main

import (
	"errors"
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	"os"
	"path/filepath"
	"strings"

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

	itemCount, err := a.GetItemCount()
	must(err)
	log.Printf("Archive has %d items", err)

	is.Stats = &sz.ReadStats{}

	for i := int64(0); i < itemCount; i++ {
		for j := 0; j < 2; j++ {
			is.Stats.RecordRead(0, 0)
		}

		func() {
			item := a.GetItem(i)
			if item == nil {
				must(errors.New("null item :("))
			}
			defer item.Free()

			outPath := filepath.ToSlash(item.GetStringProperty(sz.PidPath))
			// Remove illegal character for windows paths, see
			// https://msdn.microsoft.com/en-us/library/windows/desktop/aa365247(v=vs.85).aspx
			for i := byte(0); i <= 31; i++ {
				outPath = strings.Replace(outPath, string([]byte{i}), "_", -1)
			}

			absoluteOutPath := filepath.Join("out", outPath)

			// log.Printf("out      = '%s'", outPath)
			// for i := 0; i < len(outPath); i++ {
			// 	log.Printf("out[%d] = %0x ", i, outPath[i])
			// }

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

			os, err := sz.NewOutStream(of)
			must(err)
			defer os.Close()
			defer os.Free()

			err = a.Extract(item, os)
			must(err)
		}()
	}

	width := len(is.Stats.Reads)
	height := 800
	log.Printf("Making %dx%d image", width, height)

	rect := image.Rect(0, 0, width, height)
	img := image.NewRGBA(rect)

	black := &color.RGBA{
		R: 0,
		G: 0,
		B: 0,
		A: 255,
	}
	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			img.Set(x, y, black)
		}
	}

	scale := 1.0 / float64(stats.Size()) * float64(height)
	c := &color.RGBA{
		R: 255,
		G: 0,
		B: 0,
		A: 255,
	}

	var maxReadSize int64 = 1
	for _, op := range is.Stats.Reads {
		if op.Size > maxReadSize {
			maxReadSize = op.Size
		}
	}

	for x, op := range is.Stats.Reads {
		ymin := int(math.Floor(float64(op.Offset) * scale))
		ymax := int(math.Ceil(float64(op.Offset+op.Size) * scale))

		cd := *c
		cd.G = uint8(float64(op.Size) / float64(maxReadSize) * 255)

		for y := ymin; y <= ymax; y++ {
			img.Set(x, y, &cd)
		}
	}

	imageFile, err := os.Create("out/reads.png")
	must(err)
	defer imageFile.Close()

	err = png.Encode(imageFile, img)
	must(err)
}

func must(err error) {
	if err != nil {
		log.Panic(err)
	}
}

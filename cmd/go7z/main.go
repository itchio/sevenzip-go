package main

import (
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
	"github.com/go-errors/errors"
)

type ecs struct {
	// muffin
}

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
	log.Printf("Archive has %d items", itemCount)

	is.Stats = &sz.ReadStats{}

	ec, err := sz.NewExtractCallback(&ecs{})
	must(err)
	defer ec.Free()

	var indices = make([]int64, itemCount)
	for i := 0; i < int(itemCount); i++ {
		indices[i] = int64(i)
	}

	err = a.ExtractSeveral(indices, ec)
	must(err)

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

func (e *ecs) GetStream(item *sz.Item) (*sz.OutStream, error) {
	outPath := filepath.ToSlash(item.GetStringProperty(sz.PidPath))
	// Remove illegal character for windows paths, see
	// https://msdn.microsoft.com/en-us/library/windows/desktop/aa365247(v=vs.85).aspx
	for i := byte(0); i <= 31; i++ {
		outPath = strings.Replace(outPath, string([]byte{i}), "_", -1)
	}

	log.Printf("Extracting %s", outPath)

	absoluteOutPath := filepath.Join("out", outPath)
	of, err := os.Create(absoluteOutPath)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	os, err := sz.NewOutStream(of)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	return os, nil
}

func (e *ecs) SetProgress(complete int64, total int64) {
	log.Printf("Progress: %s / %s",
		humanize.IBytes(uint64(complete)),
		humanize.IBytes(uint64(total)),
	)
}

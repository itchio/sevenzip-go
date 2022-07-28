package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/itchio/headway/united"
	"github.com/itchio/sevenzip-go/sz"
	"github.com/pkg/errors"
)

type ecs struct {
	// muffin
}

type mcs struct {
	FirstName  string
	CurVolName string
	f          *os.File
}

//2022/07/28 03:45:46 mcs GetFirstVolumeName()
//2022/07/28 03:45:46 mcs MoveToVolume(test-magic.7z.001)
//2022/07/28 03:45:46 mcs MoveToVolume(test-magic.7z.001)
//2022/07/28 03:45:46 mcs OpenCurrentVolumeStream()
//2022/07/28 03:45:46 mcs GetCurrentVolumeSize()
//2022/07/28 03:45:46 mcs GetFirstVolumeName()
//2022/07/28 03:45:46 mcs MoveToVolume(test-magic.7z.001)
//2022/07/28 03:45:46 mcs MoveToVolume(test-magic.7z.002)
//2022/07/28 03:45:46 mcs OpenCurrentVolumeStream()
//2022/07/28 03:45:46 mcs GetCurrentVolumeSize()
//2022/07/28 03:45:46 mcs MoveToVolume(test-magic.7z.003)
//2022/07/28 03:45:46 mcs OpenCurrentVolumeStream()
//2022/07/28 03:45:46 mcs GetCurrentVolumeSize()
//2022/07/28 03:45:46 mcs MoveToVolume(test-magic.7z.004)

func (m *mcs) GetFirstVolumeName() string {
	log.Printf("mcs GetFirstVolumeName()")
	m.MoveToVolume(m.FirstName)
	return m.FirstName
}

func (m *mcs) MoveToVolume(volumeName string) error {
	log.Printf("mcs MoveToVolume(%s)", volumeName)
	var err error
	m.f.Close()
	m.f, err = os.OpenFile(volumeName, os.O_RDONLY, 0)
	if err != nil {
		log.Printf("mcs MoveToVolume(%s), error: %v", volumeName, err)
		return err
	}
	m.CurVolName = volumeName
	return nil
}

func (m *mcs) GetCurrentVolumeSize() uint64 {
	log.Printf("mcs GetCurrentVolumeSize()")
	info, err := m.f.Stat()
	if err != nil {
		return 0
	}
	return uint64(info.Size())
}

func (m *mcs) OpenCurrentVolumeStream() (*sz.InStream, error) {
	log.Printf("mcs OpenCurrentVolumeStream()")
	ext := filepath.Ext(m.CurVolName)
	if ext != "" {
		ext = ext[1:]
	}

	f, err := os.OpenFile(m.CurVolName, os.O_RDONLY, 0)
	if err != nil {
		return nil, err
	}
	return sz.NewInStream(f, ext, int64(m.GetCurrentVolumeSize()))
}

func main() {
	lib, err := sz.NewLib()
	must(err)
	log.Printf("Initialized 7-zip %s...", lib.GetVersion())
	defer lib.Free()

	args := os.Args[1:]

	if len(args) < 1 {
		log.Printf("Usage: go7z_multi FIRST_ARCHIVE [PASSWORD]")
		os.Exit(1)
	}

	password := ""
	if len(args) >= 2 {
		password = args[1]
	}

	inPath := args[0]
	ext := filepath.Ext(inPath)
	if ext != "" {
		ext = ext[1:]
	}
	log.Printf("ext = %s", ext)

	f, err := os.Open(inPath)
	must(err)

	f.Close()

	//is, err := sz.NewInStream(f, ext, stats.Size())
	//must(err)
	//log.Printf("Created input stream (%s, %d bytes)...", inPath, stats.Size())
	//
	//is.Stats = &sz.ReadStats{}
	s, err := sz.NewMultiVolumeCallback(&mcs{FirstName: inPath})
	log.Printf("Created input stream (%s, %d bytes)...", inPath)

	a, err := lib.OpenMultiVolumeArchive(s, password, false)
	must(err)

	log.Printf("Opened archive: format is (%s)", a.GetArchiveFormat())

	itemCount, err := a.GetItemCount()
	must(err)
	log.Printf("Archive has %d items", itemCount)

	ec, err := sz.NewExtractCallback(&ecs{})
	must(err)
	defer ec.Free()

	var indices = make([]int64, itemCount)
	for i := 0; i < int(itemCount); i++ {
		indices[i] = int64(i)
	}

	err = a.ExtractSeveral(indices, ec)
	must(err)

	errs := ec.Errors()
	if len(errs) > 0 {
		log.Printf("There were %d errors during extraction:", len(errs))
		for _, err := range errs {
			log.Printf("- %s", err.Error())
		}
	}
}

func must(err error) {
	if err != nil {
		log.Panic(err)
	}
}

func (e *ecs) GetStream(item *sz.Item) (*sz.OutStream, error) {
	propPath, ok := item.GetStringProperty(sz.PidPath)
	if !ok {
		return nil, errors.New("could not get item path")
	}

	outPath := filepath.ToSlash(propPath)
	// Remove illegal character for windows paths, see
	// https://msdn.microsoft.com/en-us/library/windows/desktop/aa365247(v=vs.85).aspx
	for i := byte(0); i <= 31; i++ {
		outPath = strings.Replace(outPath, string([]byte{i}), "_", -1)
	}

	absoluteOutPath := filepath.Join("out", outPath)

	log.Printf("  ")
	log.Printf("==> Extracting %d: %s", item.GetArchiveIndex(), outPath)

	if attrib, ok := item.GetUInt64Property(sz.PidAttrib); ok {
		log.Printf("==> Attrib       %08x", attrib)
	}
	if attrib, ok := item.GetUInt64Property(sz.PidPosixAttrib); ok {
		log.Printf("==> Posix Attrib %08x", attrib)
	}
	if symlink, ok := item.GetStringProperty(sz.PidSymLink); ok {
		log.Printf("==> Symlink dest: %s", symlink)
	}

	isDir, _ := item.GetBoolProperty(sz.PidIsDir)
	if isDir {
		log.Printf("Making %s", outPath)

		err := os.MkdirAll(absoluteOutPath, 0755)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		// is a dir, just skip it
		return nil, nil
	}

	err := os.MkdirAll(filepath.Dir(absoluteOutPath), 0755)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	of, err := os.Create(absoluteOutPath)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	os, err := sz.NewOutStream(of)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return os, nil
}

func (e *ecs) SetProgress(complete int64, total int64) {
	log.Printf("Progress: %s / %s",
		united.FormatBytes(complete),
		united.FormatBytes(total),
	)
}

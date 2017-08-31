package idg

import (
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

const (
	HeaderAcceptRanges = "Accept-Ranges"
	DefaultParts       = 8
)

type FileDler interface {
	StartDownload() error
	StopDownload()
}

type File struct {
	Name        string
	Size        int64
	Dir         string
	RemoteURL   string
	AcceptRange bool
	Wait        *sync.WaitGroup
	FileParts   FileParts
}

func NewFile(remoteURL string) (*File, error) {
	var err error
	file := &File{
		RemoteURL: remoteURL,
		Dir:       ".",
		Wait:      &sync.WaitGroup{},
		FileParts: FileParts{},
	}

	res, err := http.DefaultClient.Get(remoteURL)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 200 {
		return nil, ErrFilePermission
	}

	if _, ok := res.Header[HeaderAcceptRanges]; ok {
		file.AcceptRange = true
	}

	file.Size = res.ContentLength
	file.Name, err = GetFileName(res)

	return file, err
}

func (file *File) StartDownload() error {

	var parts int64
	parts = DefaultParts

	if !file.AcceptRange || file.Size <= 0 {
		parts = 1
	}

	modBytes := file.Size % parts
	rangeBytes := file.Size / parts
	var lastBytes int64

	for part := int64(0); part < parts; part++ {
		startByte := rangeBytes * part
		endByte := startByte + rangeBytes
		if startByte > 0 {
			startByte++
		}
		filePart := NewPart(file, part+1, startByte, endByte)
		file.FileParts = append(file.FileParts, filePart)
		filePart.startDownload()
		lastBytes = endByte
	}

	if modBytes > 0 {
		filePart := NewPart(file, parts+1, lastBytes, file.Size)
		file.FileParts = append(file.FileParts, filePart)
		filePart.startDownload()
	}

	file.Wait.Wait()

	return file.join()
}

func (file *File) join() error {

	path := file.Dir + "/" + file.Name
	fileWriter, err := os.Create(path)
	if err != nil {
		log.Fatal(err)
		return err
	}

	sort.Sort(file.FileParts)
	for _, part := range file.FileParts {
		reader, err := os.Open(part.getPath())
		if err != nil {
			log.Fatal(err)
			return err
		}
		if _, err := io.Copy(fileWriter, reader); err != nil {
			return err
		}
		os.Remove(part.getPath())
	}

	return nil
}

func GetFilePartName(fileName string, part int64) string {
	return fmt.Sprintf("%s.part.%d", fileName, part)
}

func GetFileName(resp *http.Response) (string, error) {
	filename := resp.Request.URL.Path
	if cd := resp.Header.Get("Content-Disposition"); cd != "" {
		if _, params, err := mime.ParseMediaType(cd); err == nil {
			filename = params["filename"]
		}
	}

	// sanitize
	if filename == "" || strings.HasSuffix(filename, "/") || strings.Contains(filename, "\x00") {
		return "", ErrNoFilename
	}

	filename = filepath.Base(path.Clean("/" + filename))
	if filename == "" || filename == "." || filename == "/" {
		return "", ErrNoFilename
	}

	return filename, nil
}

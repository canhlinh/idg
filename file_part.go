package idg

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

type FilePartResult struct {
}

type FilePart struct {
	File       *File
	PartNumber int64
	StartByte  int64
	EndByte    int64
	FileWriter io.WriteCloser
	quit       chan bool
}

type FileParts []*FilePart

func NewPart(file *File, partNumber, fromByte, toByte int64) *FilePart {
	return &FilePart{
		File:       file,
		PartNumber: partNumber,
		StartByte:  fromByte,
		EndByte:    toByte,
		quit:       make(chan bool, 1),
	}
}

func (part *FilePart) startDownload() error {
	part.File.wait.Add(1)
	go func() {
		defer part.File.wait.Done()

		req, _ := http.NewRequest(http.MethodGet, part.File.RemoteURL, nil)
		req.Header.Add("Range", fmt.Sprintf("bytes=%d-%d", part.StartByte, part.EndByte))
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Println(err)
			return
		}

		if res.StatusCode != 200 && res.StatusCode != 206 {
			return
		}

		fileWriter, err := os.Create(part.getPath())
		if err != nil {
			log.Println(err)
			return
		}
		defer res.Body.Close()
		part.FileWriter = fileWriter
		io.Copy(fileWriter, res.Body)
		fileWriter.Close()
	}()

	return nil
}

func (part *FilePart) getPath() string {
	return fmt.Sprintf("%s/%s.part.%d", part.File.Dir, part.File.Name, part.PartNumber)
}

func (part *FilePart) stopDownload() {
	part.FileWriter.Close()
}

func (s FileParts) Len() int {
	return len(s)
}
func (s FileParts) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s FileParts) Less(i, j int) bool {
	return s[i].PartNumber < s[j].PartNumber
}
package idg

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const (
	ReTriesDownload = 10
)

type FilePartResult struct {
}

type FilePart struct {
	File       *File
	PartNumber int64
	StartByte  int64
	EndByte    int64
	path       string
	FileWriter io.WriteCloser
	quit       chan bool
	attempt    int
}

type FileParts []*FilePart

func NewPart(file *File, partNumber, fromByte, toByte int64) *FilePart {
	return &FilePart{
		File:       file,
		PartNumber: partNumber,
		StartByte:  fromByte,
		EndByte:    toByte,
		path:       fmt.Sprintf("%s/%s.part.%d", file.dir, file.Name, partNumber),
		quit:       make(chan bool, 1),
		attempt:    0,
	}
}

func (part *FilePart) startDownload() error {
	part.File.wait.Add(1)
	go func() {
		defer func() {
			part.File.wait.Done()
		}()

	TRY_DOWNLOAD:
		part.attempt++

		req, _ := http.NewRequest(http.MethodGet, part.File.RemoteURL, nil)
		if part.File.header != nil {
			for key, value := range part.File.header {
				req.Header.Add(key, value)
			}
		}

		if part.File.maxPart > 1 {
			req.Header.Add("Range", fmt.Sprintf("bytes=%d-%d", part.StartByte, part.EndByte))
		}
		for _, cookie := range part.File.Cookies {
			req.AddCookie(cookie)
		}

		part.File.mutex.Lock()
		res, err := http.DefaultTransport.RoundTrip(req)
		if err != nil {
			part.File.mutex.Unlock()
			if part.attempt > ReTriesDownload {
				part.File.errorListener <- err
				return
			}
			time.Sleep(3 * time.Second)
			goto TRY_DOWNLOAD
		}
		part.File.mutex.Unlock()

		if res.StatusCode != 200 && res.StatusCode != 206 {
			goto TRY_DOWNLOAD
		}

		fileWriter, err := os.Create(part.path)
		if err != nil {
			if part.attempt > ReTriesDownload {
				part.File.errorListener <- err
				return
			}
			time.Sleep(3 * time.Second)
			goto TRY_DOWNLOAD
		}
		defer res.Body.Close()
		defer fileWriter.Close()

		part.FileWriter = fileWriter
		if err := part.copyBuffer(fileWriter, res.Body); err != nil {
			part.File.errorListener <- err
			return
		}
	}()

	return nil
}

func (part *FilePart) stopDownload() {
	part.FileWriter.Close()
}

func (part *FilePart) copyBuffer(dst io.Writer, src io.Reader) (err error) {
	// If the reader has a WriteTo method, use it to do the copy.
	// Avoids an allocation and a copy.
	if wt, ok := src.(io.WriterTo); ok {
		wt.WriteTo(dst)
		return
	}
	// Similarly, if the writer has a ReadFrom method, use it to do the copy.
	if rt, ok := dst.(io.ReaderFrom); ok {
		rt.ReadFrom(src)
		return
	}

	buf := make([]byte, 32*1024)

	for {
		nr, er := src.Read(buf)
		if nr > 0 {
			nw, ew := dst.Write(buf[0:nr])
			if ew != nil {
				err = ew
				break
			}
			if nr != nw {
				err = io.ErrShortWrite
				break
			}
			part.File.ProgressHandler <- nw
		}
		if er != nil {
			if er != io.EOF {
				err = er
			}
			break
		}
	}
	return err
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

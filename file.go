package idg

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	pb "gopkg.in/cheggaaa/pb.v1"
)

const (
	HeaderAcceptRanges = "Accept-Ranges"
	DefaultParts       = 8
	DefaultDir         = "."
)

type FileDler interface {
	StartDownload() error
	StopDownload()
}

type File struct {
	Name             string
	Size             int64
	dir              string
	path             string
	RemoteURL        string
	AcceptRange      bool
	wait             *sync.WaitGroup
	FileParts        FileParts
	DownloadedBytes  int64
	ProgressHandler  chan int
	SupportMultiPart bool
	Cookies          []*http.Cookie
	MaxPart          int64
}

func NewFile(remoteURL string, cookies ...*http.Cookie) (*File, error) {
	var err error
	file := &File{
		RemoteURL:       remoteURL,
		wait:            &sync.WaitGroup{},
		FileParts:       FileParts{},
		ProgressHandler: make(chan int, DefaultParts),
		Cookies:         cookies,
		MaxPart:         DefaultParts,
	}

	req, _ := http.NewRequest(http.MethodGet, remoteURL, nil)

	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return nil, ErrFilePermission
	}

	if _, ok := res.Header[HeaderAcceptRanges]; ok {
		file.AcceptRange = true
	}

	file.Size = res.ContentLength
	file.Name, err = GetFileName(res)
	file.SetDir(DefaultDir)

	if file.AcceptRange && file.Size > 0 {
		file.SupportMultiPart = true
	}

	return file, err
}

func (file *File) SetDir(dir string) {
	os.MkdirAll(dir, os.ModePerm)
	file.dir = dir
	file.path = dir + "/" + file.Name
}

func (file *File) StartDownload() error {
	if file.SupportMultiPart {

		rangeBytes := file.Size / file.MaxPart
		var lastBytes int64

		for part := int64(0); part < file.MaxPart; part++ {
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

		if lastBytes < file.Size {
			filePart := NewPart(file, file.MaxPart+1, lastBytes+1, file.Size)
			file.FileParts = append(file.FileParts, filePart)
			filePart.startDownload()
		}
	} else {
		filePart := NewPart(file, 1, 0, file.Size)
		file.FileParts = append(file.FileParts, filePart)
		filePart.startDownload()
	}

	file.monitor()
	file.Wait()
	close(file.ProgressHandler)
	return file.join()
}

func (file *File) monitor() {
	go func() {
		bar := pb.New(int(file.Size)).SetUnits(pb.U_BYTES)
		bar.ShowSpeed = true
		bar.ShowFinalTime = false
		if !file.SupportMultiPart {
			bar.ShowPercent = false
			bar.ShowBar = false
		}
		bar.Start()
		defer bar.Finish()
		for {
			select {
			case nw, ok := <-file.ProgressHandler:
				if !ok {
					return
				}
				bar.Add(nw)
				file.DownloadedBytes += int64(nw)
			}
		}

	}()
}

func (file *File) Wait() {
	file.wait.Wait()
}

func (file *File) join() error {

	fileWriter, err := os.Create(file.path)
	if err != nil {
		fmt.Println(err)
		return err
	}

	sort.Sort(file.FileParts)
	for _, part := range file.FileParts {
		reader, err := os.Open(part.path)
		if err != nil {
			fmt.Println(err)
			return err
		}
		if _, err := io.Copy(fileWriter, reader); err != nil {
			return err
		}
		os.Remove(part.path)
	}

	fileWriter.Close()
	return nil
}

func (file *File) GetPath() string {
	return file.path
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

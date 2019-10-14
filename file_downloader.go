package idg

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"gopkg.in/cheggaaa/pb.v1"
)

type PartReport struct {
	FilePart       *FilePart
	DownloadedPath string
	Err            error
}

// FileDownloader a worker for downloading a file
type FileDownloader struct {
	Client          *http.Client
	File            *File
	Bar             *pb.ProgressBar
	MaxConnections  int
	Dir             string
	dispatcher      chan *FilePart
	partReport      chan *PartReport
	quitSignal      chan bool
	progressMonitor chan int
}

func NewFileDownloader(file *File, dir string, maxConnections int) *FileDownloader {
	return &FileDownloader{
		Client:         &http.Client{},
		File:           file,
		Dir:            dir,
		MaxConnections: maxConnections,
		Bar:            pb.New(0),
	}
}

// Download download a file
func (fileDownloader *FileDownloader) Download() (string, error) {
	if err := fileDownloader.parseFile(); err != nil {
		return "", nil
	}

	if !fileDownloader.File.AcceptRange {
		fileDownloader.MaxConnections = 1
	}

	fileDownloader.startDispatcher()
	doneSignal := fileDownloader.startDownloaders()
	downloadedParts := []*FilePart{}

LOOP:
	for {
		select {
		case <-doneSignal:
			close(fileDownloader.quitSignal)
			break LOOP
		case partReport := <-fileDownloader.partReport:
			if partReport.Err != nil {
				close(fileDownloader.quitSignal)
				return "", partReport.Err
			}

			downloadedParts = append(downloadedParts, partReport.FilePart)
		}
	}

	filePath, err := fileDownloader.join(downloadedParts)
	if err != nil {
		return "", nil
	}

	fileDownloader.File.DiskPath = filePath
	return filePath, nil
}

func (fileDownloader *FileDownloader) parseFile() error {
	if _, err := url.Parse(fileDownloader.File.URL); err != nil {
		return err
	}

	req, _ := http.NewRequest(http.MethodGet, fileDownloader.File.URL, nil)
	for key, value := range fileDownloader.File.Header {
		req.Header.Add(key, value)
	}

	if len(fileDownloader.File.Cookies) > 0 {
		fileDownloader.Client.Jar.SetCookies(req.URL, fileDownloader.File.Cookies)
	}

	res, err := fileDownloader.Client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return errors.New(res.Status)
	}

	fileDownloader.File.Name = getFileName(res)
	fileDownloader.File.AcceptRange = isAcceptByteRange(res)
	fileDownloader.File.Size = res.ContentLength

	if fileDownloader.File.Size == 0 {
		return errors.New("Failed to get the file's size")
	}

	return nil
}

func (fileDownloader *FileDownloader) downloadPart(part *FilePart) *PartReport {
	partName := fmt.Sprintf("%s_%d", fileDownloader.File.Name, part.PartNo)
	partFilePath := filepath.Join(fileDownloader.Dir, partName)
	partReport := &PartReport{FilePart: part}

	req, _ := http.NewRequest(http.MethodGet, fileDownloader.File.URL, nil)
	for key, value := range fileDownloader.File.Header {
		req.Header.Add(key, value)
	}
	if fileDownloader.File.AcceptRange {
		req.Header.Add("Range", fmt.Sprintf("bytes=%d-%d", part.Begin, part.End))
	}

	res, err := fileDownloader.Client.Do(req)
	if err != nil {
		partReport.Err = err
		return partReport
	}

	if res.StatusCode != 200 {
		if err != nil {
			partReport.Err = err
			return partReport
		}
	}

	diskfile, err := os.Create(partFilePath)
	if err != nil {
		partReport.Err = err
		return partReport
	}

	if err := fileDownloader.copyBuffer(diskfile, res.Body); err != nil {
		partReport.Err = err
		return partReport
	}

	partReport.FilePart.DiskPath = partFilePath
	return partReport
}

func (fileDownloader *FileDownloader) startDownloaders() chan bool {
	wg := &sync.WaitGroup{}
	wg.Add(fileDownloader.MaxConnections)

	for i := 0; i < fileDownloader.MaxConnections; i++ {
		go func() {
			defer wg.Done()

			for {
				select {
				case part, ok := <-fileDownloader.dispatcher:
					if !ok {
						// all parts are done
						return
					}

					fileDownloader.partReport <- fileDownloader.downloadPart(part)
				case <-fileDownloader.quitSignal:
					return
				}
			}
		}()
	}

	return wait(wg)
}

func (fileDownloader *FileDownloader) startDispatcher() {
	fileDownloader.monitor()

	fileDownloader.dispatcher = make(chan *FilePart, fileDownloader.MaxConnections)
	fileDownloader.quitSignal = make(chan bool)
	fileDownloader.partReport = make(chan *PartReport, fileDownloader.MaxConnections)

	go func() {
		defer close(fileDownloader.dispatcher)

		rangeBytes := fileDownloader.File.Size / int64(fileDownloader.MaxConnections)
		for i := 0; i < fileDownloader.MaxConnections; i++ {
			begin := rangeBytes * int64(i)
			end := begin + rangeBytes

			if begin > 0 {
				begin = begin + 1
			}

			if i == fileDownloader.MaxConnections-1 {
				end = fileDownloader.File.Size
			}

			filePart := &FilePart{
				PartNo: int64(i),
				Begin:  begin,
				End:    end,
			}

			select {
			case fileDownloader.dispatcher <- filePart:
			case <-fileDownloader.quitSignal:
				return
			}
		}
	}()
}

func (fileDownloader *FileDownloader) join(parts []*FilePart) (string, error) {
	filepath := filepath.Join(fileDownloader.Dir, fileDownloader.File.Name)

	file, err := os.Create(filepath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	sort.Slice(parts, func(i, j int) bool {
		return parts[i].PartNo < parts[j].PartNo
	})

	for _, part := range parts {

		bytes, err := getFileBytes(part.DiskPath)
		if err != nil {
			return "", err
		}

		if _, err := file.Write(bytes); err != nil {
			return "", err
		}

		if err := os.RemoveAll(part.DiskPath); err != nil {
			return "", err
		}
	}

	return filepath, nil
}

func (fileDownloader *FileDownloader) monitor() {
	fileDownloader.Bar.SetTotal64(fileDownloader.File.Size)
	fileDownloader.Bar.SetUnits(pb.U_BYTES)
	fileDownloader.Bar.Start()
	fileDownloader.Bar.ShowSpeed = true
	fileDownloader.Bar.ShowElapsedTime = true

	fileDownloader.progressMonitor = make(chan int, 1024)

	go func() {
		for {
			select {
			case bytes := <-fileDownloader.progressMonitor:
				fileDownloader.Bar.Add(bytes)
			case <-fileDownloader.quitSignal:
				fileDownloader.Bar.Finish()
				return
			}
		}
	}()
}

func (fileDownloader *FileDownloader) copyBuffer(dst io.Writer, src io.Reader) (err error) {
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
			fileDownloader.progressMonitor <- nw
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

func DownloadSingleFile(file *File, dir string, maxConnections int) (string, error) {
	return NewFileDownloader(file, dir, maxConnections).Download()
}

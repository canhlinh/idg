package idg

import (
	"net/http"
)

const (
	DefaultParts = 8
	DefaultDir   = "."
)

// File present a file
type File struct {
	Name        string
	Size        int64
	URL         string
	AcceptRange bool
	Header      map[string]string
	Cookies     []*http.Cookie
	DiskPath    string
}

// FilePart present a part of a File
type FilePart struct {
	PartNo   int64
	Begin    int64
	End      int64
	DiskPath string
}

// NewFile creates a new file
func NewFile(remoteURL string, cookies []*http.Cookie, header map[string]string) *File {

	return &File{
		URL:     remoteURL,
		Cookies: cookies,
		Header:  header,
	}
}

package idg

import "errors"

var (
	ErrNoFilename     = errors.New("no filename could be determined")
	ErrFilePermission = errors.New("could not download file")
)

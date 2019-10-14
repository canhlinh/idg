package main

import (
	"flag"
	"log"
	"net/url"

	"github.com/canhlinh/idg"
)

var (
	downloadUrl string
)

func init() {
	flag.StringVar(&downloadUrl, "url", "", "Download url")
	flag.Parse()

	if _, err := url.Parse(downloadUrl); err != nil {
		log.Fatal(err)
	}
}

func main() {
	file := idg.NewFile(downloadUrl, nil, nil)
	if _, err := idg.DownloadSingleFile(file, "./", 4); err != nil {
		panic(err)
	}
}

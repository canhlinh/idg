package main

import (
	"flag"
	"log"
	"net/url"
	"time"

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
	file, err := idg.NewFile(downloadUrl, nil, nil)
	file.SetPart(20)
	if err != nil {
		log.Fatal(err)
	}

	if err := file.StartDownload(); err != nil {
		log.Fatal(err)
	}
	time.Sleep(time.Second)
}

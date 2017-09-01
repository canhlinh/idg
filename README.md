# idg
Fast file download implementation in Golang.

### Run uni test
```
Make test
```

### Example
```
	file, err := idg.NewFile("https://storage.googleapis.com/golang/go1.9.linux-amd64.tar.gz")
	if err != nil {
		log.Fatal(err)
	}

	if err := file.StartDownload(); err != nil {
		log.Fatal(err)
	}
	time.Sleep(time.Second)
```

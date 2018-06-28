package idg

import (
	"io/ioutil"
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	TestRemoteURL = "http://localhost:8080/small.mp4"
	TestFileName  = "small.mp4"
	TestFileMd5   = "a3ac7ddabb263c2d00b73e8177d15c8d"
)

func TestMain(m *testing.M) {
	go func() {
		http.Handle("/", http.FileServer(http.Dir("./sample")))
		http.ListenAndServe(":8080", nil)
	}()
	m.Run()
}

func TestNewFile(t *testing.T) {

	file, err := NewFile(TestRemoteURL, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, TestRemoteURL, file.RemoteURL)
	assert.Equal(t, ".", file.dir)
	assert.NotNil(t, file.Wait)
	assert.NotNil(t, file.FileParts)
	assert.True(t, file.Size > 0)
	assert.Equal(t, TestFileName, file.Name)
	assert.Nil(t, file.header)

	header := map[string]string{"a": "a", "b": "b"}
	file2, _ := NewFile(TestRemoteURL, nil, header)
	assert.Equal(t, header, file2.header)
}

func TestStartDownload(t *testing.T) {
	file, err := NewFile(TestRemoteURL, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	if err := file.StartDownload(); err != nil {
		t.Fatal(err)
	}

	fileReader, err := os.Open(file.GetPath())
	if err != nil {
		t.Fatal(err)
	}
	fileData, err := ioutil.ReadAll(fileReader)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(len(fileData))
	assert.Equal(t, TestFileMd5, HashMD5(fileData), "Md5 is not match")
	os.Remove(file.GetPath())
}

func TestStartDownloadWithCustomDir(t *testing.T) {
	file, err := NewFile(TestRemoteURL, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	file.SetDir("custom_dir")

	if err := file.StartDownload(); err != nil {
		t.Fatal(err)
	}

	fileReader, err := os.Open(file.GetPath())
	if err != nil {
		t.Fatal(err)
	}
	fileData, err := ioutil.ReadAll(fileReader)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, TestFileMd5, HashMD5(fileData), "Md5 is not match")
	os.RemoveAll(file.dir)
}

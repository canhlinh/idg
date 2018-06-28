package idg

import "testing"
import "github.com/stretchr/testify/assert"
import "os"
import "io/ioutil"

func TestNewPart(t *testing.T) {
	file, err := NewFile(TestRemoteURL, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	part := NewPart(file, 1, 0, 1024)
	assert.Equal(t, file, part.File)
	assert.Equal(t, int64(1), part.PartNumber)
	assert.Equal(t, int64(0), part.StartByte)
	assert.Equal(t, int64(1024), part.EndByte)
	assert.NotNil(t, part.quit)
}

func TestStartDownloadPart(t *testing.T) {
	file, err := NewFile(TestRemoteURL, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	part := NewPart(file, 1, 0, 1024)
	if err := part.startDownload(); err != nil {
		t.Fatal(err)
	}
	file.monitor()
	file.Wait()

	reader, err := os.Open(part.path)
	if err != nil {
		t.Fatal(err)
	}
	partData, err := ioutil.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(part.path)
	assert.Equal(t, len(partData), 1025)
}

func TestStartDownloadPartDie(t *testing.T) {
	file, err := NewFile(TestRemoteURL, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	part1 := NewPart(file, 2, 0, 1024/2)
	if err := part1.startDownload(); err != nil {
		t.Fatal(err)
	}

	part2 := NewPart(file, 2, part1.StartByte+1, 1024)
	if err := part2.startDownload(); err != nil {
		t.Fatal(err)
	}
	file.monitor()
	file.Wait()
}

func TestGetPathOfPart(t *testing.T) {
	file, err := NewFile(TestRemoteURL, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	part := NewPart(file, 1, 0, 1024)
	assert.Equal(t, "./small.mp4.part.1", part.path)
}

package pelican

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"

	"compress/gzip"
)

// ReadGzippedFile reads from path gzipped, returning
// the uncompressed bytes.
func ReadGzippedFile(gzipped string) ([]byte, error) {
	f, err := os.Open(gzipped)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		panic(err)
	}
	defer gz.Close()

	bytes, err := ioutil.ReadAll(gz)
	if err != nil {
		panic(err)
	}

	return bytes, err
}

// UnGzipFile reads the file gzipped into memory, and
// the writes it back out to disk in file ungzipped without
// the compression.
func UnGzipFile(gzipped, ungzipped string) error {

	bytes, err := ReadGzippedFile(gzipped)
	if err != nil {
		panic(err)
	}

	err = ioutil.WriteFile(ungzipped, bytes, 0600)
	if err != nil {
		panic(err)
	}

	return err
}

// GzipFile reads the file in path ungzipped, then
// writes it back out to path gzipped in compressed
// format.
func GzipFile(ungzipped, gzipped string) error {

	by, err := ioutil.ReadFile(ungzipped)
	if err != nil {
		return err
	}

	err = WriteGzippedFile(by, gzipped)
	return err
}

// WriteGzippedFile writes the bytes in by to the file named by gzipped.
func WriteGzippedFile(by []byte, gzipped string) error {
	fn := gzipped + ".temp"
	var file *os.File
	file, err := os.Create(fn)
	if err != nil {
		panic(fmt.Errorf("problem creating outfile '%s': %s", fn, err))
	}
	defer file.Close()

	gz := gzip.NewWriter(file)
	defer gz.Close()

	_, err = bytes.NewBuffer(by).WriteTo(gz)
	if err != nil {
		panic(err)
	}

	gz.Close()
	file.Close()
	exec.Command("mv", fn, gzipped).Run()

	return err
}

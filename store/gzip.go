package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"

	"compress/gzip"
)

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

func GzipFile(ungzipped, gzipped string) error {

	by, err := ioutil.ReadFile(ungzipped)
	if err != nil {
		return err
	}

	err = WriteGzippedFile(by, gzipped)
	return err
}

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

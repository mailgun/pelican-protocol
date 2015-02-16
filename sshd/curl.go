package main

import (
	"io/ioutil"
	"net/http"
)

func MyCurl(url string) string {
	response, err := http.Get(url)
	if err != nil {
		panicOn(err)
	} else {
		defer response.Body.Close()
		contents, err := ioutil.ReadAll(response.Body)
		panicOn(err)
		return string(contents)
	}
	return ""
}

package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
)

func MyCurl(addr string) string {
	response, err := http.Get(fmt.Sprintf("http://%s", addr))
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

package main

import (
	"fmt"
	"github.com/glycerine/ruid"
	"github.com/mailgun/pelican-protocol"
	"io/ioutil"
	"os"
)

func FetchOrGenSecretIdForService(path string) string {
	if FileExists(path) {
		return FetchSecretIdForService(path)
	} else {
		return GenSecretIdForService(path)
	}
}

func FetchSecretIdForService(path string) string {
	by, err := ioutil.ReadFile(path)
	panicOn(err)
	return string(by)
}

func GenSecretIdForService(path string) string {
	// generate this once and make it a unique RUID for each server's service.
	// usernames generated depend on this, so if you change the service Id,
	// you will change the user's account name
	myExternalIP := pelican.GetExternalIP()
	ruidGen := ruid.NewRuidGen(myExternalIP)
	id := ruidGen.Ruid2()
	fmt.Printf("SecretIdForService = '%s'\n", id)

	f, err := os.Create(path)
	panicOn(err)
	defer f.Close()
	f.WriteString(id)
	fmt.Fprintf(f, "\n")
	return id
}

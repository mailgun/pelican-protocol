package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/glycerine/ruid"
	"github.com/mailgun/pelican-protocol"
)

func FetchOrGenSecretIdForService(path string) string {
	if FileExists(path) {
		secret, err := FetchSecretIdForService(path)
		panicOn(err)
		return secret
	} else {
		return GenSecretIdForService(path)
	}
}

func FetchSecretIdForService(path string) (string, error) {
	by, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(by), nil
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

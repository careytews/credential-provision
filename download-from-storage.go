package main

import (
	"fmt"
	"io/ioutil"
	"os"
	//	"bytes"
)

func main() {

	if len(os.Args) != 4 {
		fmt.Fprintln(os.Stderr, "Usage:")
		fmt.Fprintln(os.Stderr,
			"  dowload-from-storage <key> <user> <file>")
		os.Exit(1)
	}

	// Get environment variables.
	keyfile := os.Args[1]

	// Read the key file
	key, err := ioutil.ReadFile(keyfile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Couldn't read key file: %s\n",
			err.Error())
		return
	}

	user := os.Args[2]

	filename := os.Args[3]

	svc, err := StorageSignin(key)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Couldn't connect: %s\n",
			err.Error())
		return
	}

	fmt.Fprintf(os.Stderr, "Connected.\n")

	bucket := Getenv("BUCKET", "")
	path := user + "/" + filename

	err = Download(svc, bucket, path, os.Stdout)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Couldn't download: %s\n",
			err.Error())
		return
	}

}

package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
)

func main() {

	if len(os.Args) != 5 {
		fmt.Println("Usage:")
		fmt.Println("  upload-to-storage <key> <user> <data-to-upload> <file>")
		os.Exit(1)
	}

	// Get environment variables.
	keyfile := os.Args[1]

	// Read the key file
	key, err := ioutil.ReadFile(keyfile)
	if err != nil {
		fmt.Printf("Couldn't read key file: %s\n",
			err.Error())
		return
	}

	user := os.Args[2]

	// Read the key file
	content, err := ioutil.ReadFile(os.Args[3])
	if err != nil {
		fmt.Printf("Couldn't read content file: %s\n",
			err.Error())
		return
	}

	filename := os.Args[4]

	svc, err := StorageSignin(key)
	if err != nil {
		fmt.Printf("Couldn't connect: %s\n",
			err.Error())
		return
	}

	fmt.Printf("Connected.\n")

	reader := bytes.NewReader(content)
	bucket := Getenv("BUCKET", "")
	path := user + "/" + filename

	err = Upload(svc, user, bucket, path, reader, -1)
	if err != nil {
		fmt.Printf("Couldn't upload: %s\n",
			err.Error())
		return
	}

}

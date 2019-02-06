package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"google.golang.org/api/storage/v1"
	//	"bytes"
)

func delete(svc *storage.Service, user, bucket, path string) error {

	err := svc.Objects.Delete(bucket, path).Do()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Couldn't Delete object: %s\n",
			err.Error())
		return err
	}

	return nil

}

func main() {

	if len(os.Args) != 4 {
		fmt.Fprintln(os.Stderr, "Usage:")
		fmt.Fprintln(os.Stderr,
			"  delete-from-storage <key> <user> <file>")
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

	err = delete(svc, user, bucket, path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Couldn't delete: %s\n",
			err.Error())
		return
	}

}

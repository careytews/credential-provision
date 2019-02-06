package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"google.golang.org/api/storage/v1"
)

func uploadCRL(svc *storage.Service, bucket string, destFile string, reader io.Reader) error {

	var object storage.Object
	object.Name = destFile
	object.Kind = "storage#object"
	object.CacheControl = "private, max-age=0, no-transform"
	object.ContentType = "application/pkix-crl"

	obj, err := svc.Objects.Insert(bucket, &object).
		Media(reader).Do()

	if err != nil {
		return err
	}

	fmt.Println("Created object " + obj.Id)

	return nil
}

func main() {

	if len(os.Args) != 5 {
		fmt.Println("Usage:")
		fmt.Println("  upload-to-storage <key> <bucket> <sourcefile> <destfile>")
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

	bucket := os.Args[2]

	// Read the content file
	content, err := ioutil.ReadFile(os.Args[3])
	if err != nil {
		fmt.Printf("Couldn't read content file: %s\n",
			err.Error())
		return
	}

	destFile := os.Args[4]

	svc, err := StorageSignin(key)
	if err != nil {
		fmt.Printf("Couldn't connect: %s\n",
			err.Error())
		return
	}

	fmt.Printf("Connected.\n")

	reader := bytes.NewReader(content)

	err = uploadCRL(svc, bucket, destFile, reader)
	if err != nil {
		fmt.Printf("Couldn't upload: %s\n",
			err.Error())
		return
	}

}

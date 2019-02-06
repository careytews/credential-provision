package main

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	cloudkms "google.golang.org/api/cloudkms/v1" // TODO: this is deprecated, should be using:
	// cloudkms "cloud.google.com/go/kms/apiv1"
	storage "google.golang.org/api/storage/v1" // TODO: this is deprecated, should be using:
	// storage "cloud.google.com/go/storage"
)

// Get environment variable
func Getenv(env string, def string) string {
	s := os.Getenv(env)
	if s == "" {
		return def
	} else {
		return s
	}
}

func StorageSignin(key []byte) (*storage.Service, error) {

	// Create JWT from key file
	config, err := google.JWTConfigFromJSON(key)
	if err != nil {
		return nil, errors.New("JWTConfigFromJSON: " + err.Error())
	}

	// Access scope
	config.Scopes = []string{storage.CloudPlatformScope}

	// Create service client.
	client := config.Client(oauth2.NoContext)

	// Connect to Google Storage
	svc, err := storage.New(client)
	if err != nil {
		return nil, errors.New("Create client: " + err.Error())
	}

	return svc, nil

}

func CloudKMSSignin(key []byte) (*cloudkms.Service, error) {

	// Create JWT from key file
	config, err := google.JWTConfigFromJSON(key)
	if err != nil {
		return nil, errors.New("JWTConfigFromJSON: " + err.Error())
	}

	// Access scope
	config.Scopes = []string{cloudkms.CloudPlatformScope}

	// Create service client.
	client := config.Client(oauth2.NoContext)

	// Connect to KMS
	svc, err := cloudkms.New(client)
	if err != nil {
		return nil, errors.New("Create client: " + err.Error())
	}

	return svc, nil

}

func GetGeneration(svc *storage.Service, bucket, path string, generation *int64) error {

	obj, err := svc.Objects.Get(bucket, path).Do()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Couldn't get object: %s\n",
			err.Error())
		return err
	}

	*generation = obj.Generation

	return nil

}

func Download(svc *storage.Service, bucket, path string, writer io.Writer) error {

	resp, err := svc.Objects.Get(bucket, path).Download()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Couldn't get object: %s\n",
			err.Error())
		return err
	}

	io.Copy(writer, resp.Body)
	resp.Body.Close()

	return nil

}

// Upload - Upload item to Google Storage
// Supplying a generation number will only upload the item if the generation number
// matches the one supplied. If there is no match, the upload will fail.
// Provide a negative generation number to upload the item without the generation check
func Upload(svc *storage.Service, user, bucket, path string, reader io.Reader, generation int64) error {

	var object storage.Object
	object.Name = path
	object.Kind = "storage#object"

	var obj *storage.Object
	var err error
	if generation < 0 {
		obj, err = svc.Objects.Insert(bucket, &object).
			Media(reader).Do()
	} else {
		obj, err = svc.Objects.Insert(bucket, &object).IfGenerationMatch(generation).
			Media(reader).Do()
	}

	if err != nil {
		return err
	}

	fmt.Println("Created object " + obj.Id)

	// Ensure user can read their creds
	var ac storage.ObjectAccessControl
	ac.Role = "READER"

	fmt.Println("Set policy...")
	_, err = svc.ObjectAccessControls.Update(bucket, path,
		//		obj.Id,
		"user-"+user, &ac).Do()
	if err != nil {
		return err
	}

	return nil

}

func KeyID(user string) string {
	h := sha256.New()
	h.Write([]byte("qK^45X/X{{]D!fTinC:"))
	h.Write([]byte(user))
	hash := fmt.Sprintf("%x", h.Sum(nil))
	return hash[0:62]
}

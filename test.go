package main

import (
	"os"

	"google.golang.org/api/storage/v1"

	//	"io/ioutil"
	"fmt"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	//	"bytes"

	"io"
	"log"
)

func download(svc *storage.Service, bucket, path string, writer io.Writer) error {

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

func main() {

	// Your credentials should be obtained from the Google
	// Developer Console (https://console.developers.google.com).
	config := &oauth2.Config{
		ClientID:     "1041863416400-bedifa4at0q40nduvi2s6qpum4jqmok5.apps.googleusercontent.com",
		ClientSecret: "pkvQuA9rx0tGIs6IBB2d_3y8",
		RedirectURL:  "http://example.org",
		Scopes: []string{
			storage.CloudPlatformScope,
		},
		Endpoint: google.Endpoint,
	}

	// Redirect user to Google's consent page to ask for permission
	// for the scopes specified above.
	url := config.AuthCodeURL("state")
	fmt.Printf("Visit the URL for the auth dialog: %v\n", url)

	// Handle the exchange code to initiate a transport.
	tok, err := config.Exchange(oauth2.NoContext, "authorization-code")
	if err != nil {
		log.Fatal(err)
	}

	// Access scope
	config.Scopes = []string{storage.CloudPlatformScope}

	// Create service client.
	client := config.Client(oauth2.NoContext, tok)

	// Connect to Google Storage
	svc, err := storage.New(client)
	if err != nil {
		log.Fatal(err)
	}

	bucket := "trust-networks"
	path := "gaffer-sample.json"

	err = Download(svc, bucket, path, os.Stdout)
	if err != nil {
		log.Fatal(err)
	}

}

package main

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"

	"google.golang.org/api/cloudkms/v1"
)

func encrypt(svc *cloudkms.Service, user string, data []byte) error {

	// Get environment variables.
	projectID := Getenv("PROJECT_ID", "")
	keyRing := Getenv("KEY_RING", "")

	cryptoKey := KeyID(user)

	template := "projects/%s/locations/%s/keyRings/%s/cryptoKeys/%s"
	resourceName := fmt.Sprintf(template, projectID, "global", keyRing,
		cryptoKey)

	fmt.Fprintf(os.Stderr, "Create crpyto key...\n")
	resp, err := svc.Projects.Locations.KeyRings.CryptoKeys.
		Encrypt(resourceName, &cloudkms.EncryptRequest{
			Plaintext: base64.StdEncoding.EncodeToString(data),
		}).Do()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Encrypt failed: %s\n", err.Error())
		return nil
	}

	fmt.Fprintln(os.Stderr, "Success.")

	_ = resp

	data, _ = base64.StdEncoding.DecodeString(resp.Ciphertext)

	fmt.Printf("%x", data)
	return nil

}

func main() {

	if len(os.Args) != 4 {
		fmt.Println("Usage:")
		fmt.Println("  encode-key <key> <user> <fkey>")
		os.Exit(1)
	}

	// Get environment variables.
	privatejson := os.Args[1]

	// Read the key file
	private, err := ioutil.ReadFile(privatejson)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Couldn't read key file: %s\n",
			err.Error())
		return
	}

	user := os.Args[2]

	// Read the key file
	key, err := ioutil.ReadFile(os.Args[3])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Couldn't read key file: %s\n",
			err.Error())
		return
	}

	aeskey := make([]byte, len(key)/2)

	_, err = hex.Decode(aeskey, key)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Couldn't decode hex\n",
			err.Error())
		return
	}

	svc, err := CloudKMSSignin(private)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Couldn't connect: %s\n",
			err.Error())
		return
	}

	encrypt(svc, user, aeskey)

}

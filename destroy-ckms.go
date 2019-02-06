package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"google.golang.org/api/cloudkms/v1"
)

func destroyKms(svc *cloudkms.Service, user string) error {

	// Get environment variables.
	projectID := Getenv("PROJECT_ID", "")
	keyRing := Getenv("KEY_RING", "")

	cryptoKey := KeyID(user)

	template := "projects/%s/locations/%s/keyRings/%s"
	resourceName := fmt.Sprintf(template, projectID, "global", keyRing)

	fmt.Printf("Check for keyring %s...\n", keyRing)
	_, err := svc.Projects.Locations.KeyRings.
		Get(resourceName).Do()
	if err != nil {
		fmt.Printf("KeyRing get failed: %s\n",
			err.Error())
		fmt.Printf("Maybe key ring %s does not exist.\n", keyRing)
		return err
	}

	fmt.Println("Key ring exists.")

	template = "projects/%s/locations/%s/keyRings/%s/cryptoKeys/%s"
	resourceName = fmt.Sprintf(template, projectID, "global", keyRing,
		cryptoKey)

	fmt.Println("List crpyto key...")
	res, err := svc.Projects.Locations.KeyRings.CryptoKeys.
		CryptoKeyVersions.
		List(resourceName).Do()
	if err != nil {
		fmt.Printf("CryptoKey create failed: %s\n",
			err.Error())
		return err
	}

	for _, v := range res.CryptoKeyVersions {

		fmt.Println("Delete " + v.Name + "...")

		_, err := svc.Projects.Locations.KeyRings.CryptoKeys.
			CryptoKeyVersions.
			Destroy(v.Name,
				&cloudkms.DestroyCryptoKeyVersionRequest{}).Do()
		if err != nil {
			fmt.Printf("CryptoKey Destroy failed: %s\n",
				err.Error())
		} else {
			fmt.Println("Success.")
		}
	}

	return nil

}

func main() {

	if len(os.Args) != 3 {
		fmt.Println("Usage:")
		fmt.Println("  destroy-ckms <key> <user>")
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

	svc, err := CloudKMSSignin(key)
	if err != nil {
		fmt.Printf("Couldn't connect: %s\n",
			err.Error())
		return
	}

	destroyKms(svc, user)

}

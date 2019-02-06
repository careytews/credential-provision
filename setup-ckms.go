package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"google.golang.org/api/cloudkms/v1"
)

func initialiseKms(svc *cloudkms.Service, user string, isSa bool) error {

	// Get environment variables.
	projectID := Getenv("PROJECT_ID", "")
	keyRing := Getenv("KEY_RING", "")
	serviceAccount := Getenv("SERVICE_ACCOUNT", "")

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

	template = "projects/%s/locations/%s/keyRings/%s"
	resourceName = fmt.Sprintf(template, projectID, "global", keyRing)

	ck := cloudkms.CryptoKey{
		Purpose: "ENCRYPT_DECRYPT",
	}

	fmt.Println("Create crpyto key...")
	_, err = svc.Projects.Locations.KeyRings.CryptoKeys.
		Create(resourceName, &ck).CryptoKeyId(cryptoKey).Do()
	if err != nil {
		fmt.Printf("CryptoKey create failed: %s\n",
			err.Error())
		fmt.Println("Maybe it already exists.")
	}

	// Setup CloudKMS policy.
	// - User can use key to decrypt.
	// - Cred Service Account can use key to encrypt.

	var uString string
	if isSa {
		uString = "serviceAccount:" + user
	} else {
		uString = "user:" + user
	}

	var pol cloudkms.Policy
	pol.Bindings = []*cloudkms.Binding{
		&cloudkms.Binding{
			Members: []string{uString},
			Role:    "roles/cloudkms.cryptoKeyDecrypter",
		},
		&cloudkms.Binding{
			Members: []string{"serviceAccount:" + serviceAccount},
			Role:    "roles/cloudkms.cryptoKeyEncrypter",
		},
	}

	// Policy request
	polreq := cloudkms.SetIamPolicyRequest{
		Policy: &pol,
	}

	// Work out crypto key resource name.
	template = "projects/%s/locations/%s/keyRings/%s/cryptoKeys/%s"
	resourceName = fmt.Sprintf(template, projectID, "global", keyRing,
		cryptoKey)

	fmt.Println("Set IAM policy on crypto key...")
	// Set IAM policy on crypto key.
	_, err = svc.Projects.Locations.KeyRings.CryptoKeys.
		SetIamPolicy(resourceName, &polreq).Do()
	if err != nil {
		fmt.Printf("CryptoKey SetIamPolicy failed: %s\n",
			err.Error())
		return err
	}

	fmt.Println("Success.")

	return nil

}

func main() {

	if len(os.Args) < 3 {
		fmt.Println("Usage:")
		fmt.Println("  setup-ckms <key> <user> [<isSa>]")
		fmt.Println("    isSa=yes|no")
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

	isSa := false
	if len(os.Args) > 3 {
		if os.Args[3] == "yes" {
			isSa = true
		}
	}

	svc, err := CloudKMSSignin(key)
	if err != nil {
		fmt.Printf("Couldn't connect: %s\n",
			err.Error())
		return
	}

	initialiseKms(svc, user, isSa)

}

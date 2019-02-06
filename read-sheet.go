package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/sheets/v4"
)

func driveSignin(key []byte) (*drive.Service, error) {

	// Create JWT from key file
	config, err := google.JWTConfigFromJSON(key)
	if err != nil {
		return nil, errors.New("JWTConfigFromJSON: " + err.Error())
	}

	// Access scope
	config.Scopes = []string{drive.DriveScope}

	// Create service client.
	client := config.Client(oauth2.NoContext)

	// Connect to Google Drive
	svc, err := drive.New(client)
	if err != nil {
		return nil, errors.New("Create client: " + err.Error())
	}

	return svc, nil

}

func sheetsSignin(key []byte) (*sheets.Service, error) {

	// Create JWT from key file
	config, err := google.JWTConfigFromJSON(key)
	if err != nil {
		return nil, errors.New("JWTConfigFromJSON: " + err.Error())
	}

	// Access scope
	config.Scopes = []string{sheets.SpreadsheetsScope}

	// Create service client.
	client := config.Client(oauth2.NoContext)

	// Connect to Google Drive
	svc, err := sheets.New(client)
	if err != nil {
		return nil, errors.New("Create client: " + err.Error())
	}

	return svc, nil

}

func get(svc *drive.Service, name string) (string, error) {

	list, err := svc.Files.List().PageSize(10).
		Fields("nextPageToken, files(id, name)").
		IncludeTeamDriveItems(true).
		SupportsTeamDrives(true).
		Q("name = '" + name + "'").Do()
	if err != nil {
		return "", err
	}

	if len(list.Files) == 0 {
		return "", nil
	}

	return list.Files[0].Id, nil

}

func checkSheet(dSvc *drive.Service, sSvc *sheets.Service, name string) error {

	id, err := get(dSvc, name)
	if err != nil {
		return err
	}

	if id == "" {
		return errors.New("File not found in drive")
	}

	readRange := "A2:D"

	resp, err := sSvc.Spreadsheets.Values.Get(id, readRange).Do()
	if err != nil {
		return err
	}

	if len(resp.Values) < 0 {
		return nil
	}

	for rownum, row := range resp.Values {

		// Need at least 3 columns worth to be interesting.
		if len(row) <= 3 {
			continue
		}

		application := row[0].(string)
		user := row[1].(string)
		identity := row[2].(string)
		action := row[3].(string)

		if action == "create" && application == "vpn" {

			fmt.Println()
			fmt.Println("---- Creating VPN key for " + identity)

			out, err := exec.Command("./create-vpn-key", user, identity).
				Output()

			if err != nil {
				fmt.Println("Error: %s\n", err.Error())
			}

			fmt.Printf("%s", out)

			if err == nil {

				rng := "D" + strconv.Itoa(rownum+2)

				vr := &sheets.ValueRange{
					Values: [][]interface{}{
						[]interface{}{
							"",
							"created",
							out,
						},
					},
				}

				_, err =
					sSvc.Spreadsheets.Values.Update(id,
						rng,
						vr).
						ValueInputOption("USER_ENTERED").
						Do()
				if err != nil {
					fmt.Println("Update error " + err.Error())
				}

			}

		}

		if action == "create" && application == "web" {

			fmt.Println()
			fmt.Println("---- Creating web key for " + identity)

			out, err := exec.Command("./create-web-key", user, identity).
				Output()

			if err != nil {
				fmt.Println("Error: %s\n", err.Error())
			}

			fmt.Printf("%s", out)

			if err == nil {

				rng := "D" + strconv.Itoa(rownum+2)

				vr := &sheets.ValueRange{
					Values: [][]interface{}{
						[]interface{}{
							"",
							"created",
							out,
						},
					},
				}

				_, err =
					sSvc.Spreadsheets.Values.Update(id,
						rng,
						vr).
						ValueInputOption("USER_ENTERED").
						Do()
				if err != nil {
					fmt.Println("Update error " + err.Error())
				}

			}

		}

	}

	return nil

}

func main() {

	name := Getenv("SHEET_FILENAME", "Trust Networks SaaS Accounts")

	// Get environment variables.
	keyfile := Getenv("KEY", "private.json")

	// Read the key file
	key, err := ioutil.ReadFile(keyfile)
	if err != nil {
		fmt.Printf("Couldn't read key file: %s\n",
			err.Error())
		return
	}

	dSvc, err := driveSignin(key)
	if err != nil {
		fmt.Printf("Couldn't connect: %s\n",
			err.Error())
		return
	}

	sSvc, err := sheetsSignin(key)
	if err != nil {
		fmt.Printf("Couldn't connect: %s\n",
			err.Error())
		return
	}

	fmt.Printf("Connected.\n")

	for {

		err = checkSheet(dSvc, sSvc, name)
		if err != nil {
			fmt.Printf("Error: %s\n", err.Error())
			os.Exit(1)
		}

		time.Sleep(30 * time.Second)

	}

}

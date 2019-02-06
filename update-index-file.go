/*********************************************************************************************

	update-index.go

	Write data to storage avoiding potential race-condition using Google's
	if-generation-match checks.

	NOTE: this code is specifically for Google Storage. Porting required if using with
	other cloud storage.

*********************************************************************************************/

package main

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io/ioutil"
	"math"
	"math/big"
	"os"
	"strings"
	"time"
)

// Multiple request retries
// Cloud Storage is a distributed system. Because requests can fail due to network or service conditions,
// Google recommends that you retry failures with exponential backoff. However, due to the nature of
// distributed systems, sometimes these retries can cause surprising behavior.
// https://cloud.google.com/storage/docs/exponential-backoff

func backoff(i float64) time.Duration {
	// Use crypt random to ensure source is good (not needed for security,
	// but better than seeding on time, which wouldn't avoid the clash we're using the backoff for...)
	randMilliSecs, _ := rand.Int(rand.Reader, big.NewInt(1000))
	// waitTime in milliseconds for precision
	// Add some jitter to avoid clashes with other backed-off attempts.
	waitTime := (time.Duration(math.Pow(2, i)) * time.Second) +
		(time.Duration(randMilliSecs.Int64()) * time.Millisecond)
	time.Sleep(waitTime)
	return waitTime
}

func main() {
	// Parse arguments
	if len(os.Args) != 6 {
		fmt.Fprintln(os.Stderr, "Usage:")
		fmt.Fprintln(os.Stderr,
			"  update-index <key> <user> <line to remove> <replacement line> <index file>")
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

	lineToRemove := os.Args[3]
	replacementLine := os.Args[4]
	indexFile := os.Args[5]

	// Download data to edit
	svc, err := StorageSignin(key)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Couldn't connect: %s\n",
			err.Error())
		return
	}

	fmt.Fprintf(os.Stderr, "Connected.\n")

	bucket := Getenv("BUCKET", "")
	path := user + "/" + indexFile

	// Download data, edit data, upload new data.
	// Re-try with exponential backoff

	// Backoff set-up
	elapsedTime := time.Duration(0)
	backoffTime := time.Duration(32) * time.Second
	i := float64(0)
	for elapsedTime < backoffTime {
		// Whilst within re-try period...

		// Get generation info
		var generation int64
		err = GetGeneration(svc, bucket, path, &generation)

		// Get data to edit
		var downloadedData bytes.Buffer // TODO: is this the correct writer?
		err = Download(svc, bucket, path, &downloadedData)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Couldn't download: %s\n",
				err.Error())
			return
		}

		// Edit data
		downloadedLines := strings.Split(downloadedData.String(), "\n")
		var content string
		// Skip any lines (i.e. keys) that match the device for which we are updating
		for _, line := range downloadedLines {
			if !strings.Contains(line, lineToRemove) && line != "" {
				content += line
				content += "\n"
			}
		}
		// Note: replacementLine must be added as a single line

		if replacementLine != "" {
			content += replacementLine
		}

		// Attempt to upload updated data
		// (iff we haven't lost race, otherwise re-try with exponential backoff)
		// Upload adds ifGenerationMatch flag

		reader := bytes.NewReader([]byte(content))

		err = Upload(svc, user, bucket, path, reader, generation)
		if err != nil {
			fmt.Printf("Couldn't upload: %s\n",
				err.Error())
			// 412 is generation mis-match so we'll re-try, otherwise we'll give up immediately
			if !strings.Contains(string(err.Error()), "412") {
				break
			}
		} else {
			// Success - stop trying
			break
		}

		// Upload failed, backoff
		elapsedTime += backoff(i)
		i += 1.0
	}
}

package main

import (
	"crypto/rand"
	"fmt"
	"os"
)

const key_len = 256 / 8

func main() {

	var b [key_len]byte
	_, err := rand.Read(b[:])

	if err != nil {
		fmt.Printf("Random read: %s\n", err.Error())
		os.Exit(1)
	}

	fmt.Printf("%x", b)

}
